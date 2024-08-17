package main

import (
	"github.com/maxroll-media-group/meilisearch-proxy/pkg/config"
	"github.com/maxroll-media-group/meilisearch-proxy/pkg/logger"
	"github.com/maxroll-media-group/meilisearch-proxy/pkg/proxy"
)

func main() {

	logger := logger.GetLogger()
	config, err := config.LoadConfig(false)

	if err != nil {
		logger.Fatal().Msgf("Error loading config: %s", err)
	}

	proxy := proxy.NewProxy(config)

	proxy.Listen()
}
