package main

import (
	"github.com/maxroll-media-group/meilisearch-proxy/pkg/config"
	"github.com/maxroll-media-group/meilisearch-proxy/pkg/proxy"
)

func main() {

	config, err := config.LoadConfig(false)

	if err != nil {
		panic(err)
	}

	proxy := proxy.NewProxy(config)

	proxy.Listen()
}
