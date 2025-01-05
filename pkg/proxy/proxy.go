package proxy

import (
	"compress/flate"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/eko/gocache/lib/v4/cache"
	"github.com/eko/gocache/lib/v4/store"
	"github.com/maxroll-media-group/meilisearch-proxy/pkg/caching"
	"github.com/maxroll-media-group/meilisearch-proxy/pkg/config"
	"github.com/maxroll-media-group/meilisearch-proxy/pkg/logger"
	"github.com/maxroll-media-group/meilisearch-proxy/pkg/util"
	"github.com/rs/zerolog"
)

type Proxy struct {
	source *url.URL

	proxy       *httputil.ReverseProxy
	cache       *cache.Cache[string]
	config      *config.Config
	startupTime time.Time
	context.Context
	zerolog.Logger
}

func NewProxy(config *config.Config) *Proxy {
	logger := logger.GetLogger()

	source, err := url.Parse(config.MeilisearchHost)
	if err != nil {
		logger.Fatal().Msgf("Error parsing Meilisearch host: %s", err)
	}

	logger.Info().Msgf("Meilisearch host: %s", source.String())

	ctx := context.Background()
	proxy := httputil.NewSingleHostReverseProxy(source)

	proxy.Director = func(req *http.Request) {
		req.URL.Scheme = source.Scheme
		req.URL.Host = source.Host
		req.URL.Path = util.SingleJoiningSlash(source.Path, req.URL.Path)
		if source.RawQuery == "" || req.URL.RawQuery == "" {
			req.URL.RawQuery = source.RawQuery + req.URL.RawQuery
		} else {
			req.URL.RawQuery = source.RawQuery + "&" + req.URL.RawQuery
		}
		req.Host = source.Host // Ensure the Host header is set
		if _, ok := req.Header["User-Agent"]; !ok {
			req.Header.Set("User-Agent", "")
		}

		req.Header.Set("Accept-Encoding", "deflate,gzip")

	}
	cache := caching.NewCache(ctx, config.CacheConfig)

	if config.AutoRestartInterval > 0 {
		logger.Info().Msgf("Auto restart interval set to %s", config.AutoRestartInterval)
	}

	return &Proxy{
		source:      source,
		proxy:       proxy,
		cache:       cache,
		config:      config,
		Context:     ctx,
		Logger:      logger,
		startupTime: time.Now(),
	}
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	if r.URL.Path == "/health" {
		// TODO: Implement a more robust health check
		// challenge: the health check should check if the underlying meilisearch is reachable
		// and if the cache is working. However, stale cache entries should still be used to avoid downtime.

		autoRestartInterval := p.config.AutoRestartInterval

		if autoRestartInterval > 0 {
			// auto restart interval is time (1h, 1d, 1w, 1m, 1y)
			// if the proxy is running for more than the interval, it should restart itself
			// this is to avoid memory leaks and other issues
			if time.Since(p.startupTime) > autoRestartInterval {
				p.Logger.Warn().Msgf("Restarting proxy after running for %s", autoRestartInterval)

				// fail the health check
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}

		}

		w.WriteHeader(http.StatusOK)
		return
	}

	if regexp.MustCompile(`^/indexes/[^/]+/search$`).MatchString(r.URL.Path) {
		p.handleSearch(w, r)
	} else if regexp.MustCompile(`^/purge(/|$)`).MatchString(r.URL.Path) {
		p.handlePurge(w, r)
	} else {
		p.handleDefault(w, r)
	}
}

func (p *Proxy) handleSearch(w http.ResponseWriter, r *http.Request) {
	indexName := util.ExtractIndexName(r.URL.Path)

	cacheKey := sha256.Sum256([]byte(r.URL.Path))
	if r.Method == http.MethodPost {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			log.Printf("Error reading request body: %s", err)
			http.Error(w, "Error reading request body", http.StatusInternalServerError)
			return
		}
		cacheKey = sha256.Sum256(append(cacheKey[:], body...))
		r.Body = io.NopCloser(strings.NewReader(string(body))) // Reset the body after reading
	}

	cacheKeyString := fmt.Sprintf("%x", cacheKey)

	// Check if response is in cache
	if response, err := p.cache.Get(r.Context(), cacheKeyString); err == nil {
		p.Logger.Info().Msgf("[%s] Cache hit for %s, key: %s", indexName, r.URL.Path, cacheKeyString)

		w.Write([]byte(response))
		return
	}

	p.Logger.Info().Msgf("[%s] Cache miss for %s, key: %s", indexName, r.URL.Path, cacheKeyString)

	// Capture the response for caching
	recorder := httptest.NewRecorder()
	responseBody, err := p.recordProxyRequest(r, recorder)

	// Write the captured response to the original response writer
	for k, v := range recorder.Header() {

		if k == "Content-Encoding" {
			continue
		}

		w.Header()[k] = v
	}
	w.WriteHeader(recorder.Code)
	w.Write(responseBody)

	// never cache an error response or an empty response
	if recorder.Code != http.StatusOK || len(responseBody) == 0 {
		p.Logger.Debug().Msgf("Not caching response for %s, key: %s", r.URL.Path, cacheKeyString)
		p.Logger.Warn().Msgf("[%s] Could not reach upstream Meilisearch. Path: %s, key: %s", indexName, r.URL.Path, cacheKeyString)
		return
	}

	// Store response in cache
	p.Logger.Debug().Msgf("[%s] Storing response in cache for %s, key: %s", indexName, r.URL.Path, cacheKeyString)

	err = p.cache.Set(r.Context(), cacheKeyString, string(responseBody[:]), store.WithTags([]string{indexName}))

	if err != nil {
		p.Logger.Error().Msgf("[%s] Error storing response in cache for %s, key: %s: %s", indexName, r.URL.Path, cacheKeyString, err)
	}
}

