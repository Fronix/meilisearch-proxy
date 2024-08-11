package config_test

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/maxroll-media-group/meilisearch-proxy/pkg/config"
)

var _ = Describe("Config", func() {

	Context("LoadConfig Memory", func() {
		It("should load the config", func() {
			// set fake env variables
			os.Setenv("CACHE_ENGINE", "memory")
			os.Setenv("CACHE_TTL", "300")
			os.Setenv("MEILISEARCH_HOST", "http://localhost:7700")
			os.Setenv("MEILISEARCH_MASTER_KEY", "masterKey")
			os.Setenv("PROXY_MASTER_KEY", "proxyMasterKey")
			os.Setenv("PORT", "8080")

			cfg, err := config.LoadConfig(true)

			Expect(err).To(BeNil())

			expectedConfig := &config.Config{
				MeilisearchHost:        "http://localhost:7700",
				MeilisearchMasterKey:   "masterKey",
				ProxyMasterKey:         "proxyMasterKey",
				ProxyMasterKeyOverride: false,
				Port:                   "8080",
				CacheConfig: &config.CacheConfig{
					TTL:    300,
					Engine: "memory",
					Url:    "",
				},
			}

			Expect(cfg).To(Equal(expectedConfig))
		})
	})

	// cleanup env vars
	AfterEach(func() {
		os.Unsetenv("CACHE_ENGINE")
		os.Unsetenv("CACHE_TTL")
		os.Unsetenv("MEILISEARCH_HOST")
		os.Unsetenv("MEILISEARCH_MASTER_KEY")
		os.Unsetenv("PROXY_MASTER_KEY")
		os.Unsetenv("PORT")

	})

})
