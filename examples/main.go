package main

import (
	"fmt"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
)

// ============================================================================
// main — 使用 ConfigInstance 从 conf.json5 读取完整配置
// ============================================================================

func main() {
	// 定位到 examples 目录下的 conf.json5
	_, caller, _, _ := runtime.Caller(0)
	dir := filepath.Dir(caller)
	filePath := filepath.Join(dir, "conf.json5")

	cfg, err := ConfigInstance(filePath)
	if err != nil {
		panic(err)
	}

	// ========================================================================
	// 输出关键配置项
	// ========================================================================
	fmt.Println("=== ConfigInstance 加载成功 ===")
	fmt.Printf("FileCache     = %s\n", cfg.FileCache)
	fmt.Printf("SyncTime      = %dms\n", cfg.SyncTime)
	fmt.Printf("SyncFile      = %s\n", cfg.SyncFile)
	fmt.Printf("PointMap      = %s\n", cfg.PointMap)
	fmt.Printf("QueueSize     = %d\n", cfg.QueueSize)
	fmt.Printf("SendInterval  = %dms\n", cfg.SendInterval)

	// Log
	if cfg.Log != nil {
		fmt.Println("--- Log ---")
		fmt.Printf("  LogLevel      = %d\n", cfg.Log.LogLevel)
		fmt.Printf("  Colorful      = %t\n", cfg.Log.Colorful)
		fmt.Printf("  MaxAge        = %dh\n", cfg.Log.MaxAge)
		fmt.Printf("  MaxSize       = %d\n", cfg.Log.MaxSize)
		fmt.Printf("  RotationTime  = %dh\n", cfg.Log.RotationTime)
		fmt.Printf("  Console       = %t\n", cfg.Log.Console)
		fmt.Printf("  Async         = %t\n", cfg.Log.Async)
		fmt.Printf("  FlushInterval = %dms\n", cfg.Log.FlushInterval)
		fmt.Printf("  BufferSize    = %d\n", cfg.Log.BufferSize)
		fmt.Printf("  LogDir        = %s\n", cfg.Log.LogDir)
	}

	// WebRPC
	if cfg.WebRPC != nil {
		fmt.Println("--- WebRPC ---")
		fmt.Printf("  Host           = %s\n", cfg.WebRPC.Host)
		fmt.Printf("  Port           = %d\n", cfg.WebRPC.Port)
		fmt.Printf("  ReloadCooldown = %dmin\n", cfg.WebRPC.ReloadCooldown)
	}

	// Database
	fmt.Println("--- Database ---")
	for _, db := range cfg.Database {
		if db != nil {
			fmt.Printf("  [%s] type=%s\n", db.Tag, db.DBType)
		}
	}
	if len(cfg.Database) == 0 {
		fmt.Println("  (none)")
	}

	// Source
	fmt.Println("--- Source ---")
	for _, s := range cfg.Source {
		if s != nil {
			printMQItem(s)
		}
	}

	// Target
	fmt.Println("--- Target ---")
	for _, t := range cfg.Target {
		if t != nil {
			printMQItem(t)
		}
	}

	fmt.Println()
	fmt.Println("所有配置项均已正确解析, any 包裹的 MQ 配置已自动转为对应类型")
}

// printMQItem 打印 MQItemObj 的详细配置（反射实现，自动遍历所有字段）
func printMQItem(item *MQItemObj) {
	fmt.Printf("  [%s] type=%s compress=%t\n", item.Tag, item.MQType, item.IsCompress)
	if item.Item == nil {
		fmt.Println("    item: nil")
		return
	}
	printStruct("    ", reflect.ValueOf(item.Item), 0)
}

// printStruct 利用反射库 递归打印结构体的所有字段
func printStruct(prefix string, val reflect.Value, depth int) {
	if depth > 4 {
		fmt.Printf("%s...\n", prefix)
		return
	}

	// 解引用指针
	for val.Kind() == reflect.Pointer {
		if val.IsNil() {
			fmt.Printf("%snil\n", prefix)
			return
		}
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		fmt.Printf("%s%v\n", prefix, val.Interface())
		return
	}

	typ := val.Type()
	// 打印类型名作为标题
	fmt.Printf("%s%s:\n", prefix, typ.Name())
	inner := prefix + "  "

	for i := range val.NumField() {
		fv := val.Field(i)
		ft := typ.Field(i)

		// 跳过未导出的字段
		if !ft.IsExported() {
			continue
		}

		name := ft.Name

		// 解引用指针
		fvKind := fv.Kind()
		for fvKind == reflect.Pointer {
			if fv.IsNil() {
				fmt.Printf("%s%s = nil\n", inner, name)
				goto nextField
			}
			fv = fv.Elem()
			fvKind = fv.Kind()
		}

		switch fvKind {
		case reflect.Struct:
			printStruct(inner, fv, depth+1)
		case reflect.Slice:
			if fv.Type().Elem().Kind() == reflect.Struct || fv.Type().Elem().Kind() == reflect.Ptr {
				// 结构体/指针切片
				fmt.Printf("%s%s:\n", inner, name)
				for j := range fv.Len() {
					sv := fv.Index(j)
					printStruct(inner+"  ", sv, depth+1)
				}
				if fv.Len() == 0 {
					fmt.Printf("%s  (empty)\n", inner)
				}
			} else {
				fmt.Printf("%s%s = %v\n", inner, name, fv.Interface())
			}
		case reflect.Map:
			fmt.Printf("%s%s = %v\n", inner, name, fv.Interface())
		default:
			valStr := fmt.Sprintf("%v", fv.Interface())
			// 密码字段脱敏
			if strings.Contains(strings.ToLower(name), "password") {
				valStr = maskPwd(valStr)
			}
			fmt.Printf("%s%s = %s\n", inner, name, valStr)
		}

	nextField:
	}
}

func maskPwd(s string) string {
	if len(s) == 0 {
		return "(empty)"
	}
	if len(s) <= 4 {
		return "****"
	}
	return s[:2] + "****" + s[len(s)-2:]
}