func (p *Proxy) recordProxyRequest(r *http.Request, recorder *httptest.ResponseRecorder) ([]byte, error) {
	p.Logger.Debug().Msgf("Proxying request to %s", r.URL.String())

	p.proxy.ServeHTTP(recorder, r)

	// Check if the response is compressed
	encoding := recorder.Header().Get("Content-Encoding")
	var body []byte
	var err error

	switch encoding {
	case "gzip":
		p.Logger.Debug().Msg("Decompressing gzip response body")
		gz, err := gzip.NewReader(recorder.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to create gzip reader: %w", err)
		}
		defer gz.Close()
		body, err = io.ReadAll(gz)
		if err != nil {
			return nil, fmt.Errorf("failed to read gzip body: %w", err)
		}
	case "deflate":
		p.Logger.Debug().Msg("Decompressing deflate response body")
		// Decompress deflate
		fl := flate.NewReader(recorder.Body)
		defer fl.Close()
		body, err = io.ReadAll(fl)
		if err != nil {
			return nil, fmt.Errorf("failed to read deflate body: %w", err)
		}
	default:
		p.Logger.Debug().Msg("Using response body as is")
		body = recorder.Body.Bytes()
	}

	return body, nil
}

func (p *Proxy) handleDefault(w http.ResponseWriter, r *http.Request) {
	finalURL := p.source.ResolveReference(r.URL)
	p.Logger.Debug().Msgf("Handling request for %s", finalURL.String())
	p.proxy.ServeHTTP(w, r)
}

func (p *Proxy) handlePurge(w http.ResponseWriter, r *http.Request) {

	indexName := util.ExtractIndexName(r.URL.Path)

	p.Logger.Info().Msg("Cache purge request received")

	if p.config.ProxyPurgeToken != "" {
		token := r.Header.Get("Authorization")

		if token != fmt.Sprintf("Bearer %s", p.config.ProxyPurgeToken) {
			p.Logger.Error().Msg("Unauthorized purge request")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
	}

	err := p.PurgeCache(indexName)

	if err != nil {
		p.Logger.Error().Msgf("Error purging cache: %s", err)
		http.Error(w, "Error purging cache", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (p *Proxy) Listen() {
	mux := http.NewServeMux()

	// mux / with both middlewares
	mux.Handle("/", p.authMiddleware(p.headersMiddleware(p)))

	log.Printf("Starting proxy server on  :%s", p.config.Port)

	http.ListenAndServe(fmt.Sprintf(":%s", p.config.Port), mux)
}

func (p *Proxy) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")

		if p.config.ProxyMasterKeyOverride {
			r.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.config.MeilisearchMasterKey))

			if token != fmt.Sprintf("Bearer %s", p.config.ProxyMasterKey) {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

func (p *Proxy) headersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Meilisearch-Client")
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (p *Proxy) PurgeCache(index string) error {

	// test if the underlying meilisearch is reachable
	_, err := http.Get(p.source.String())
	if err != nil {
		return fmt.Errorf("Error reaching Meilisearch host: %s, refusing to purge cache", err)
	}

	if index != "" {
		p.Logger.Info().Msgf("Purging cache for index: %s", index)
		return p.cache.Invalidate(p.Context, store.WithInvalidateTags([]string{index}))
	} else {
		p.Logger.Info().Msg("Purging full cache for all indexes")

		return p.cache.Clear(p.Context)
	}
}

func (p *Proxy) GetCache() *cache.Cache[string] {
	return p.cache
}
