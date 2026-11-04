package config

import (
	"github.com/Neo-Nedio/Lime-server/pkg/configcenter"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	zrpc.RpcServerConf
	ConfigCenter configcenter.ConfigCenter
}
