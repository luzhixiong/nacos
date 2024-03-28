package common

type NacosService struct {
	Ip          string `json:"ip"`
	Port        uint64 `json:"port"`
	ServiceName string `json:"service_name"`
}
