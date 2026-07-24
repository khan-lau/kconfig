package main

import (
	kconf "github.com/khan-lau/kconfig"

	mqConf "github.com/khan-lau/kmq/service/mq/config"
)

type Configure struct {
	FileCache    string         `json:"fileCache" toml:"fileCache" yaml:"fileCache" hcl:"fileCache,attr"`             // 文件缓存目录, 中转文件临时存放路径配置
	SyncTime     uint64         `json:"syncTime" toml:"syncTime" yaml:"syncTime" hcl:"syncTime,attr"`                 // 同步周期, 单位毫秒,不低于1000毫秒
	SyncFile     string         `json:"syncFile" toml:"syncFile" yaml:"syncFile" hcl:"syncFile,attr"`                 // 同步文件路径, 同步偏移量缓存文件路径配置
	PointMap     string         `json:"pointMap" toml:"pointMap" yaml:"pointMap" hcl:"pointMap,attr"`                 // 映射点配置文件路径, 映射点配置文件路径配置
	QueueSize    int            `json:"queueSize" toml:"queueSize" yaml:"queueSize" hcl:"queueSize,attr"`             // 消息队列大小, 不超过cpu核心数的2倍
	SendInterval int64          `json:"sendInterval" toml:"sendInterval" yaml:"sendInterval" hcl:"sendInterval,attr"` // 发送间隔, 单位毫秒
	Log          *Log           `json:"log" toml:"log" yaml:"log" hcl:"log,block"`                                    // 日志配置
	WebRPC       *WebRPC        `json:"webRPC" toml:"webRPC" yaml:"webRPC" hcl:"webRPC,block"`                        // web api 配置
	Database     []*DatabaseObj `json:"database" toml:"database" yaml:"database" hcl:"database,block"`                // 数据库配置列表
	Source       []*MQItemObj   `json:"source" toml:"source" yaml:"source" hcl:"source,block"`                        // 消息队列配置
	Target       []*MQItemObj   `json:"target" toml:"target" yaml:"target" hcl:"target,block"`                        // 消息队列配置
}

