package nacos

import (
	"context"
	"errors"
	"fmt"
	"gitea.innolabs.cloud/server/nacos/common"
	"gitea.innolabs.cloud/server/nacos/config"
	"github.com/nacos-group/nacos-sdk-go/clients"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/vo"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
	"sync"
	"time"
)

var (
	nacosMgr *common.NacosMgr
	mu       sync.RWMutex

	grpcConns = sync.Map{}
)

func BeginNacos(cfg config.Nacos, grpcCfg config.GRPC) *common.NacosMgr {
	clientConfig := constant.ClientConfig{
		TimeoutMs:           5000,
		NotLoadCacheAtStart: true,
		LogDir:              "./log",
		CacheDir:            "./cache",
		LogLevel:            "warn",
		NamespaceId:         cfg.NameSpaceId,
		Username:            cfg.UserName,
		Password:            cfg.Password,
	}
	serverConfigs := []constant.ServerConfig{
		{
			IpAddr:      cfg.Host,
			ContextPath: cfg.ContextPath,
			Port:        cfg.Port,
			Scheme:      cfg.Scheme,
		},
	}
	client, err := clients.NewNamingClient(
		vo.NacosClientParam{
			ClientConfig:  &clientConfig,
			ServerConfigs: serverConfigs,
		},
	)
	if err != nil {
		return nil
	}
	if grpcCfg.Ip != "" && grpcCfg.Port != 0 {
		_, er := client.RegisterInstance(vo.RegisterInstanceParam{
			Ip:          grpcCfg.Ip,
			Port:        grpcCfg.Port,
			ServiceName: grpcCfg.ServiceName,
			Weight:      10,
			Enable:      true,
			Healthy:     true,
			Ephemeral:   true,
			Metadata:    map[string]string{"type": "grpc"},
			ClusterName: cfg.ClusterName, // default value is DEFAULT
			GroupName:   cfg.GroupName,   // default value is DEFAULT_GROUP
		})
		if er != nil {
			fmt.Printf("nacos register instance failed;name: %s ;error: %w", grpcCfg.ServiceName, err)
		} else {
			fmt.Printf("nacos register instance success;name: %s ;", grpcCfg.ServiceName)
		}
	}
	mgr := &common.NacosMgr{
		Client: client,
	}
	if cfg.CfgNameSpaceId != "" { // 是否开启配置中心
		cfgMgr, er := common.NewConfigMgr(&cfg)
		if er != nil {
			fmt.Printf("nacos register instance failed;name: %s ;error: %w", grpcCfg.ServiceName, er)
		} else {
			mgr.CfgStore = cfgMgr
		}
	}
	return mgr
}

func GetNacosService(serviceName string, cfg config.Nacos, grpcCfg config.GRPC) *common.NacosService {
	ip, port, err := GetNacos(cfg, grpcCfg).FindInstance(cfg.ClusterName, cfg.GroupName, serviceName)
	if err != nil {
		return nil
	}
	return &common.NacosService{Ip: ip, Port: port, ServiceName: serviceName}
}

func GetNacosGrpcConn(ns *common.NacosService) (conn *grpc.ClientConn, err error) {
	if ns == nil {
		return nil, errors.New("can not find any service")
	}
	key := fmt.Sprintf("%s_%d", ns.Ip, ns.Port)
	if con, ok := grpcConns.Load(key); ok {
		return con.(*grpc.ClientConn), nil
	} else {
		conn = grpcConn(ns, 0)
		if conn != nil {
			grpcConns.Store(key, conn)
			go grpcStayConnected(key, conn)
			return
		} else {
			return nil, errors.New("connect to grpc server failed")
		}
	}
}

// grpc最多重连2次
func grpcConn(ns *common.NacosService, cnt int) (conn *grpc.ClientConn) {
	if cnt >= 3 {
		return nil
	}
	conn, err := grpc.Dial(fmt.Sprintf("%s:%d", ns.Ip, ns.Port), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return grpcConn(ns, cnt+1)
	}
	return
}

// grpc连接自动检测是否已断开，连接最多维持1小时
func grpcStayConnected(key string, conn *grpc.ClientConn) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Hour)
	defer func() {
		cancel()
		if conn != nil {
			_ = conn.Close()
		}
	}()

	for {
		state := conn.GetState()
		switch state {
		case connectivity.Idle:
			//conn.Connect()
		case connectivity.Shutdown:
			grpcConns.Delete(key)
			return
		}
		if conn.WaitForStateChange(ctx, state) {
			if state == connectivity.Ready && conn.GetState() == connectivity.Idle { // 由ready变回idle一般为服务器断开连接
				grpcConns.Delete(key)
				return
			}
		} else {
			grpcConns.Delete(key)
			return
		}
	}
}

// 开始
func GetNacos(cfg config.Nacos, grpcCfg config.GRPC) *common.NacosMgr {
	if nacosMgr == nil {
		mu.Lock()
		defer mu.Unlock()
		if nacosMgr == nil {
			nacosMgr = BeginNacos(cfg, grpcCfg)
		}
	}
	return nacosMgr
}
