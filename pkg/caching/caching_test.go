package caching_test

import (
	"context"

	"github.com/alicebob/miniredis/v2"
	"github.com/maxroll-media-group/meilisearch-proxy/pkg/caching"
	"github.com/maxroll-media-group/meilisearch-proxy/pkg/config"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Caching", func() {
	ctx := context.Background()
	// Describe("NewCache with Memory engine", func() {
	// 	It("should create a memory cache and store/retrieve values", func() {
	// 		cacheConfig := &config.CacheConfig{
	// 			Engine: "memory",
	// 			TTL:    10,
	// 		}
	// 		cacheMgr = caching.NewCache(ctx, cacheConfig)

	// 		err := cacheMgr.Set(ctx, "key", "value")
	// 		Expect(err).To(BeNil())

	// 		val, err := cacheMgr.Get(ctx, "key")
	// 		Expect(err).To(BeNil())
	// 		Expect(val).To(Equal("value"))
	// 	})
	// })

	Describe("NewCache with Redis engine", func() {
		var miniRedis *miniredis.Miniredis
		var addr string

		BeforeEach(func() {
			var err error
			miniRedis, err = miniredis.Run()
			addr = miniRedis.Addr()
			Expect(err).To(BeNil())
		})

		AfterEach(func() {
			miniRedis.Close()
		})

		It("should create a Redis cache and store/retrieve values", func() {
			cacheConfig := &config.CacheConfig{
				Engine: "redis",
				Url:    "redis://" + addr,
				TTL:    10,
			}
			cacheMgr := caching.NewCache(ctx, cacheConfig)

			err := cacheMgr.Set(ctx, "key", "value")
			Expect(err).To(BeNil())

			val, err := cacheMgr.Get(ctx, "key")
			Expect(err).To(BeNil())
			Expect(val).To(Equal("value"))
		})

		// It("should fall back to memory cache if Redis is unavailable", func() {

		// 	miniRedis.Close() // Simulate Redis being unavailable

		// 	cacheConfig := &config.CacheConfig{
		// 		Engine: "redis",
		// 		Url:    "redis://" + addr,
		// 		TTL:    10,
		// 	}
		// 	cacheMgr := caching.NewCache(ctx, cacheConfig)

		// 	err := cacheMgr.Set(ctx, "fallback-key", "fallback-value")
		// 	Expect(err).To(BeNil())

		// 	val, err := cacheMgr.Get(ctx, "fallback-key")
		// 	Expect(err).To(BeNil())
		// 	Expect(val).To(Equal("fallback-value"))
		// })
	})

	Describe("NewCache with an unknown engine", func() {
		It("should panic with an unknown cache engine", func() {
			cacheConfig := &config.CacheConfig{
				Engine: "unknown",
				TTL:    10,
			}
			Expect(func() {
				caching.NewCache(ctx, cacheConfig)
			}).To(Panic())
		})
	})
})
