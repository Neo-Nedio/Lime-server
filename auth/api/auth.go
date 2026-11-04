// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/Neo-Nedio/Lime-server/auth/api/internal/config"
	"github.com/Neo-Nedio/Lime-server/auth/api/internal/handler"
	"github.com/Neo-Nedio/Lime-server/auth/api/internal/svc"

	"github.com/Neo-Nedio/Lime-server/pkg/configcenter"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/rest"
)

var configFile = flag.String("f", "auth/api/etc/auth-api.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)
	if err := configcenter.Load(&c); err != nil {
		log.Fatalf("加载远程配置失败: %v", err)
	}

	server := rest.MustNewServer(c.RestConf)
	defer server.Stop()

	ctx := svc.NewServiceContext(c)
	handler.RegisterHandlers(server, ctx)

	fmt.Printf("Starting server at %s:%d...\n", c.Host, c.Port)
	server.Start()
}
