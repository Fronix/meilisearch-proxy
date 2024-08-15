package config

import (
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"github.com/maxroll-media-group/meilisearch-proxy/pkg/logger"
)

type Config struct {
	MeilisearchHost        string
	MeilisearchMasterKey   string
	ProxyMasterKey         string
	ProxyMasterKeyOverride bool
	ProxyPurgeToken        string
	Port                   string
	CacheConfig            *CacheConfig
}

type CacheConfig struct {
	TTL    time.Duration
	Engine string
	Url    string
}

func LoadConfig(skipUrlCheck bool) (*Config, error) {
	logger := logger.GetLogger()

	err := godotenv.Load()
	if err != nil {
		logger.Warn().Msg("Error loading .env file")
	}

	CacheConfig := &CacheConfig{
		TTL:    300,
		Engine: "memory",
		Url:    "",
	}

	if os.Getenv("CACHE_ENGINE") != "" {
		if os.Getenv("CACHE_ENGINE") == "redis" && os.Getenv("CACHE_URL") == "" {
			logger.Fatal().Msg("CACHE_URL is required when using Redis cache")
		}

		CacheConfig.Engine = os.Getenv("CACHE_ENGINE")
		CacheConfig.Url = os.Getenv("CACHE_URL")
	}

	if os.Getenv("CACHE_TTL") != "" {
		ttl, err := strconv.Atoi(os.Getenv("CACHE_TTL"))
		if err != nil {
			logger.Fatal().Msg("CACHE_TTL must be an integer")
		}
		CacheConfig.TTL = time.Duration(ttl)

		if CacheConfig.TTL < 1 {
			logger.Fatal().Msg("CACHE_TTL must be greater than 0")
		}
	}

	config := &Config{
		MeilisearchHost:        os.Getenv("MEILISEARCH_HOST"),
		MeilisearchMasterKey:   os.Getenv("MEILISEARCH_MASTER_KEY"),
		ProxyMasterKey:         os.Getenv("PROXY_MASTER_KEY"),
		ProxyPurgeToken:        os.Getenv("PROXY_PURGE_TOKEN"),
		ProxyMasterKeyOverride: false,
		Port:                   os.Getenv("PORT"),
		CacheConfig:            CacheConfig,
	}

	if config.Port == "" {
		config.Port = "8080"
	}

	override, err := strconv.ParseBool(os.Getenv("PROXY_MASTER_KEY_OVERRIDE"))
	if err == nil {
		config.ProxyMasterKeyOverride = override
	}

	if config.ProxyMasterKey == "" && config.ProxyMasterKeyOverride && config.MeilisearchMasterKey == "" {
		logger.Fatal().Msg("PROXY_MASTER_KEY_OVERRIDE is enabled but PROXY_MASTER_KEY is not set")
	}

	// check if the host is reachable
	if !skipUrlCheck {
		_, err = http.Get(config.MeilisearchHost)

		if err != nil {
			return nil, err
		}
	}

	return config, nil
}
