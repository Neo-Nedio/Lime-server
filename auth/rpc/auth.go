package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/Neo-Nedio/Lime-server/auth/rpc/internal/config"
	"github.com/Neo-Nedio/Lime-server/auth/rpc/internal/server"
	"github.com/Neo-Nedio/Lime-server/auth/rpc/internal/svc"
	"github.com/Neo-Nedio/Lime-server/auth/rpc/pb"

	"github.com/Neo-Nedio/Lime-server/pkg/configcenter"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "etc/auth.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)
	if err := configcenter.Load(&c); err != nil {
		log.Fatalf("加载远程配置失败: %v", err)
	}
	ctx := svc.NewServiceContext(c)

	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		pb.RegisterAuthServer(grpcServer, server.NewAuthServer(ctx))

		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})
	defer s.Stop()

	fmt.Printf("Starting rpc server at %s...\n", c.ListenOn)
	s.Start()
}
