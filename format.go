// Package kconf 是一个通用的多格式配置文件读取库。
//
// 核心能力：
//   - 自动识别文件格式（json / json5 / jsonc / yaml / toml / hcl）
//   - 支持注册自定义格式
//   - 读取到 map[string]any（自由结构）
//   - 读取到任意结构体
//   - 纯数据读取，不做任何业务假设
package kconf

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"sync"

	toml "github.com/BurntSushi/toml"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/tidwall/jsonc"
	json5 "github.com/titanous/json5"
	"github.com/zclconf/go-cty/cty"
	yaml3 "gopkg.in/yaml.v3"
)

// ---------------------------------------------------------------------------
// 解码器
// ---------------------------------------------------------------------------

// UnmarshalFunc 反序列化函数。签名与 encoding/json.Unmarshal 一致。
type UnmarshalFunc func(data []byte, v any) error

var (
	builtin = map[string]UnmarshalFunc{} // 内置格式反序列化函数
	mu      sync.RWMutex
	custom  map[string]UnmarshalFunc // 自定义格式反序列化函数
)

func init() {
	builtin[".json"] = json.Unmarshal
	builtin[".json5"] = json5.Unmarshal
	builtin[".jsonc"] = jsoncUnmarshal // JSON with comments
	builtin[".yaml"] = yaml3.Unmarshal
	builtin[".yml"] = yaml3.Unmarshal
	builtin[".toml"] = toml.Unmarshal
	builtin[".hcl"] = hclUnmarshal // HashiCorp Configuration Language
}

// jsoncUnmarshal 处理 JSONC（JSON with Comments）：
// 先剥离注释和尾部逗号，再用标准 json 解析。
func jsoncUnmarshal(data []byte, v any) error {
	cleaned := jsonc.ToJSON(data)
	return json.Unmarshal(cleaned, v)
}

// hclUnmarshal 处理 HCL（HashiCorp Configuration Language）。
//
// HCL 是类型化语言，动态解码有限制：
//   - ReadTo() 目标为 struct → 使用 gohcl.Decode，完美支持
//   - Read() 目标为 map → 仅支持纯属性（无 block）的简单 HCL
//     如果包含 block，gohcl.Decode 会尝试解码到 map，这只在 struct 下工作
//     所以我们回退到先解码到动态 map
//
// 实现策略：
//   - 先用 hclparse.ParseHCL 解析
//   - 如果目标是 struct，直接 gohcl.Decode
//   - 如果目标是 map，先转 JSON 中间格式
func hclUnmarshal(data []byte, v any) error {
	f, diags := hclparse.NewParser().ParseHCL(data, "config.hcl")
	if diags.HasErrors() {
		return diags
	}

	// 判断目标是否为 struct 指针
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Pointer && !rv.IsNil() && rv.Elem().Kind() == reflect.Struct {
		diags = gohcl.DecodeBody(f.Body, nil, v)
		if diags.HasErrors() {
			return diags
		}
		return nil
	}

	// map 类型：直接赋值，跳过 JSON 中间格式，避免类型信息损失
	attrs, diags := f.Body.JustAttributes()
	if diags.HasErrors() {
		return fmt.Errorf("hcl: %v\n  tip: 配置包含 block 时请使用 ReadTo() 配合 struct", diags.Error())
	}
	m := make(map[string]any, len(attrs))
	for name, attr := range attrs {
		val, valDiags := attr.Expr.Value(nil)
		if valDiags.HasErrors() {
			return valDiags
		}
		m[name] = ctyToGo(val)
	}

	// v 是 *map[string]any（由 Read 传入），直接赋值
	if ptr, ok := v.(*map[string]any); ok {
		*ptr = m
		return nil
	}
	// fallback：其他类型走 JSON 转换
	jsonData, _ := json.Marshal(m)
	return json.Unmarshal(jsonData, v)
}

// ---------------------------------------------------------------------------
// 编码器（写回文件）
// ---------------------------------------------------------------------------

// MarshalFunc 序列化函数。签名与 encoding/json.Marshal 一致。
type MarshalFunc func(v any) ([]byte, error)

var (
	builtinMarshal = map[string]MarshalFunc{}
	marshalMu      sync.RWMutex
	customMarshal  map[string]MarshalFunc
)

func init() {
	// json / json5 / jsonc 都复用 json 序列化（json5/jsonc 向前兼容 json）
	builtinMarshal[".json"] = marshalJSON
	builtinMarshal[".json5"] = marshalJSON
	builtinMarshal[".jsonc"] = marshalJSON
	builtinMarshal[".yaml"] = marshalYAML
	builtinMarshal[".yml"] = marshalYAML
	builtinMarshal[".toml"] = marshalTOML
	// HCL 不支持序列化写回
}

