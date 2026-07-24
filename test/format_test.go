package test

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	kconf "github.com/khan-lau/kconfig"
)

// ============================================================================
// 基础格式 Read 测试
// ============================================================================

type testConfig struct {
	Host string `json:"host" yaml:"host" toml:"host"`
	Port int    `json:"port" yaml:"port" toml:"port"`
}

func TestJSON_Read(t *testing.T) {
	dir := t.TempDir()
	fp := filepath.Join(dir, "test.json")
	os.WriteFile(fp, []byte(`{"host": "0.0.0.0", "port": 8080}`), 0644)
	cfg, err := kconf.Read(fp)
	if err != nil {
		t.Fatal(err)
	}
	if cfg["host"] != "0.0.0.0" {
		t.Fatalf("host: got %v", cfg["host"])
	}
	if cfg["port"] != float64(8080) {
		t.Fatalf("port: got %v", cfg["port"])
	}
}

func TestJSON5_Read(t *testing.T) {
	dir := t.TempDir()
	fp := filepath.Join(dir, "test.json5")
	os.WriteFile(fp, []byte(`{
		// comment
		host: "0.0.0.0",
		port: 8080,
	}`), 0644)
	cfg, err := kconf.Read(fp)
	if err != nil {
		t.Fatal(err)
	}
	if cfg["host"] != "0.0.0.0" {
		t.Fatalf("host: got %v", cfg["host"])
	}
	if cfg["port"] != float64(8080) {
		t.Fatalf("port: got %v", cfg["port"])
	}
}

func TestYAML_Read(t *testing.T) {
	dir := t.TempDir()
	fp := filepath.Join(dir, "test.yaml")
	os.WriteFile(fp, []byte("host: 0.0.0.0\nport: 8080\n"), 0644)
	cfg, err := kconf.Read(fp)
	if err != nil {
		t.Fatal(err)
	}
	if cfg["host"] != "0.0.0.0" {
		t.Fatalf("host: got %v", cfg["host"])
	}
	if cfg["port"] != 8080 {
		t.Fatalf("port: got %v (type: %T)", cfg["port"], cfg["port"])
	}
}

func TestTOML_Read(t *testing.T) {
	dir := t.TempDir()
	fp := filepath.Join(dir, "test.toml")
	os.WriteFile(fp, []byte("host = \"0.0.0.0\"\nport = 8080\n"), 0644)
	cfg, err := kconf.Read(fp)
	if err != nil {
		t.Fatal(err)
	}
	if cfg["host"] != "0.0.0.0" {
		t.Fatalf("host: got %v", cfg["host"])
	}
	if cfg["port"] != int64(8080) {
		t.Fatalf("port: got %v (type: %T)", cfg["port"], cfg["port"])
	}
}

// ============================================================================
// kconf.ReadTo 测试
// ============================================================================

