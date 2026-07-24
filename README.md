# kconfig

通用的多格式配置文件读取库，支持 **json / json5 / jsonc / yaml / toml / hcl**

核心设计：**不做业务假设** —— 只管格式解析，不管内容结构。

---

## 安装

```bash
go get github.com/khan-lau/kconfig
```

## 快速开始

### 读取到 map（内容不固定）

```go
import "github.com/khan-lau/kconfig"

cfg, _ := kconf.Read("config.yaml")
host := cfg["host"].(string)
port := cfg["port"].(int)
```

### 读取到 struct（已知结构）

```go
var cfg AppConfig
err := kconf.ReadTo("config.toml", &cfg)
```

### 同一行代码，自动识别格式

json / json5 / yaml / toml / hcl 自动按后缀识别：

```go
// 后缀不同，代码一样
cfg1, _ := kconf.Read("app.json")   // → json
cfg2, _ := kconf.Read("app.yaml")   // → yaml
cfg3, _ := kconf.Read("app.toml")   // → toml
cfg4, _ := kconf.Read("app.hcl")    // → hcl
```

---

## API

### Read

读取配置文件，以 `map[string]any` 返回。

```go
func Read(filePath string, opts ...Option) (map[string]any, error)
```

### ReadTo

读取配置文件，解码到指定的结构体。

```go
func ReadTo(filePath string, target any, opts ...Option) error
```

### Save

将配置写回文件。格式由文件后缀自动识别。

```go
func Save(filePath string, v any) error
```

```go
cfg, _ := kconf.Read("config.yaml")
cfg["port"] = 9090
kconf.Save("config.yaml", cfg)
```

> **注意事项**
>
> - **HCL** 不支持写回（HCL 是类型化语言，无通用序列化方案）
> - 其余格式写回时，**注释和格式排版会丢失**—— Go 的 marshaler 重新生成结构化的输出，注释不属于数据模型，序列化时不保留
> - json5 写回后标准化为标准 JSON（移除注释、尾部逗号、无引号 key），**但读回结果一致**

### ReadRaw

读取配置文件，以原始字节返回。

```go
func ReadRaw(filePath string) ([]byte, error)
```

### Detect

检测文件格式，返回格式名称。

```go
func Detect(filePath string) (string, error)
```

### Register / RegisterMarshal

注册自定义格式的解码器/编码器。

```go
kconf.Register(".conf", myUnmarshalFunc)       // 读
kconf.RegisterMarshal(".conf", myMarshalFunc)  // 写
```

---

## Processor（后处理）

`Processor` 接口在反序列化后执行，用于设置默认值、校验、类型转换。

```go
// map 场景 — 补默认值
cfg, _ := kconf.Read("config.yaml",
    kconf.WithProcessor(kconf.ProcessFunc(func(v any) error {
        m := v.(map[string]any)
        if m["port"] == nil {
            m["port"] = 8080
        }
        return nil
    })))

// struct 场景 — 非法值修正
var cfg AppConfig
kconf.ReadTo("config.toml", &cfg,
    kconf.WithProcessor(kconf.ProcessFunc(func(v any) error {
        c := v.(*AppConfig)
        if c.Port <= 0 {
            c.Port = 8080
        }
        return nil
    })))
```

---

## DecodeMapToStruct（map → 结构体）

当配置包含 `any` 字段（如 `Item any`），反序列化后为 `map[string]any`，可用该方法转为具体类型：

```go
import "github.com/khan-lau/kconfig"

for _, db := range cfg.Database {
    if m, ok := db.Item.(map[string]any); ok {
        switch db.DBType {
        case "redis":
            var rc RedisConfig
            kconf.DecodeMapToStruct(m, &rc)
            db.Item = &rc
        }
    }
}
```

---

## 支持格式


| 格式    | 后缀             | struct 标签  | 读   | 写   | 备注                        |
|--------|------------------|-------------|:----:|:----:|---------------------------|
| JSON   | `.json`          | `json:`     | 支持  | 支持 | 标准 JSON                   |
| JSON5  | `.json5`         | `json:`     | 支持  | 支持 | 写回标准化为标准 JSON，注释丢失 |
| JSONC  | `.jsonc`         | `json:`     | 支持  | 支持 | 写回为标准 JSON，注释丢失      |
| YAML   | `.yaml` / `.yml` | `yaml:`     | 支持  | 支持 | 写回时注释丢失                |
| TOML   | `.toml`          | `toml:`     | 支持  | 支持 | 写回时注释丢失                |
| HCL    | `.hcl`           | `hcl:`      | 支持  | 只读 | 无通用序列化方案              |  


### JSON（`.json`）

