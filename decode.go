package kconf

import "github.com/khan-lau/mapstructure"

// DecodeMapToStruct 将 map[string]any 解码到指定的目标结构体中。
// 对应你项目中 process() 里 mapstructure.Decode(anyMap, &xxxConf) 的用法。
//
// 典型场景（Processor 内使用）：
//
//	kconf.WithProcessor(kconf.ProcessFunc(func(v any) error {
//	    cfg := v.(*AppConfig)
//	    for _, db := range cfg.Database {
//	        if m, ok := db.Item.(map[string]any); ok {
//	            switch db.DBType {
//	            case "redis":
//	                var redisConf RedisConfig
//	                if err := kconf.DecodeMapToStruct(m, &redisConf); err != nil {
//	                    return err
//	                }
//	                db.Item = &redisConf
//	            }
//	        }
//	    }
//	    return nil
//	}))
func DecodeMapToStruct(src map[string]any, dst any) error {
	return mapstructure.Decode(src, dst)
}
