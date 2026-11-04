package svc

import (
	"github.com/Neo-Nedio/Lime-server/auth/models"
	"github.com/Neo-Nedio/Lime-server/auth/rpc/internal/config"
	"github.com/zeromicro/go-zero/core/stores/postgres"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

type ServiceContext struct {
	Config config.Config
	*redis.Redis
	models.UserModel
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config:    c,
		Redis:     redis.MustNewRedis(c.Redis),
		UserModel: models.NewUserModel(postgres.New(c.PostgreSQL.DSN)),
	}
}
