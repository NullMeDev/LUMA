package proxy

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"universal-checker/pkg/types"
)

// ScrapeSources are predefined proxy sources for auto-scraping
var ScrapeSources = []string{
	"https://www.proxy-list.download/api/v1/get?type=http",
	"https://www.proxy-list.download/api/v1/get?type=https",
	"https://www.proxy-list.download/api/v1/get?type=socks4",
	"https://www.proxy-list.download/api/v1/get?type=socks5",
}

// Scraper scrapes and validates proxies concurrently
type Scraper struct {
	Config *types.CheckerConfig
}

// NewScraper creates a new proxy scraper
func NewScraper(config *types.CheckerConfig) *Scraper {
	return &Scraper{
		Config: config,
	}
}

// ScrapeAndValidate scrapes proxies from sources and validates them
func (s *Scraper) ScrapeAndValidate() ([]types.Proxy, error) {
	proxiesChan := make(chan types.Proxy, 100)
	var wg sync.WaitGroup

	// Scrape proxies
	for _, source := range ScrapeSources {
		wg.Add(1)
		go func(source string) {
			defer wg.Done()

			if err := s.scrapeSource(source, proxiesChan); err != nil {
				log.Printf("Error scraping source %s: %v", source, err)
			}
		}(source)
	}

	// Wait for all scraping to finish
	go func() {
		wg.Wait()
		close(proxiesChan)
	}()

	// Validate proxies
	proxies := []types.Proxy{}
	for proxy := range proxiesChan {
		if s.validateProxy(&proxy) {
			proxies = append(proxies, proxy)
		}
	}

	return proxies, nil
}

// scrapeSource scrapes proxies from a given source
func (s *Scraper) scrapeSource(source string, proxiesChan chan<- types.Proxy) error {
	response, err := http.Get(source)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	data, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		fields := strings.Split(strings.TrimSpace(line), ":")
		if len(fields) != 2 {
			continue
		}

		proxy := types.Proxy{
			Host: fields[0],
			Port: s.toInt(fields[1]),
			Type: s.detectProxyType(source),
		}
		proxiesChan <- proxy
	}

	return nil
}

// validateProxy validates the proxy by checking connectivity
func (s *Scraper) validateProxy(proxy *types.Proxy) bool {
	client := &http.Client{
		Timeout: time.Duration(s.Config.ProxyTimeout) * time.Millisecond,
		Transport: &http.Transport{
			Proxy: s.proxyURL(proxy),
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	start := time.Now()
	response, err := client.Get("https://www.google.com")
	if err != nil {
		return false
	}
	defer response.Body.Close()

	proxy.Latency = int(time.Since(start).Milliseconds())
	proxy.Working = true
	proxy.LastTest = time.Now()

	return true
}

// proxyURL generates a URL for the proxy
func (s *Scraper) proxyURL(proxy *types.Proxy) func(*http.Request) (*url.URL, error) {
	return func(_ *http.Request) (*url.URL, error) {
		return url.Parse(fmt.Sprintf("%s://%s:%d", string(proxy.Type), proxy.Host, proxy.Port))
	}
}

// detectProxyType detects the proxy type from the source URL
func (s *Scraper) detectProxyType(source string) types.ProxyType {
	switch {
	case strings.Contains(source, "socks4"):
		return types.ProxyTypeSOCKS4
	case strings.Contains(source, "socks5"):
		return types.ProxyTypeSOCKS5
	case strings.Contains(source, "https"):
		return types.ProxyTypeHTTPS
	default:
		return types.ProxyTypeHTTP
	}
}

// toInt converts a string to an integer
func (s *Scraper) toInt(value string) int {
	if number, err := strconv.Atoi(value); err == nil {
		return number
	}
	return 0
}
