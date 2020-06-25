package main

import (
	"flag"
	"github.com/spf13/viper"
	"github.com/Bundle-App/blockatlas/config"
	"github.com/Bundle-App/blockatlas/pkg/logger"
	"github.com/Bundle-App/blockatlas/storage"
	"github.com/Bundle-App/blockatlas/syncmarkets"
	"path/filepath"
)

const (
	defaultConfigPath = "../../config.yml"
)

var (
	confPath string
	cache    *storage.Storage
)

func init() {
	cache = storage.New()

	flag.StringVar(&confPath, "c", defaultConfigPath, "config file for observer")
	flag.Parse()

	confPath, err := filepath.Abs(confPath)
	if err != nil {
		logger.Fatal(err)
	}

	config.LoadConfig(confPath)
	logger.InitLogger()

	host := viper.GetString("storage.redis")
	err = cache.Init(host)
	if err != nil {
		logger.Fatal(err)
	}
}

func main() {
	syncmarkets.InitRates(cache)
	syncmarkets.InitMarkets(cache)
	<-make(chan struct{})
}
