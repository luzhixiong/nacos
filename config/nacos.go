package config

type Nacos struct {
	Host        string `mapstructure:"host" json:"host" yaml:"host" `                           // 域名/IP
	Port        uint64 `mapstructure:"port" json:"port" yaml:"port"`                            // 端口
	ContextPath string `mapstructure:"context-path" json:"context-path" yaml:"context-path"`    // 前缀 默认/nacos
	Scheme      string `mapstructure:"scheme" json:"scheme" yaml:"scheme"`                      // http或https
	NameSpaceId string `mapstructure:"name-space-id" json:"name-space-id" yaml:"name-space-id"` // 命名空间
	UserName    string `mapstructure:"username" json:"username" yaml:"username"`                // 账号
	Password    string `mapstructure:"password" json:"password" yaml:"password"`                // 密码
	ClusterName string `mapstructure:"cluster-name" json:"cluster-name" yaml:"cluster-name"`    // 集群名 默认:DEFAULT
	GroupName   string `mapstructure:"group-name" json:"group-name" yaml:"group-name"`          // 分组名 默认:DEFAULT_GROUP

	CfgNameSpaceId string `mapstructure:"cfg-name-space-id" json:"cfg-name-space-id" yaml:"cfg-name-space-id"` // 配置中心命名空间/也用于判断是否开启配置中心
}
