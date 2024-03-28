package config

type GRPC struct {
	Ip          string  `mapstructure:"ip" json:"ip" yaml:"ip"`                               // IP
	Port        uint64  `mapstructure:"port" json:"port" yaml:"port"`                         // 端口
	ServiceName string  `mapstructure:"service-name" json:"service-name" yaml:"service-name"` // 服务
	Weight      float64 `mapstructure:"weight" json:"weight" yaml:"weight"`                   // 权重
}
