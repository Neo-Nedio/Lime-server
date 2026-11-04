package config

import (
	"github.com/Neo-Nedio/Lime-server/pkg/configcenter"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	zrpc.RpcServerConf
	ConfigCenter configcenter.ConfigCenter

	PostgreSQL struct {
		DSN          string
		MaxOpenConns int
		MaxIdleConns int
	}

	Redis redis.RedisConf

	JWT struct {
		AccessSecret  string
		RefreshSecret string
		AccessExpire  int64
		RefreshExpire int64
	}

	Mailer struct {
		Host     string
		Port     int
		Username string
		Password string
		From     string
	}
}
