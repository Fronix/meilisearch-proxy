package caching

import (
	"context"
	"time"

	"github.com/dgraph-io/ristretto"
	"github.com/eko/gocache/lib/v4/cache"
	"github.com/eko/gocache/lib/v4/store"
	ristretto_store "github.com/eko/gocache/store/ristretto/v4"

	redis_store "github.com/eko/gocache/store/redis/v4"
	"github.com/maxroll-media-group/meilisearch-proxy/pkg/config"
	"github.com/maxroll-media-group/meilisearch-proxy/pkg/logger"
	"github.com/redis/go-redis/v9"
)

func GetMemoryCache(config *config.CacheConfig) *cache.Cache[string] {

	ristrettoCache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: 1000,
		MaxCost:     100,
		BufferItems: 64,
	})
	if err != nil {
		panic(err)
	}
	ristrettoStore := ristretto_store.NewRistretto(ristrettoCache, store.WithExpiration(config.TTL*time.Second))

	cacheManager := cache.New[string](ristrettoStore)

	return cacheManager
}

func NewCache(ctx context.Context, config *config.CacheConfig) *cache.Cache[string] {
	logger := logger.GetLogger()

	logger.Info().Msgf("Creating cache with engine: %s, expiration: %d seconds", config.Engine, config.TTL)

	if config.Engine == "memory" {
		logger.Info().Msg("Using memory cache")
		return GetMemoryCache(config)
	} else if config.Engine == "redis" {

		opts, err := redis.ParseURL(config.Url)
		if err != nil {
			logger.Fatal().Msgf("Error parsing Redis URL: %s", err)
		}

		logger.Info().Msgf("Using Redis cache with URL: %s", config.Url)

		redis := redis.NewClient(opts)
		redisStore := redis_store.NewRedis(redis, store.WithExpiration(config.TTL*time.Second))

		status := redis.Ping(ctx)

		if status.Err() != nil {

			logger.Error().Msg("Redis not available, falling back to memory cache")
			return GetMemoryCache(config)
		}

		cacheManager := cache.New[string](redisStore)

		return cacheManager
	}

	panic("Unknown cache engine")
}
