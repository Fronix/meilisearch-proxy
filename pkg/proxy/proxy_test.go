package proxy_test

import (
	"fmt"
	"io"
	"net"
	"net/http"

	"github.com/alicebob/miniredis/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/maxroll-media-group/meilisearch-proxy/pkg/config"
	"github.com/maxroll-media-group/meilisearch-proxy/pkg/proxy"
)

const testJSON = `{"name":"test"}`
const testIndexJSON = `
{
	"results": [
		{
			"uid": "profiles",
			"createdAt": "2024-08-07T22:32:45.025568204Z",
			"updatedAt": "2024-08-11T14:39:20.410931753Z",
			"primaryKey": "id"
		}]
}
`

var _ = Describe("Proxy", Ordered, func() {

	var redis *miniredis.Miniredis
	var fakeMeilisearch *http.Server
	var addr string
	var proxyUrl string

	// start a fake Meilisearch server

	BeforeAll(func() {
		redis, _ = miniredis.Run()
		addr = "redis://" + redis.Addr()
		proxyListener, err := net.Listen("tcp", ":0")
		if err != nil {
			panic(err)
		}

		meilisearchListener, err := net.Listen("tcp", ":0")
		if err != nil {
			panic(err)
		}

		proxyUrl = fmt.Sprintf("http://localhost:%d", proxyListener.Addr().(*net.TCPAddr).Port)

		mux := http.NewServeMux()
		mux.HandleFunc("/indexes/test/search", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(testJSON))
			w.WriteHeader(http.StatusOK)
		})

		mux.HandleFunc("/indexes", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(testIndexJSON))
			w.WriteHeader(http.StatusOK)
		})

		fakeMeilisearch = &http.Server{
			Addr:    fmt.Sprintf("localhost:%d", meilisearchListener.Addr().(*net.TCPAddr).Port),
			Handler: mux,
		}

		go fakeMeilisearch.ListenAndServe()

		cfg := &config.Config{
			MeilisearchHost:        "http://localhost:7777",
			MeilisearchMasterKey:   "masterKey",
			ProxyMasterKey:         "proxyMasterKey",
			ProxyMasterKeyOverride: false,
			Port:                   fmt.Sprintf("%d", proxyListener.Addr().(*net.TCPAddr).Port),
			CacheConfig: &config.CacheConfig{
				TTL:    300,
				Engine: "redis",
				Url:    addr,
			},
		}

		proxyServer := proxy.NewProxy(cfg)
		go proxyServer.Listen()
	})

	Context("ProxyCalls", func() {

		It("should proxy POST indexes", func() {
			// create a request
			req, _ := http.NewRequest("POST", proxyUrl+"/indexes/test/search", nil)

			// req should contain "{"name":"test"}"
			resp, err := http.DefaultClient.Do(req)
			Expect(err).To(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			resBody, err := io.ReadAll(resp.Body)

			Expect(err).To(BeNil())

			// resp should contain "{"name":"test"}"
			Expect(resBody).To(Equal([]byte(testJSON)))
		})
	})

	It("should have test data in cache", func() {

		// create a request
		_, _ = http.NewRequest("GET", proxyUrl+"/indexes/test/search", nil)

		cached, err := redis.Get("e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855")

		Expect(err).To(BeNil())

		redis.Set("test", "dsfdsf")

		Expect(cached).To(Equal(testJSON))
	})

	It("should simply proxy other requests", func() {

		// create a request
		req, _ := http.NewRequest("GET", proxyUrl+"/indexes", nil)

		resp, err := http.DefaultClient.Do(req)
		Expect(err).To(BeNil())

		Expect(resp.StatusCode).To(Equal(http.StatusOK))

		resBody, err := io.ReadAll(resp.Body)

		Expect(err).To(BeNil())

		Expect(resBody).To(Equal([]byte(testIndexJSON)))
	})

	AfterAll(func() {
		fakeMeilisearch.Close()
		redis.Close()
	})

})
