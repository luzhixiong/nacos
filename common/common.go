package common

import (
	"encoding/base64"
	"gitea.innolabs.cloud/server/nacos/config"
	"github.com/nacos-group/nacos-sdk-go/clients"
	"github.com/nacos-group/nacos-sdk-go/clients/config_client"
	"github.com/nacos-group/nacos-sdk-go/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/vo"
	"sync"
)

type NacosMgr struct {
	Client   naming_client.INamingClient
	CfgStore *ConfigMgr
}

func (nm *NacosMgr) Start() {
}

func (nm *NacosMgr) Stop() {
}

func (nm *NacosMgr) FindInstance(clusterName, groupName, serviceName string) (ip string, port uint64, err error) {
	ins, err := nm.Client.SelectOneHealthyInstance(vo.SelectOneHealthInstanceParam{
		Clusters:    []string{clusterName},
		ServiceName: serviceName,
		GroupName:   groupName,
	})
	if err != nil {
		return
	}
	return ins.Ip, ins.Port, err
}

// ////////////////////////// config_store /////////////////////////////////
type ACfgEventType int

const (
	ACfgAdd ACfgEventType = iota
	ACfgChange
	ACfgErr
)

type ACfgEvt struct {
	DataId string
	Group  string
	Evt    ACfgEventType
	Data   string
	Err    error
}

var _names = map[ACfgEventType]string{
	ACfgAdd:    "Config Load",
	ACfgChange: "Config Watch Change",
	ACfgErr:    "Config op Error",
}

func EvtName(e ACfgEventType) string {
	return _names[e]
}

var configMgr *ConfigMgr

type ConfigMgr struct {
	client config_client.IConfigClient
	cfg    *config.Nacos
	Stores map[string]*ConfigStore
}

func NewConfigMgr(cfg *config.Nacos) (*ConfigMgr, error) {
	sc := []constant.ServerConfig{
		{
			IpAddr: cfg.Host,
			Port:   cfg.Port,
		},
	}

	nameSpaceId := cfg.NameSpaceId
	if cfg.CfgNameSpaceId != "" {
		nameSpaceId = cfg.CfgNameSpaceId
	}
	cc := constant.ClientConfig{
		NamespaceId:         nameSpaceId,
		TimeoutMs:           30000,
		NotLoadCacheAtStart: true,
		LogDir:              "./log",
		CacheDir:            "./cache",
		LogLevel:            "warn",
		Username:            cfg.UserName,
		Password:            cfg.Password,
	}

	client, err := clients.CreateConfigClient(map[string]interface{}{
		"serverConfigs": sc,
		"clientConfig":  cc,
	})
	if err != nil {
		return nil, err
	}
	configMgr = &ConfigMgr{
		client: client,
		cfg:    cfg,
	}
	//for _, di := range cfg.DataIds {
	//	cfgStore := configMgr.newConfigStore(di, cfg.GroupName)
	//	go cfgStore.Load()
	//	configMgr.stores[di] = cfgStore
	//}
	return configMgr, nil
}

type ConfigStore struct {
	sync.RWMutex

	dataId      string
	group       string
	watchEventC chan *ACfgEvt
}

func (cm *ConfigMgr) NewConfigStore(dataId string, watchEvt chan *ACfgEvt) *ConfigStore {
	return &ConfigStore{
		dataId:      dataId,
		group:       cm.cfg.GroupName,
		watchEventC: watchEvt,
	}
}

func (s *ConfigStore) loadConfig() error {
	s.Lock()
	defer s.Unlock()

	content, err := configMgr.client.GetConfig(vo.ConfigParam{
		DataId: s.dataId,
		Group:  s.group,
	})
	e := ACfgAdd
	if err != nil {
		e = ACfgErr
	}
	s.watchEventC <- &ACfgEvt{
		DataId: s.dataId,
		Group:  s.group,
		Evt:    e,
		Data:   content,
		Err:    err,
	}
	return nil
}

func (s *ConfigStore) watchConfig() {
	err := configMgr.client.ListenConfig(vo.ConfigParam{
		DataId: s.dataId,
		Group:  s.group,
		OnChange: func(namespace, group, dataId, data string) {
			s.watchEventC <- &ACfgEvt{
				DataId: dataId,
				Group:  group,
				Evt:    ACfgChange,
				Data:   data,
			}
		},
	})
	if err != nil {
		s.watchEventC <- &ACfgEvt{
			DataId: s.dataId,
			Group:  s.group,
			Evt:    ACfgErr,
			Data:   "",
			Err:    err,
		}
	}
}

func (s *ConfigStore) stopWatch() {
	_ = configMgr.client.CancelListenConfig(vo.ConfigParam{
		DataId: s.dataId,
		Group:  s.group,
	})
}

// /////////////////////////////////////////////////////////////////
func (s *ConfigStore) Load() {
	err := s.loadConfig()
	if err != nil {
		return
	}
	s.watchConfig()
}

func (s *ConfigStore) LoadIm() (string, error) {
	s.Lock()
	defer s.Unlock()

	content, err := configMgr.client.GetConfig(vo.ConfigParam{
		DataId: s.dataId,
		Group:  s.group,
	})
	if err != nil {
		return "", err
	}
	s.watchConfig()

	return content, nil
}

func (s *ConfigStore) Publish(data []byte) (bool, error) {
	s.Lock()
	defer s.Unlock()

	flag, err := configMgr.client.PublishConfig(vo.ConfigParam{
		DataId:  s.dataId,
		Group:   s.group,
		Content: base64.StdEncoding.EncodeToString(data),
	})
	return flag, err
}

func (s *ConfigStore) Stop() {
	s.stopWatch()
}
