package proxy

import (
	"context"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	structuredLogger "universal-checker/internal/logger"
	"universal-checker/pkg/types"
	"universal-checker/pkg/utils"
	"golang.org/x/net/proxy"
)

// ScrapeSources are predefined proxy sources for auto-scraping
var ScrapeSources = []string{
	"https://www.proxy-list.download/api/v1/get?type=http",
	"https://www.proxy-list.download/api/v1/get?type=https",
	"https://www.proxy-list.download/api/v1/get?type=socks4",
	"https://www.proxy-list.download/api/v1/get?type=socks5",
	"https://raw.githubusercontent.com/TheSpeedX/PROXY-List/master/http.txt",
	"https://raw.githubusercontent.com/TheSpeedX/PROXY-List/master/socks4.txt",
	"https://raw.githubusercontent.com/TheSpeedX/PROXY-List/master/socks5.txt",
	"https://raw.githubusercontent.com/clarketm/proxy-list/master/proxy-list-raw.txt",
}

// Scraper scrapes and validates proxies concurrently
type Scraper struct {
	Config *types.CheckerConfig
	Logger *structuredLogger.StructuredLogger
}

// NewScraper creates a new proxy scraper
func NewScraper(config *types.CheckerConfig, logger *structuredLogger.StructuredLogger) *Scraper {
	return &Scraper{
		Config: config,
		Logger: logger,
	}
}

