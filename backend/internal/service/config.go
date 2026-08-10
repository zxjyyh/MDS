package service

import "github.com/zxjyyh/mds/internal/config"

var appConfig *config.Config

func InitConfig(cfg *config.Config) {
	appConfig = cfg
}

func GetConfig() *config.Config {
	return appConfig
}