func marshalJSON(v any) ([]byte, error) {
	return json.MarshalIndent(v, "", "  ")
}

func marshalYAML(v any) ([]byte, error) {
	return yaml3.Marshal(v)
}

func marshalTOML(v any) ([]byte, error) {
	var buf bytes.Buffer
	err := toml.NewEncoder(&buf).Encode(v)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// RegisterMarshal 注册自定义文件后缀对应的序列化函数。
func RegisterMarshal(ext string, fn MarshalFunc) {
	marshalMu.Lock()
	defer marshalMu.Unlock()
	if customMarshal == nil {
		customMarshal = make(map[string]MarshalFunc)
	}
	customMarshal[strings.ToLower(ext)] = fn
}

// marshaler 根据文件路径查找对应的序列化函数。
func marshaler(filePath string) (ext string, fn MarshalFunc, err error) {
	ext = strings.ToLower(filepath.Ext(filePath))
	if ext == "" {
		return ext, nil, fmt.Errorf("kconf: '%s' missing file extension", filePath)
	}

	marshalMu.RLock()
	fn, ok := customMarshal[ext]
	marshalMu.RUnlock()
	if ok {
		return ext, fn, nil
	}

	fn, ok = builtinMarshal[ext]
	if !ok {
		if ext == ".hcl" {
			return ext, nil, fmt.Errorf("kconf: HCL format does not support Save/写回 (no universal serialization scheme; use ReadTo with struct instead)")
		}
		return ext, nil, fmt.Errorf("kconf: unsupported format for marshal '%s' (supported: json, json5, jsonc, yaml, toml)", ext)
	}
	return ext, fn, nil
}

// ctyToGo 将 cty.Value 转换为 Go 原生类型（json.Marshal 兼容）。
func ctyToGo(val cty.Value) any {
	if val.IsNull() {
		return nil
	}

	ty := val.Type()
	switch {
	case ty == cty.String:
		return val.AsString()
	case ty == cty.Number:
		f := val.AsBigFloat()
		i, acc := f.Int64()
		if acc == 0 {
			return i
		}
		v, _ := f.Float64()
		return v
	case ty == cty.Bool:
		return val.True()
	case ty.IsListType() || ty.IsTupleType():
		items := val.AsValueSlice()
		result := make([]any, 0, len(items))
		for _, item := range items {
			result = append(result, ctyToGo(item))
		}
		return result
	case ty.IsMapType() || ty.IsObjectType():
		items := val.AsValueMap()
		result := make(map[string]any, len(items))
		for k, v := range items {
			result[k] = ctyToGo(v)
		}
		return result
	case ty == cty.DynamicPseudoType:
		return ctyToGo(val)
	default:
		return val.AsString()
	}
}

// ---------------------------------------------------------------------------
// 公开 API
// ---------------------------------------------------------------------------

// Register 注册自定义文件后缀对应的反序列化函数。
// 注册后会覆盖后缀相同的内置处理器。
//
//	ext: 文件后缀，例如 ".properties" / ".conf"
//	fn:  反序列化函数
func Register(ext string, fn UnmarshalFunc) {
	mu.Lock()
	defer mu.Unlock()
	if custom == nil {
		custom = make(map[string]UnmarshalFunc)
	}
	custom[strings.ToLower(ext)] = fn
}

// decoder 根据文件路径查找对应的反序列化函数。
func decoder(filePath string) (ext string, fn UnmarshalFunc, err error) {
	ext = strings.ToLower(filepath.Ext(filePath))
	if ext == "" {
		return ext, nil, fmt.Errorf("kconf: '%s' missing file extension", filePath)
	}

	mu.RLock()
	fn, ok := custom[ext]
	mu.RUnlock()
	if ok {
		return ext, fn, nil
	}

	fn, ok = builtin[ext]
	if !ok {
		return ext, nil, fmt.Errorf("kconf: unsupported format '%s' (supported: json, json5, jsonc, yaml, toml, hcl)", ext)
	}
	return ext, fn, nil
}

// Detect 返回文件使用的配置格式名称（"json"、"json5"、"yaml"、"toml" 等）。
func Detect(filePath string) (string, error) {
	ext, _, err := decoder(filePath)
	if err != nil {
		return "", err
	}
	return ext[1:], nil // 去掉前导 "."
}
