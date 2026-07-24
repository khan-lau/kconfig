package kconf

import "os"

// ============================================================================
// Processor 接口 — 反序列化后的处理/加工接口
// ============================================================================

// Processor 配置解析完成后的处理接口。
// 可用于设置默认值、校验、类型转换（mapstructure 解码）等场景。
type Processor interface {
	Process(v any) error
}

// ProcessFunc 将普通函数适配为 Processor 接口。
type ProcessFunc func(v any) error

func (f ProcessFunc) Process(v any) error {
	return f(v)
}

// ============================================================================
// Option 函数选项
// ============================================================================

// Option 是 Read / ReadTo 的可选参数。
type Option func(*options)

type options struct {
	processor Processor
}

func defaultOptions() options {
	return options{}
}

// WithProcessor 设置一个 Processor，在反序列化完成后调用。
// processor.Process 接收的 v 与 Read/ReadTo 返回的值是同一个对象，
// 可以直接修改它。
//
//	Read("config.yaml", WithProcessor(ProcessFunc(func(v any) error {
//	    m := v.(map[string]any)
//	    if m["port"] == nil { m["port"] = 8080 }
//	    return nil
//	})))
func WithProcessor(p Processor) Option {
	return func(o *options) {
		o.processor = p
	}
}

// ============================================================================
// 核心 API
// ============================================================================

// Read 读取配置文件并以 map[string]any 的形式返回。
// 配置内容不限定结构，适用于"配置内容不固定"的场景。
//
//	cfg, err := kconf.Read("config.yaml")
//	host := cfg["host"].(string)
//
//	// 带后处理
//	cfg, err := kconf.Read("config.yaml",
//	    kconf.WithProcessor(myProcessor))
func Read(filePath string, opts ...Option) (map[string]any, error) {
	o := defaultOptions()
	for _, opt := range opts {
		opt(&o)
	}

	data, fn, err := readAndDecoder(filePath)
	if err != nil {
		return nil, err
	}

	var v map[string]any
	if err = fn(data, &v); err != nil {
		return nil, err
	}

	if o.processor != nil {
		if err = o.processor.Process(v); err != nil {
			return nil, err
		}
	}
	return v, nil
}

// ReadTo 读取配置文件并解码到指定的目标结构体中。
// 适合已知配置结构的场景。target 必须是一个指针。
//
//	var cfg MyConfig
//	err := kconf.ReadTo("config.yaml", &cfg)
//
//	// 带后处理
//	err := kconf.ReadTo("config.yaml", &cfg,
//	    kconf.WithProcessor(myProcessor))
func ReadTo(filePath string, target any, opts ...Option) error {
	o := defaultOptions()
	for _, opt := range opts {
		opt(&o)
	}

	data, fn, err := readAndDecoder(filePath)
	if err != nil {
		return err
	}

	if err = fn(data, target); err != nil {
		return err
	}

	if o.processor != nil {
		return o.processor.Process(target)
	}
	return nil
}

// ReadRaw 读取配置文件并以 []byte 的形式返回原始内容。
func ReadRaw(filePath string) ([]byte, error) {
	return os.ReadFile(filePath)
}

// Save 将配置写回文件。格式由文件后缀自动识别。
//
//	cfg, _ := kconf.Read("config.yaml")
//	cfg["port"] = 9090
//	kconf.Save("config.yaml", cfg)
//
//	// struct 也一样
//	kconf.Save("config.toml", &appCfg)
func Save(filePath string, v any) error {
	_, fn, err := marshaler(filePath)
	if err != nil {
		return err
	}
	data, err := fn(v)
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, data, 0644)
}

// readAndDecoder 读取文件内容并根据文件后缀名(扩展名)选择对应的解码器。
// 返回文件原始字节、解码器函数、错误。

func readAndDecoder(filePath string) (data []byte, fn UnmarshalFunc, err error) {
	data, err = os.ReadFile(filePath)
	if err != nil {
		return nil, nil, err
	}

	_, fn, err = decoder(filePath)
	if err != nil {
		return nil, nil, err
	}
	return data, fn, nil
}