// ConfigInstance 读取配置文件，返回 Config 实例。
// 格式由文件后缀自动识别（json / json5 / yaml / toml / hcl）。
// 嵌套的 any 字段（Database.Item、Source.Item、Target.Item）会自动从
// map 解码为对应的 mqConf.*Config 结构体。
func ConfigInstance(filePath string) (*Configure, error) {
	var cfg Configure
	err := kconf.ReadTo(filePath, &cfg, kconf.WithProcessor(kconf.ProcessFunc(postProcess)))
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

// postProcess 反序列化后的后处理：
//   - 将 Database/Source/Target 中 any 包裹的配置 map 转为具体结构体
//   - 设置默认值
func postProcess(v any) error {
	cfg := v.(*Configure)

	// ---------- Database ----------
	for _, item := range cfg.Database {
		if item == nil || item.Item == nil {
			continue
		}
		m, ok := item.Item.(map[string]any)
		if !ok {
			continue
		}
		switch item.DBType {
		case "redis":
			var rc mqConf.RedisConfig
			if err := kconf.DecodeMapToStruct(m, &rc); err != nil {
				return err
			}
			item.Item = &rc
		case "postgres":
			var pc PostgresConf
			if err := kconf.DecodeMapToStruct(m, &pc); err != nil {
				return err
			}
			item.Item = &pc
		}
	}

	// ---------- Source ----------
	for _, item := range cfg.Source {
		if item == nil || item.Item == nil {
			continue
		}
		m, ok := item.Item.(map[string]any)
		if !ok {
			continue
		}
		switch item.MQType {
		case "natscoremq":
			var nc mqConf.NatsCoreConfig
			if err := kconf.DecodeMapToStruct(m, &nc); err != nil {
				return err
			}
			item.Item = &nc
		case "natsjsmq":
			var nc mqConf.NatsJsConfig
			if err := kconf.DecodeMapToStruct(m, &nc); err != nil {
				return err
			}
			nc.ConsumerConfig.StartWithTimestamp = -1
			if nc.ConsumerConfig.AutoCommit == "" {
				nc.ConsumerConfig.AutoCommit = "native"
			}
			item.Item = &nc
		case "kafkamq":
			var kc mqConf.KafkaConfig
			if err := kconf.DecodeMapToStruct(m, &kc); err != nil {
				return err
			}
			item.Item = &kc
		case "rabbitmq":
			var rc mqConf.RabbitConfig
			if err := kconf.DecodeMapToStruct(m, &rc); err != nil {
				return err
			}
			item.Item = &rc
		case "rocketmq":
			var rc mqConf.RocketConfig
			if err := kconf.DecodeMapToStruct(m, &rc); err != nil {
				return err
			}
			item.Item = &rc
		case "redismq":
			var rc mqConf.RedisConfig
			if err := kconf.DecodeMapToStruct(m, &rc); err != nil {
				return err
			}
			item.Item = &rc
		case "mqtt3":
			var mc mqConf.MqttConfig
			if err := kconf.DecodeMapToStruct(m, &mc); err != nil {
				return err
			}
			item.Item = &mc
		}
	}

	// ---------- Target ----------
	for _, item := range cfg.Target {
		if item == nil || item.Item == nil {
			continue
		}
		m, ok := item.Item.(map[string]any)
		if !ok {
			continue
		}
		switch item.MQType {
		case "natscoremq":
			var nc mqConf.NatsCoreConfig
			if err := kconf.DecodeMapToStruct(m, &nc); err != nil {
				return err
			}
			item.Item = &nc
		case "natsjsmq":
			var nc mqConf.NatsJsConfig
			if err := kconf.DecodeMapToStruct(m, &nc); err != nil {
				return err
			}
			item.Item = &nc
		case "kafkamq":
			var kc mqConf.KafkaConfig
			if err := kconf.DecodeMapToStruct(m, &kc); err != nil {
				return err
			}
			item.Item = &kc
		case "rabbitmq":
			var rc mqConf.RabbitConfig
			if err := kconf.DecodeMapToStruct(m, &rc); err != nil {
				return err
			}
			item.Item = &rc
		case "rocketmq":
			var rc mqConf.RocketConfig
			if err := kconf.DecodeMapToStruct(m, &rc); err != nil {
				return err
			}
			item.Item = &rc
		case "redismq":
			var rc mqConf.RedisConfig
			if err := kconf.DecodeMapToStruct(m, &rc); err != nil {
				return err
			}
			item.Item = &rc
		case "mqtt3":
			var mc mqConf.MqttConfig
			if err := kconf.DecodeMapToStruct(m, &mc); err != nil {
				return err
			}
			item.Item = &mc
		}
	}

	// ---------- Log 默认值 ----------
	if cfg.Log != nil {
		if cfg.Log.FlushInterval < 1 {
			cfg.Log.FlushInterval = 1000
		}
		if cfg.Log.BufferSize < 1 {
			cfg.Log.BufferSize = 4 * 1024 * 1024
		}
	}

	return nil
}

// ============================================================================
// 子类型定义
// ============================================================================

type Log struct {
	LogLevel      int8   `json:"logLevel" toml:"logLevel" yaml:"logLevel" hcl:"logLevel,attr"`                     // 日志等级
	Colorful      bool   `json:"colorful" toml:"colorful" yaml:"colorful" hcl:"colorful,attr"`                     // 是否彩色输出
	MaxAge        int    `json:"maxAge" toml:"maxAge" yaml:"maxAge" hcl:"maxAge,attr"`                             // 文件最大保存数量
	MaxSize       int64  `json:"maxSize" toml:"maxSize" yaml:"maxSize" hcl:"maxSize,attr"`                         // 文件最大滚动大小, 单位为Byte, 默认10G
	RotationTime  int    `json:"rotationTime" toml:"rotationTime" yaml:"rotationTime" hcl:"rotationTime,attr"`     // 文件最大滚动时间
	Console       bool   `json:"console" toml:"console" yaml:"console" hcl:"console,attr"`                         // 是否输出到控制台
	Async         bool   `json:"async" toml:"async" yaml:"async" hcl:"async,attr"`                                 // 是否异步输出
	FlushInterval int64  `json:"flushInterval" toml:"flushInterval" yaml:"flushInterval" hcl:"flushInterval,attr"` // 异步输出日志缓冲区刷新间隔, 单位为毫秒
	BufferSize    int64  `json:"bufferSize" toml:"bufferSize" yaml:"bufferSize" hcl:"bufferSize,attr"`             // 异步输出日志缓冲区大小, 单位为条数
	LogDir        string `json:"logDir" toml:"logDir" yaml:"logDir" hcl:"logDir,attr"`                             // 日志文件存储目录
}

type WebRPC struct {
	Host           string `json:"host" toml:"host" yaml:"host" hcl:"host,attr"`                                         // webRPC服务地址
	Port           uint16 `json:"port" toml:"port" yaml:"port" hcl:"port,attr"`                                         // 端口号
	ReloadCooldown uint64 `json:"reloadCooldown" toml:"reloadCooldown" yaml:"reloadCooldown" hcl:"reloadCooldown,attr"` // pointMap热加载冷却间隔, 单位分钟
}

type PostgresConf struct {
	Url         string `json:"url" toml:"url" yaml:"url" hcl:"url,attr"`                                 // 数据库连接
	MaxConn     int    `json:"maxConn" toml:"maxConn" yaml:"maxConn" hcl:"maxConn,attr"`                 // 最大连接数
	MaxIdle     int    `json:"maxIdle" toml:"maxIdle" yaml:"maxIdle" hcl:"maxIdle,attr"`                 // 最大空闲连接数
	MaxIdleTime int    `json:"maxIdleTime" toml:"maxIdleTime" yaml:"maxIdleTime" hcl:"maxIdleTime,attr"` // 最大空闲时间, 单位毫秒
	MaxLifetime int    `json:"maxLifetime" toml:"maxLifetime" yaml:"maxLifetime" hcl:"maxLifetime,attr"` // 最长生命周期, 单位毫秒
	Schema      string `json:"schema" toml:"schema" yaml:"schema" hcl:"schema,attr"`                     // 数据库schema
}

type DatabaseObj struct {
	Tag    string `json:"tag" toml:"tag" yaml:"tag" hcl:"tag,attr"`     // 用于路由的标签, 用于区分不同的数据库, 名字必须在作用域内唯一, 不可重复
	DBType string `json:"type" toml:"type" yaml:"type" hcl:"type,attr"` // 数据库类型, 支持的类型: redis postgres
	Item   any    `json:"rdb" toml:"rdb" yaml:"rdb" hcl:"rdb,attr"`     // 数据库配置
}

type MQItemObj struct {
	Tag        string `json:"tag" toml:"tag" yaml:"tag" hcl:"tag,attr"`                             // 用于路由的标签, 用于区分不同的消息队列, 名字必须在作用域内唯一, 不可重复
	IsCompress bool   `json:"isCompress" toml:"isCompress" yaml:"isCompress" hcl:"isCompress,attr"` // 是否压缩消息
	MQType     string `json:"type" toml:"type" yaml:"type" hcl:"type,attr"`                         // 消息队列类型, 支持的类型: kafkamq, rabbitmq, redismq, rocketmq, mqtt3
	Item       any    `json:"mq" toml:"mq" yaml:"mq" hcl:"mq,attr"`                                 // 消息队列配置
}

func (that *MQItemObj) MQConfig() any {
	return that.Item
}