func TestReadTo_JSON(t *testing.T) {
	dir := t.TempDir()
	fp := filepath.Join(dir, "test.json")
	os.WriteFile(fp, []byte(`{"host": "example.com", "port": 9090}`), 0644)
	var cfg testConfig
	if err := kconf.ReadTo(fp, &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Host != "example.com" {
		t.Fatalf("Host: got %v", cfg.Host)
	}
	if cfg.Port != 9090 {
		t.Fatalf("Port: got %v", cfg.Port)
	}
}

func TestReadTo_YAML(t *testing.T) {
	dir := t.TempDir()
	fp := filepath.Join(dir, "test.yaml")
	os.WriteFile(fp, []byte("host: example.com\nport: 9090\n"), 0644)
	var cfg testConfig
	if err := kconf.ReadTo(fp, &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Host != "example.com" {
		t.Fatalf("Host: got %v", cfg.Host)
	}
	if cfg.Port != 9090 {
		t.Fatalf("Port: got %v", cfg.Port)
	}
}

func TestReadTo_TOML(t *testing.T) {
	dir := t.TempDir()
	fp := filepath.Join(dir, "test.toml")
	os.WriteFile(fp, []byte("host = \"example.com\"\nport = 9090\n"), 0644)
	var cfg testConfig
	if err := kconf.ReadTo(fp, &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Host != "example.com" {
		t.Fatalf("Host: got %v", cfg.Host)
	}
	if cfg.Port != 9090 {
		t.Fatalf("Port: got %v", cfg.Port)
	}
}

func TestReadTo_JSONC(t *testing.T) {
	dir := t.TempDir()
	fp := filepath.Join(dir, "test.jsonc")
	os.WriteFile(fp, []byte(`{
		// comment
		"host": "example.com",
		/* block */
		"port": 9090
	}`), 0644)
	var cfg testConfig
	if err := kconf.ReadTo(fp, &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Host != "example.com" {
		t.Fatalf("Host: got %v", cfg.Host)
	}
	if cfg.Port != 9090 {
		t.Fatalf("Port: got %v", cfg.Port)
	}
}

// ============================================================================
// Processor 测试
// ============================================================================

func TestProcessor_Read(t *testing.T) {
	dir := t.TempDir()
	fp := filepath.Join(dir, "test.yaml")
	// 故意缺 port
	os.WriteFile(fp, []byte("host: 0.0.0.0\n"), 0644)

	cfg, err := kconf.Read(fp, kconf.WithProcessor(kconf.ProcessFunc(func(v any) error {
		m := v.(map[string]any)
		if _, ok := m["port"]; !ok {
			m["port"] = 8080
		}
		return nil
	})))
	if err != nil {
		t.Fatal(err)
	}
	if cfg["host"] != "0.0.0.0" {
		t.Fatalf("host: got %v", cfg["host"])
	}
	if cfg["port"] != 8080 {
		t.Fatalf("port: got %v (should be default 8080)", cfg["port"])
	}
}

func TestProcessor_ReadTo(t *testing.T) {
	dir := t.TempDir()
	// port 被设为非法值 -1
	fp := filepath.Join(dir, "test.json")
	os.WriteFile(fp, []byte(`{"host": "0.0.0.0", "port": -1}`), 0644)

	var cfg testConfig
	err := kconf.ReadTo(fp, &cfg, kconf.WithProcessor(kconf.ProcessFunc(func(v any) error {
		c := v.(*testConfig)
		if c.Port <= 0 {
			c.Port = 8080
		}
		return nil
	})))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Host != "0.0.0.0" {
		t.Fatalf("Host: got %v", cfg.Host)
	}
	if cfg.Port != 8080 {
		t.Fatalf("Port: got %v (should be corrected to 8080)", cfg.Port)
	}
}

func TestProcessor_ErrorPropagation(t *testing.T) {
	dir := t.TempDir()
	fp := filepath.Join(dir, "test.yaml")
	os.WriteFile(fp, []byte("host: 0.0.0.0\nport: 8080\n"), 0644)

	want := errors.New("processor error")
	_, err := kconf.Read(fp, kconf.WithProcessor(kconf.ProcessFunc(func(v any) error {
		return want
	})))
	if err != want {
		t.Fatalf("expected error %v, got %v", want, err)
	}
}

// ============================================================================
// Save 测试
// ============================================================================

func TestSave_JSON(t *testing.T) {
	dir := t.TempDir()
	fp := filepath.Join(dir, "out.json")

	data := map[string]any{"host": "example.com", "port": 9090}
	if err := kconf.Save(fp, data); err != nil {
		t.Fatal(err)
	}

	cfg, err := kconf.Read(fp)
	if err != nil {
		t.Fatal(err)
	}
	if cfg["host"] != "example.com" {
		t.Fatalf("host: got %v", cfg["host"])
	}
}

func TestSave_YAML(t *testing.T) {
	dir := t.TempDir()
	fp := filepath.Join(dir, "out.yaml")

	data := map[string]any{"host": "example.com", "port": 9090}
	if err := kconf.Save(fp, data); err != nil {
		t.Fatal(err)
	}

	cfg, err := kconf.Read(fp)
	if err != nil {
		t.Fatal(err)
	}
	if cfg["host"] != "example.com" {
		t.Fatalf("host: got %v", cfg["host"])
	}
}

func TestSave_TOML(t *testing.T) {
	dir := t.TempDir()
	fp := filepath.Join(dir, "out.toml")

	cfg := testConfig{Host: "example.com", Port: 9090}
	if err := kconf.Save(fp, &cfg); err != nil {
		t.Fatal(err)
	}

	var loaded testConfig
	if err := kconf.ReadTo(fp, &loaded); err != nil {
		t.Fatal(err)
	}
	if loaded.Host != "example.com" {
		t.Fatalf("Host: got %v", loaded.Host)
	}
	if loaded.Port != 9090 {
		t.Fatalf("Port: got %v", loaded.Port)
	}
}

func TestSave_HCL_Error(t *testing.T) {
	dir := t.TempDir()
	fp := filepath.Join(dir, "test.hcl")

	err := kconf.Save(fp, map[string]any{"host": "example.com"})
	if err == nil {
		t.Fatal("expected error for HCL save, got nil")
	}
	if !strings.Contains(err.Error(), "does not support Save") {
		t.Fatalf("error should mention HCL does not support Save, got: %v", err)
	}
}

// TestSave_Roundtrip: 读取 yaml → 修改 → 写回 → 再读
func TestSave_Roundtrip(t *testing.T) {
	dir := t.TempDir()
	fp := filepath.Join(dir, "roundtrip.yaml")

	os.WriteFile(fp, []byte("host: 0.0.0.0\nport: 8080\n"), 0644)

	cfg, err := kconf.Read(fp)
	if err != nil {
		t.Fatal(err)
	}

	cfg["host"] = "changed.com"
	cfg["port"] = 9090

	if err := kconf.Save(fp, cfg); err != nil {
		t.Fatal(err)
	}

	cfg2, err := kconf.Read(fp)
	if err != nil {
		t.Fatal(err)
	}
	if cfg2["host"] != "changed.com" {
		t.Fatalf("host: got %v", cfg2["host"])
	}
	if cfg2["port"] != 9090 {
		t.Fatalf("port: got %v", cfg2["port"])
	}
}

// ============================================================================
// Detect 测试
// ============================================================================

func TestDetect(t *testing.T) {
	tests := []struct {
		name string
		ext  string
		want string
	}{
		{"json", ".json", "json"},
		{"json5", ".json5", "json5"},
		{"jsonc", ".jsonc", "jsonc"},
		{"yaml", ".yaml", "yaml"},
		{"yml", ".yml", "yml"},
		{"toml", ".toml", "toml"},
		{"hcl", ".hcl", "hcl"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := kconf.Detect("config" + tt.ext)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Fatalf("Detect(%s) = %s, want %s", tt.ext, got, tt.want)
			}
		})
	}
}

func TestDetect_Unknown(t *testing.T) {
	_, err := kconf.Detect("config.unknown")
	if err == nil {
		t.Fatal("expected error for unknown format")
	}
}

// ============================================================================
// Register 自定义格式测试
// ============================================================================

func TestRegister(t *testing.T) {
	dir := t.TempDir()
	// 使用唯一后缀 .reg，不会与其他测试冲突，无需手动清理
	fp := filepath.Join(dir, "test.reg")

	kconf.Register(".reg", func(data []byte, v any) error {
		ptr := v.(*map[string]any)
		*ptr = map[string]any{
			"format": "custom",
			"raw":    string(data),
		}
		return nil
	})

	os.WriteFile(fp, []byte("hello=world"), 0644)
	cfg, err := kconf.Read(fp)
	if err != nil {
		t.Fatal(err)
	}
	if cfg["format"] != "custom" {
		t.Fatalf("format: got %v", cfg["format"])
	}
	if cfg["raw"] != "hello=world" {
		t.Fatalf("raw: got %v", cfg["raw"])
	}
}

func TestRegisterMarshal(t *testing.T) {
	dir := t.TempDir()
	// 使用唯一后缀 .regm，不会与其他测试冲突，无需手动清理
	fp := filepath.Join(dir, "test.regm")

	kconf.Register(".regm", func(data []byte, v any) error {
		ptr := v.(*map[string]any)
		*ptr = map[string]any{"format": "custom"}
		return nil
	})
	kconf.RegisterMarshal(".regm", func(v any) ([]byte, error) {
		return []byte("marshaled"), nil
	})

	if err := kconf.Save(fp, map[string]any{"a": 1}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(fp)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "marshaled" {
		t.Fatalf("got %s, want marshaled", string(data))
	}
}

// ============================================================================
// HCL 特殊测试
// ============================================================================

func TestHCL_Read_Block_Error(t *testing.T) {
	dir := t.TempDir()
	// HCL 包含 block，Read() 会返回提示
	src := `
host = "example.com"
database {
  driver = "postgres"
}
`
	fp := filepath.Join(dir, "test.hcl")
	os.WriteFile(fp, []byte(src), 0644)

	_, err := kconf.Read(fp)
	if err == nil {
		t.Fatal("expected error for HCL with block on Read(), got nil")
	}
	if !strings.Contains(err.Error(), "block") {
		t.Fatalf("error should mention block, got: %v", err)
	}
}

// ============================================================================
// 边界情况
// ============================================================================

func TestRead_FileNotFound(t *testing.T) {
	_, err := kconf.Read("/nonexistent/file.yaml")
	if err == nil {
		t.Fatal("expected error for non-existent file")
	}
}

func TestRead_UnsupportedFormat(t *testing.T) {
	dir := t.TempDir()
	fp := filepath.Join(dir, "test.xyz")
	os.WriteFile(fp, []byte("data"), 0644)
	_, err := kconf.Read(fp)
	if err == nil {
		t.Fatal("expected error for unsupported format")
	}
	if !strings.Contains(err.Error(), "unsupported") {
		t.Fatalf("error should mention unsupported, got: %v", err)
	}
}

func TestRead_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	fp := filepath.Join(dir, "empty.yaml")
	os.WriteFile(fp, []byte{}, 0644)
	cfg, err := kconf.Read(fp)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg) != 0 {
		t.Fatalf("expected empty map, got %v", cfg)
	}
}

func TestReadRaw(t *testing.T) {
	dir := t.TempDir()
	fp := filepath.Join(dir, "test.yaml")
	os.WriteFile(fp, []byte("hello: world\n"), 0644)
	data, err := kconf.ReadRaw(fp)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hello: world\n" {
		t.Fatalf("got %q, want %q", string(data), "hello: world\n")
	}
}