// ScrapeAndValidate scrapes proxies from sources and validates them
func (s *Scraper) ScrapeAndValidate() ([]types.Proxy, error) {
	correlationID := utils.GenerateCorrelationID()
	s.Logger.Info("Starting proxy scraping from multiple sources", map[string]interface{}{
		"total_sources": len(ScrapeSources),
		"sources": ScrapeSources,
		"correlation_id": correlationID,
	})
	proxiesChan := make(chan types.Proxy, 100)
	var wg sync.WaitGroup

	// Scrape proxies
	for _, source := range ScrapeSources {
		wg.Add(1)
		go func(source string) {
			defer wg.Done()

			if err := s.scrapeSourceWithTimeout(source, proxiesChan); err != nil {
				s.Logger.Error("Error scraping source", err, map[string]interface{}{"source": source, "correlation_id": correlationID})
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
	totalProxies := 0
	validProxies := 0
	
	for proxy := range proxiesChan {
		totalProxies++
		if s.validateProxy(&proxy) {
			validProxies++
			s.Logger.LogProxyEvent("validation_success", proxy, map[string]interface{}{"correlation_id": correlationID})
			proxies = append(proxies, proxy)
		} else {
			s.Logger.LogProxyEvent("validation_failure", proxy, map[string]interface{}{"reason": "Timeout or unreachable", "correlation_id": correlationID})
		}
	}

	// Log scraping summary
	var successRate float64
	if totalProxies > 0 {
		successRate = float64(validProxies) / float64(totalProxies) * 100
	}
	s.Logger.Info("Proxy scraping completed", map[string]interface{}{
		"total_scraped": totalProxies,
		"valid_proxies": validProxies,
		"success_rate": fmt.Sprintf("%.2f%%", successRate),
		"correlation_id": correlationID,
	})

	// Fallback mechanism - continue operation even if most proxies are invalid
	if validProxies == 0 {
		s.Logger.Warn("No valid proxies found, tool will continue without proxy support", map[string]interface{}{"correlation_id": correlationID})
		return proxies, fmt.Errorf("no valid proxies found")
	} else if successRate < 10 {
		s.Logger.Warn("Low proxy success rate detected, but continuing with available proxies", map[string]interface{}{
			"success_rate": fmt.Sprintf("%.2f%%", successRate),
			"correlation_id": correlationID,
		})
	}

	return proxies, nil
}

// scrapeSourceWithTimeout scrapes proxies using a timeout
func (s *Scraper) scrapeSourceWithTimeout(source string, proxiesChan chan<- types.Proxy) error {
	correlationID := utils.GenerateCorrelationID()
	s.Logger.Info("Scraping proxy source", map[string]interface{}{"source": source, "correlation_id": correlationID})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create HTTP client with timeout for scraping
	client := &http.Client{}

	req, err := http.NewRequestWithContext(ctx, "GET", source, nil)
	if err != nil {
		s.Logger.Error("Failed to create request", err, map[string]interface{}{"source": source, "correlation_id": correlationID})
		return err
	}

	response, err := client.Do(req)
	if err != nil {
		s.Logger.Error("Failed to fetch proxy source", err, map[string]interface{}{"source": source, "correlation_id": correlationID})
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		err = fmt.Errorf("HTTP %d: %s", response.StatusCode, response.Status)
		s.Logger.Error("Non-200 response from proxy source", err, map[string]interface{}{"source": source, "status_code": response.StatusCode, "correlation_id": correlationID})
		return err
	}

	data, err := ioutil.ReadAll(response.Body)
	if err != nil {
		s.Logger.Error("Failed to read response body", err, map[string]interface{}{"source": source, "correlation_id": correlationID})
		return err
	}

	lines := strings.Split(string(data), "\n")
	scrapedCount := 0
	for _, line := range lines {
		fields := strings.Split(strings.TrimSpace(line), ":")
		if len(fields) != 2 {
			continue
		}

		// Validate IP address format
		if !s.isValidIP(fields[0]) {
			continue
		}

		port := s.toInt(fields[1])
		if port == 0 || port > 65535 {
			continue
		}

		proxy := types.Proxy{
			Host:      fields[0],
			Port:      port,
			Type:      s.detectProxyType(source),
			CreatedAt: time.Now(),
		}
		proxiesChan <- proxy
		scrapedCount++
	}

	s.Logger.Info("Proxy source scraping completed", map[string]interface{}{
		"source": source,
		"scraped_count": scrapedCount,
		"correlation_id": correlationID,
	})
	return nil
}

// validateProxy validates the proxy by checking connectivity
func (s *Scraper) validateProxy(proxy *types.Proxy) bool {
	// Multiple test URLs for better validation
	testURLs := []string{
		"https://www.google.com",
		"https://httpbin.org/ip",
		"https://www.wikipedia.org",
	}

	for _, testURL := range testURLs {
		if s.testProxyWithURL(proxy, testURL) {
			return true
		}
	}
	return false
}

// testProxyWithURL tests a proxy against a specific URL
func (s *Scraper) testProxyWithURL(proxy *types.Proxy, testURL string) bool {
	var client *http.Client
	
	// Handle different proxy types
	switch proxy.Type {
	case types.ProxyTypeSOCKS4, types.ProxyTypeSOCKS5:
		client = s.createSOCKSClient(proxy)
	default:
		client = s.createHTTPClient(proxy)
	}
	
	if client == nil {
		return false
	}

	start := time.Now()
	responseChan := make(chan bool, 1)
	errorChan := make(chan error, 1)
	
	go func() {
		response, err := client.Get(testURL)
		if err != nil {
			errorChan <- err
			return
		}
		defer response.Body.Close()
		
		if response.StatusCode == http.StatusOK {
			proxy.Latency = int(time.Since(start).Milliseconds())
			proxy.Working = true
			proxy.LastTest = time.Now()
			responseChan <- true
		} else {
			errorChan <- fmt.Errorf("HTTP %d", response.StatusCode)
		}
	}()

	select {
	case valid := <-responseChan:
		return valid
	case err := <-errorChan:
		s.Logger.Debug("Proxy validation failed", map[string]interface{}{
			"proxy": fmt.Sprintf("%s:%d", proxy.Host, proxy.Port),
			"test_url": testURL,
			"error": err.Error(),
		})
		return false
	case <-time.After(10 * time.Second):
		s.Logger.Debug("Proxy validation timeout", map[string]interface{}{
			"proxy": fmt.Sprintf("%s:%d", proxy.Host, proxy.Port),
			"test_url": testURL,
			"timeout": "10s",
		})
		return false
	}
}

// proxyURL generates a URL for the proxy
func (s *Scraper) proxyURL(proxy *types.Proxy) func(*http.Request) (*url.URL, error) {
	return func(_ *http.Request) (*url.URL, error) {
		return url.Parse(fmt.Sprintf("%s://%s:%d", string(proxy.Type), proxy.Host, proxy.Port))
	}
}

// createHTTPClient creates an HTTP client for HTTP/HTTPS proxies
func (s *Scraper) createHTTPClient(proxy *types.Proxy) *http.Client {
	return &http.Client{
		Timeout: time.Duration(s.Config.ProxyTimeout) * time.Millisecond,
		Transport: &http.Transport{
			Proxy: s.proxyURL(proxy),
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
}

// createSOCKSClient creates an HTTP client for SOCKS proxies
func (s *Scraper) createSOCKSClient(proxyInstance *types.Proxy) *http.Client {
	proxyAddr := fmt.Sprintf("%s:%d", proxyInstance.Host, proxyInstance.Port)
	
	switch proxyInstance.Type {
	case types.ProxyTypeSOCKS4:
		// Note: golang.org/x/net/proxy doesn't support SOCKS4, fallback to HTTP proxy approach
		s.Logger.Debug("SOCKS4 not fully supported, attempting HTTP proxy approach", map[string]interface{}{
			"proxy": proxyAddr,
		})
		return &http.Client{
			Timeout: time.Duration(s.Config.ProxyTimeout) * time.Millisecond,
			Transport: &http.Transport{
				Proxy: func(_ *http.Request) (*url.URL, error) {
					return url.Parse(fmt.Sprintf("socks4://%s", proxyAddr))
				},
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		}
	case types.ProxyTypeSOCKS5:
		dialer, err := proxy.SOCKS5("tcp", proxyAddr, nil, proxy.Direct)
		if err != nil {
			s.Logger.Debug("Failed to create SOCKS5 dialer", map[string]interface{}{
				"proxy": proxyAddr,
				"error": err.Error(),
			})
			return nil
		}
		
		return &http.Client{
			Timeout: time.Duration(s.Config.ProxyTimeout) * time.Millisecond,
			Transport: &http.Transport{
				Dial: dialer.Dial,
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		}
	default:
		return nil
	}
}

// detectProxyType detects the proxy type from the source URL
func (s *Scraper) detectProxyType(source string) types.ProxyType {
	// Convert to lowercase for case-insensitive matching
	sourceLower := strings.ToLower(source)
	
	switch {
	case strings.Contains(sourceLower, "socks4"):
		return types.ProxyTypeSOCKS4
	case strings.Contains(sourceLower, "socks5"):
		return types.ProxyTypeSOCKS5
	case strings.Contains(sourceLower, "https") && !strings.Contains(sourceLower, "http.txt"):
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

// isValidIP checks if a string is a valid IP address
func (s *Scraper) isValidIP(ip string) bool {
	return net.ParseIP(ip) != nil
}

// ScrapeAndValidateWithFallback provides a fallback mechanism that ensures operation continues
func (s *Scraper) ScrapeAndValidateWithFallback() ([]types.Proxy, error) {
	proxies, err := s.ScrapeAndValidate()
	
	// If no proxies found, log warning but don't fail completely
	if err != nil && len(proxies) == 0 {
		s.Logger.Warn("Proxy scraping failed, continuing without proxies", map[string]interface{}{
			"error": err.Error(),
		})
		return []types.Proxy{}, nil // Return empty slice instead of error
	}
	
	return proxies, err
}
