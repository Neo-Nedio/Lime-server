// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package config

import (
	"github.com/Neo-Nedio/Lime-server/pkg/configcenter"
	"github.com/zeromicro/go-zero/rest"
)

type Config struct {
	rest.RestConf
	ConfigCenter configcenter.ConfigCenter
	JWT          struct {
		AccessSecret string
		AccessExpire int64
	}
}