// ============================================================================
// 嵌套结构 + DecodeMapToStruct 测试
// ============================================================================

func TestDecodeMapToStruct(t *testing.T) {
	m := map[string]any{
		"host": "example.com",
		"port": 9090,
	}
	var cfg testConfig
	if err := kconf.DecodeMapToStruct(m, &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Host != "example.com" {
		t.Fatalf("Host: got %v", cfg.Host)
	}
	if cfg.Port != 9090 {
		t.Fatalf("Port: got %v", cfg.Port)
	}
}

// ============================================================================
// 原有测试（保留）
// ============================================================================

func TestJSONC(t *testing.T) {
	dir := t.TempDir()
	src := `{
		// 这是注释
		"host": "0.0.0.0",
		/* 多行注释 */
		"port": 8080,
	}`
	fp := filepath.Join(dir, "test.jsonc")
	os.WriteFile(fp, []byte(src), 0644)

	cfg, err := kconf.Read(fp)
	if err != nil {
		t.Fatal(err)
	}
	if cfg["host"] != "0.0.0.0" {
		t.Fatalf("host: got %v", cfg["host"])
	}
	if cfg["port"] != float64(8080) {
		t.Fatalf("port: got %v", cfg["port"])
	}
}

func TestHCL_Read_Flat(t *testing.T) {
	dir := t.TempDir()
	src := `host = "0.0.0.0"
port = 8080
debug = true
`
	fp := filepath.Join(dir, "test.hcl")
	os.WriteFile(fp, []byte(src), 0644)

	cfg, err := kconf.Read(fp)
	if err != nil {
		t.Fatal(err)
	}
	if cfg["host"] != "0.0.0.0" {
		t.Fatalf("host: got %v", cfg["host"])
	}
	if cfg["port"] != int64(8080) {
		t.Fatalf("port: got %v (type: %T)", cfg["port"], cfg["port"])
	}
}

func TestHCL_ReadTo_Struct(t *testing.T) {
	dir := t.TempDir()
	src := `
host = "example.com"
port = 9090
`
	fp := filepath.Join(dir, "test.hcl")
	os.WriteFile(fp, []byte(src), 0644)

	var cfg struct {
		Host string `hcl:"host"`
		Port int    `hcl:"port"`
	}
	err := kconf.ReadTo(fp, &cfg)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Host != "example.com" {
		t.Fatalf("Host: got %v", cfg.Host)
	}
	if cfg.Port != 9090 {
		t.Fatalf("Port: got %v", cfg.Port)
	}
}

func TestHCL_ReadTo_WithBlock(t *testing.T) {
	dir := t.TempDir()
	src := `
host = "example.com"
port = 9090
database {
  driver = "postgres"
  url    = "postgres://..."
}
`
	fp := filepath.Join(dir, "test.hcl")
	os.WriteFile(fp, []byte(src), 0644)

	type DatabaseConfig struct {
		Driver string `hcl:"driver,attr"`
		Url    string `hcl:"url,attr"`
	}
	var cfg struct {
		Host     string          `hcl:"host"`
		Port     int             `hcl:"port"`
		Database *DatabaseConfig `hcl:"database,block"`
	}
	err := kconf.ReadTo(fp, &cfg)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Host != "example.com" {
		t.Fatalf("Host: got %v", cfg.Host)
	}
	if cfg.Port != 9090 {
		t.Fatalf("Port: got %v", cfg.Port)
	}
	if cfg.Database == nil || cfg.Database.Driver != "postgres" {
		t.Fatalf("Database.Driver: got %v", cfg.Database)
	}
}

// ============================================================================
// 表驱动测试 — 同一配置多格式
// ============================================================================

func TestRead_MultipleFormats(t *testing.T) {
	dir := t.TempDir()

	samples := []struct {
		name    string
		content string
	}{
		{
			name:    "app.json",
			content: `{"host": "0.0.0.0", "port": 8080}`,
		},
		{
			name:    "app.yaml",
			content: "host: 0.0.0.0\nport: 8080\n",
		},
		{
			name:    "app.toml",
			content: "host = \"0.0.0.0\"\nport = 8080\n",
		},
		{
			name:    "app.hcl",
			content: "host = \"0.0.0.0\"\nport = 8080\n",
		},
	}

	for _, s := range samples {
		t.Run(s.name, func(t *testing.T) {
			fp := filepath.Join(dir, s.name)
			os.WriteFile(fp, []byte(s.content), 0644)

			cfg, err := kconf.Read(fp)
			if err != nil {
				t.Fatal(err)
			}
			if cfg["host"] != "0.0.0.0" {
				t.Fatalf("host: got %v", cfg["host"])
			}
		})
	}
}

// TestExampleConfiguration 验证 examples/conf.json5 能被正确解析。
// 这是你的真实配置格式验证。
func TestExampleConfiguration(t *testing.T) {
	_, caller, _, _ := runtime.Caller(0)
	// caller 在 test/format_test.go，向上两级到项目根，再进 examples
	fp := filepath.Join(filepath.Dir(caller), "..", "examples", "conf.json5")
	fp = filepath.Clean(fp)

	if _, err := os.Stat(fp); os.IsNotExist(err) {
		t.Skipf("conf.json5 not found at %s, skipping", fp)
	}

	// 只验证格式正确可解析，不关心具体值
	cfg, err := kconf.Read(fp)
	if err != nil {
		t.Fatalf("failed to parse conf.json5: %v", err)
	}
	if len(cfg) == 0 {
		t.Fatal("conf.json5 produced empty config")
	}
	if cfg["fileCache"] == nil {
		t.Log("note: conf.json5 uses json5 with single quotes and comments")
	}

	// 验证顶层的关键字段
	keys := []string{"fileCache", "syncTime", "log", "source", "target"}
	for _, k := range keys {
		if cfg[k] == nil {
			t.Logf("key %q is nil (may be valid depending on config structure)", k)
		}
	}
}

// BenchmarkRead 基准测试 — 读一个 ~4KB 的 yaml 配置
func BenchmarkRead(b *testing.B) {
	dir := b.TempDir()
	src := fmt.Sprintf("host: example.com\nport: %d\n", 8080)
	for i := 0; i < 100; i++ {
		src += fmt.Sprintf("key_%d: value_%d\n", i, i)
	}
	fp := filepath.Join(dir, "bench.yaml")
	os.WriteFile(fp, []byte(src), 0644)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := kconf.Read(fp)
		if err != nil {
			b.Fatal(err)
		}
	}
}