最通用的数据交换格式，语法严格，所有语言都有原生支持。键和字符串必须使用双引号，不支持注释和尾部逗号。适合机器读写、API 交互场景。

```json
{
  "host": "0.0.0.0",
  "port": 8080,
  "debug": true,
  "rate": 0.85,
  "tags": ["api", "web"],
  "database": {
    "driver": "postgres",
    "url": "postgres://localhost/mydb"
  },
  "redis": {
    "addrs": ["10.0.0.1:6379", "10.0.0.2:6379"]
  }
}
```

### JSON5（`.json5`）

JSON 的宽松扩展，兼容标准 JSON，额外支持：单/双引号字符串、尾部逗号、单行/多行注释、无引号 key、十六进制数。适合手写配置，比 JSON 更友好。

```json5
{
  // 服务器配置
  host: '0.0.0.0',       // 单引号字符串
  port: 8080,              // 尾部逗号
  debug: true,
  rate: 0.85,
  tags: ['api', 'web'],    // 数组

  // 数据库配置
  database: {
    driver: 'postgres',
    url: 'postgres://localhost/mydb',
  },

  /* 缓存集群 */
  redis: {
    addrs: [
      '10.0.0.1:6379',
      '10.0.0.2:6379',
    ],
  },
}
```

### JSONC（`.jsonc`）

仅比 JSON 多支持注释（`//` 和 `/* */`），其余语法与标准 JSON 完全一致。适合需要注释但又想保持 JSON 严格语法的场景。

```jsonc
{
  // 服务器配置
  "host": "0.0.0.0",
  "port": 8080,
  "debug": true,
  "rate": 0.85,
  // 标签列表
  "tags": ["api", "web"],
  "database": {
    "driver": "postgres",
    /* 连接串 */
    "url": "postgres://localhost/mydb"
  },
  "redis": {
    "addrs": ["10.0.0.1:6379", "10.0.0.2:6379"]
  }
}
```

### YAML（`.yaml` / `.yml`）

基于缩进的配置语言，不写引号，结构清晰直观。天然支持多层嵌套、列表、多行字符串。适合复杂配置场景，但缩进敏感，大型文件容易出错。

```yaml
# 服务器配置
host: 0.0.0.0
port: 8080
debug: true
rate: 0.85

# 标签
tags:
  - api
  - web

# 数据库
database:
  driver: postgres
  url: postgres://localhost/mydb

# 缓存集群
redis:
  addrs:
    - 10.0.0.1:6379
    - 10.0.0.2:6379
```

### TOML（`.toml`）

显式类型、表（table）结构的配置语言。语法接近 INI，直接支持 key-value 对、嵌套表、数组。类型明确（字符串必须加引号，数字不加），可读性强。适合应用配置场景，Rust 社区广泛使用。

```toml
# 服务器配置
host = "0.0.0.0"
port = 8080
debug = true
rate = 0.85
tags = ["api", "web"]

# 数据库
[database]
driver = "postgres"
url = "postgres://localhost/mydb"

# 缓存集群
[redis]
addrs = ["10.0.0.1:6379", "10.0.0.2:6379"]
```

### HCL（`.hcl`）

HashiCorp 的类型化配置语言，支持属性（attribute）和块（block）两种结构。块可以嵌套，表达层级关系非常自然。Terraform、Consul、Vault 等工具使用 HCL。HCL 是类型化语言，解码到 struct 最完美，不支持通用序列化写回。

```hcl
host   = "0.0.0.0"
port   = 8080
debug  = true
rate   = 0.85
tags   = ["api", "web"] # 行尾注释也支持

# 数据库
database {
  driver = "postgres"
  url    = "postgres://localhost/mydb"
}

/* 缓存集群
   支持双机 */
redis {
  addrs = ["10.0.0.1:6379", "10.0.0.2:6379"]
}
```

#### HCL 注意事项

- `ReadTo(struct)` — **完美支持**，含 block 嵌套
- `Read(map)` — 仅支持纯属性（无 block），有 block 时返回提示
- `Read(map)` 路径已跳过 JSON 中间格式，`cty.Value` 直接转换 Go 原生类型，**无类型信息损失**（数字保持 `int64`，不退化到 `float64`）
- `Save()` — **不支持**（HCL 无通用序列化方案）
- struct 标签使用 `hcl:"fieldName"` / `hcl:"fieldName,attr"` / `hcl:"blockName,block"`

---

## 项目结构

```
kconf/
  format.go   格式注册、解码/编码器查找、Detect、Register/RegisterMarshal
  loader.go   Read / ReadTo / ReadRaw / Save + Processor 接口
  decode.go   DecodeMapToStruct（mapstructure 封装）
  examples/   使用 conf.json5 加载完整配置的示例
```
