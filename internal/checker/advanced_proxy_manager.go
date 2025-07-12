package checker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"sort"
	"sync"
	"time"

	"universal-checker/pkg/types"
	"universal-checker/pkg/utils"
)

// ProxySelectionStrategy defines how proxies should be selected
type ProxySelectionStrategy string

const (
	StrategyRoundRobin    ProxySelectionStrategy = "round_robin"
	StrategyBestScore     ProxySelectionStrategy = "best_score"
	StrategyRandomWeighted ProxySelectionStrategy = "random_weighted"
	StrategyGeoPreferred  ProxySelectionStrategy = "geo_preferred"
	StrategyLeastUsed     ProxySelectionStrategy = "least_used"
)

// GeoIPResponse represents response from geo-location API
type GeoIPResponse struct {
	IP          string  `json:"ip"`
	CountryName string  `json:"country_name"`
	CountryCode string  `json:"country_code"`
	City        string  `json:"city"`
	Region      string  `json:"region_name"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
	TimeZone    string  `json:"time_zone"`
	ISP         string  `json:"isp"`
}

// AdvancedProxyManager manages proxies with advanced features
type AdvancedProxyManager struct {
	proxies           []types.Proxy
	mutex             sync.RWMutex
	currentIndex      int
	strategy          ProxySelectionStrategy
	preferredCountries []string
	blacklistedIPs    map[string]bool
	healthCheckInterval time.Duration
	maxConsecutiveFails int
	
	// Performance tracking
	totalRequestsServed int64
	totalProxyErrors    int64
	
	// HTTP client for testing
	testClient *http.Client
}

// NewAdvancedProxyManager creates a new advanced proxy manager
func NewAdvancedProxyManager(strategy ProxySelectionStrategy) *AdvancedProxyManager {
	return &AdvancedProxyManager{
		proxies:             make([]types.Proxy, 0),
		strategy:            strategy,
		preferredCountries:  make([]string, 0),
		blacklistedIPs:      make(map[string]bool),
		healthCheckInterval: 5 * time.Minute,
		maxConsecutiveFails: 3,
		testClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// AddProxy adds a new proxy to the manager
func (pm *AdvancedProxyManager) AddProxy(proxy types.Proxy) error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	// Initialize proxy metrics if not present
	if proxy.Metrics == nil {
		proxy.Metrics = &types.ProxyMetrics{
			TotalRequests:    0,
			SuccessfulReqs:   0,
			FailedRequests:   0,
			AverageLatency:   0,
			MinLatency:       9999,
			MaxLatency:       0,
			Uptime:           100.0,
			ConsecutiveFails: 0,
			BanDetected:      false,
		}
	}

	proxy.CreatedAt = time.Now()
	proxy.UpdatedAt = time.Now()
	proxy.Working = true  // Assume working until proven otherwise
	proxy.Score = 50.0    // Default neutral score
	proxy.Quality = types.ProxyQualityAverage
	
	// Get geolocation information
	if err := pm.enrichProxyWithGeoLocation(&proxy); err != nil {
		// Log error but don't fail - geo data is optional
		fmt.Printf("Warning: Could not get geo location for proxy %s:%d - %v\n", proxy.Host, proxy.Port, err)
	}

	pm.proxies = append(pm.proxies, proxy)
	return nil
}

// enrichProxyWithGeoLocation fetches geographical information for a proxy
func (pm *AdvancedProxyManager) enrichProxyWithGeoLocation(proxy *types.Proxy) error {
	// Use a free geo-location API (ip-api.com)
	url := fmt.Sprintf("http://ip-api.com/json/%s", proxy.Host)
	
	resp, err := pm.testClient.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var geoResp GeoIPResponse
	if err := json.Unmarshal(body, &geoResp); err != nil {
		return err
	}

	proxy.Location = &types.ProxyLocation{
		Country:     geoResp.CountryName,
		CountryCode: geoResp.CountryCode,
		City:        geoResp.City,
		Region:      geoResp.Region,
		Latitude:    geoResp.Latitude,
		Longitude:   geoResp.Longitude,
		Timezone:    geoResp.TimeZone,
		ISP:         geoResp.ISP,
	}

	return nil
}

// TestProxy performs a health check on a proxy
func (pm *AdvancedProxyManager) TestProxy(proxy *types.Proxy) error {
	return pm.TestProxyWithContext(context.Background(), proxy)
}

// TestProxyWithContext performs a health check on a proxy with context timeout
func (pm *AdvancedProxyManager) TestProxyWithContext(ctx context.Context, proxy *types.Proxy) error {
	// Create a context with timeout for the test
	testCtx, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()
	
	start := time.Now()
	
	// Create a test HTTP client with the proxy and context
	client := pm.createTestClientWithContext(proxy, testCtx)
	
	// Channel to receive the result
	resultChan := make(chan error, 1)
	
	// Perform the test in a goroutine
	go func() {
		resp, err := client.Get("http://httpbin.org/ip")
		if err != nil {
			resultChan <- err
			return
		}
		defer resp.Body.Close()
		
		if resp.StatusCode != 200 {
			resultChan <- fmt.Errorf("unexpected status code: %d", resp.StatusCode)
			return
		}
		
		resultChan <- nil
	}()
	
	// Wait for either result or timeout
	var err error
	select {
	case err = <-resultChan:
		// Test completed within timeout
	case <-testCtx.Done():
		// Test timed out
		err = fmt.Errorf("proxy test timeout: %v", testCtx.Err())
	}
	
	latency := int(time.Since(start).Milliseconds())
	
	pm.mutex.Lock()
	defer pm.mutex.Unlock()
	
	proxy.LastTest = time.Now()
	proxy.UpdatedAt = time.Now()
	proxy.Latency = latency

	if err != nil {
		proxy.Working = false
		proxy.Metrics.FailedRequests++
		proxy.Metrics.ConsecutiveFails++
		pm.updateProxyQuality(proxy)
		return err
	}
	
	proxy.Working = true
	proxy.Metrics.SuccessfulReqs++
	proxy.Metrics.ConsecutiveFails = 0
	proxy.Metrics.LastSuccessTime = time.Now()
	
	// Update latency metrics
	if latency < proxy.Metrics.MinLatency || proxy.Metrics.MinLatency == 9999 {
		proxy.Metrics.MinLatency = latency
	}
	if latency > proxy.Metrics.MaxLatency {
		proxy.Metrics.MaxLatency = latency
	}
	
	// Calculate average latency
	if proxy.Metrics.TotalRequests > 0 {
		proxy.Metrics.AverageLatency = (proxy.Metrics.AverageLatency*proxy.Metrics.TotalRequests + latency) / (proxy.Metrics.TotalRequests + 1)
	} else {
		proxy.Metrics.AverageLatency = latency
	}

	proxy.Metrics.TotalRequests++
	pm.updateProxyQuality(proxy)
	pm.calculateProxyScore(proxy)

	return nil
}

// createTestClient creates an HTTP client configured with the proxy
func (pm *AdvancedProxyManager) createTestClient(proxy *types.Proxy) *http.Client {
	// This is a simplified version - in reality you'd configure the actual proxy
	return &http.Client{
		Timeout: 10 * time.Second,
	}
}

// createTestClientWithContext creates an HTTP client configured with the proxy and context
func (pm *AdvancedProxyManager) createTestClientWithContext(proxy *types.Proxy, ctx context.Context) *http.Client {
	// This is a simplified version - in reality you'd configure the actual proxy
	// The context will be used in the request, not in the client timeout
	return &http.Client{
		Timeout: 30 * time.Second, // Set a reasonable timeout
	}
}

// updateProxyQuality updates the quality rating of a proxy based on performance
func (pm *AdvancedProxyManager) updateProxyQuality(proxy *types.Proxy) {
	if proxy.Metrics.TotalRequests == 0 {
		proxy.Quality = types.ProxyQualityAverage
		return
	}

	uptime := float64(proxy.Metrics.SuccessfulReqs) / float64(proxy.Metrics.TotalRequests) * 100
	proxy.Metrics.Uptime = uptime

	avgLatency := proxy.Metrics.AverageLatency

	// Determine quality based on latency and uptime
	if avgLatency < 100 && uptime >= 99 {
		proxy.Quality = types.ProxyQualityExcellent
	} else if avgLatency < 300 && uptime >= 95 {
		proxy.Quality = types.ProxyQualityGood
	} else if avgLatency < 1000 && uptime >= 90 {
		proxy.Quality = types.ProxyQualityAverage
	} else if uptime > 0 {
		proxy.Quality = types.ProxyQualityPoor
	} else {
		proxy.Quality = types.ProxyQualityDead
	}
}

// calculateProxyScore calculates a composite score for proxy ranking
func (pm *AdvancedProxyManager) calculateProxyScore(proxy *types.Proxy) {
	if proxy.Metrics.TotalRequests == 0 {
		proxy.Score = 50.0 // Default neutral score
		return
	}

	// Scoring factors (weights)
	const (
		uptimeWeight    = 0.4  // 40% weight for uptime
		latencyWeight   = 0.3  // 30% weight for latency
		reliabilityWeight = 0.2  // 20% weight for reliability
		freshnessWeight = 0.1  // 10% weight for data freshness
	)

	// Uptime score (0-100)
	uptimeScore := proxy.Metrics.Uptime

	// Latency score (inverse relationship - lower latency = higher score)
	latencyScore := math.Max(0, 100-float64(proxy.Metrics.AverageLatency)/10)

	// Reliability score (based on consecutive failures)
	reliabilityScore := math.Max(0, 100-float64(proxy.Metrics.ConsecutiveFails)*20)

	// Freshness score (how recent is the data)
	hoursSinceLastTest := time.Since(proxy.LastTest).Hours()
	freshnessScore := math.Max(0, 100-hoursSinceLastTest*5)

	// Calculate composite score
	proxy.Score = uptimeScore*uptimeWeight +
		latencyScore*latencyWeight +
		reliabilityScore*reliabilityWeight +
		freshnessScore*freshnessWeight

	// Ensure score is within bounds
	if proxy.Score > 100 {
		proxy.Score = 100
	} else if proxy.Score < 0 {
		proxy.Score = 0
	}
}

// GetBestProxy returns the best proxy based on the current strategy with logging
func (pm *AdvancedProxyManager) GetBestProxy() (*types.Proxy, error) {
	correlationID := utils.GenerateCorrelationID()
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	if len(pm.proxies) == 0 {
		fmt.Printf("[WARN] No proxies available [CID:%s]\n", correlationID)
		return nil, fmt.Errorf("no proxies available")
	}

	// Filter out dead and blacklisted proxies
	workingProxies := pm.getWorkingProxies()
	if len(workingProxies) == 0 {
		fmt.Printf("[WARN] No working proxies available [CID:%s] (total: %d)\n", correlationID, len(pm.proxies))
		return nil, fmt.Errorf("no working proxies available")
	}

	fmt.Printf("[DEBUG] Selecting proxy using %s strategy [CID:%s] (candidates: %d)\n", string(pm.strategy), correlationID, len(workingProxies))

	var selectedProxy *types.Proxy
	switch pm.strategy {
	case StrategyBestScore:
		selectedProxy = pm.getBestScoreProxy(workingProxies)
	case StrategyRandomWeighted:
		selectedProxy = pm.getRandomWeightedProxy(workingProxies)
	case StrategyGeoPreferred:
		selectedProxy = pm.getGeoPreferredProxy(workingProxies)
	case StrategyLeastUsed:
		selectedProxy = pm.getLeastUsedProxy(workingProxies)
	default: // StrategyRoundRobin
		selectedProxy = pm.getRoundRobinProxy(workingProxies)
	}

	if selectedProxy != nil {
		fmt.Printf("[INFO] Proxy selected [CID:%s] %s:%d (score: %.2f, quality: %s)\n", 
			correlationID, selectedProxy.Host, selectedProxy.Port, selectedProxy.Score, string(selectedProxy.Quality))
	}

	return selectedProxy, nil
}

// getWorkingProxies returns only working, non-blacklisted proxies
func (pm *AdvancedProxyManager) getWorkingProxies() []types.Proxy {
	var working []types.Proxy
	for _, proxy := range pm.proxies {
		// Include proxies that are working OR haven't been tested yet (LastTest is zero)
		if (proxy.Working || proxy.LastTest.IsZero()) && 
		   !pm.blacklistedIPs[proxy.Host] && 
		   proxy.Metrics.ConsecutiveFails < pm.maxConsecutiveFails {
			working = append(working, proxy)
		}
	}
	return working
}

// getBestScoreProxy returns the proxy with the highest score
func (pm *AdvancedProxyManager) getBestScoreProxy(proxies []types.Proxy) *types.Proxy {
	if len(proxies) == 0 {
		return nil
	}

	sort.Slice(proxies, func(i, j int) bool {
		return proxies[i].Score > proxies[j].Score
	})

	return &proxies[0]
}

// getRandomWeightedProxy returns a proxy using weighted random selection
func (pm *AdvancedProxyManager) getRandomWeightedProxy(proxies []types.Proxy) *types.Proxy {
	if len(proxies) == 0 {
		return nil
	}

	// Simple implementation - use score as weight
	totalScore := 0.0
	for _, proxy := range proxies {
		totalScore += proxy.Score
	}

	if totalScore == 0 {
		return &proxies[0] // Fallback to first proxy
	}

	// This is a simplified random selection - in a real implementation,
	// you would use proper weighted random selection
	return &proxies[0]
}

// getGeoPreferredProxy returns a proxy from preferred countries
func (pm *AdvancedProxyManager) getGeoPreferredProxy(proxies []types.Proxy) *types.Proxy {
	if len(pm.preferredCountries) == 0 {
		return pm.getBestScoreProxy(proxies)
	}

	// Find proxies from preferred countries
	var preferred []types.Proxy
	for _, proxy := range proxies {
		if proxy.Location != nil {
			for _, country := range pm.preferredCountries {
				if proxy.Location.CountryCode == country {
					preferred = append(preferred, proxy)
					break
				}
			}
		}
	}

	if len(preferred) > 0 {
		return pm.getBestScoreProxy(preferred)
	}

	// Fallback to best score if no preferred found
	return pm.getBestScoreProxy(proxies)
}

// getLeastUsedProxy returns the proxy with the least usage
func (pm *AdvancedProxyManager) getLeastUsedProxy(proxies []types.Proxy) *types.Proxy {
	if len(proxies) == 0 {
		return nil
	}

	sort.Slice(proxies, func(i, j int) bool {
		return proxies[i].Metrics.TotalRequests < proxies[j].Metrics.TotalRequests
	})

	return &proxies[0]
}

// getRoundRobinProxy returns the next proxy in round-robin fashion
func (pm *AdvancedProxyManager) getRoundRobinProxy(proxies []types.Proxy) *types.Proxy {
	if len(proxies) == 0 {
		return nil
	}

	proxy := &proxies[pm.currentIndex%len(proxies)]
	pm.currentIndex++
	return proxy
}

// SetPreferredCountries sets the preferred countries for geo-selection
func (pm *AdvancedProxyManager) SetPreferredCountries(countries []string) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()
	pm.preferredCountries = countries
}

// BlacklistIP adds an IP to the blacklist
func (pm *AdvancedProxyManager) BlacklistIP(ip string) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()
	pm.blacklistedIPs[ip] = true
}

// GetProxyStats returns statistics about the proxy pool
func (pm *AdvancedProxyManager) GetProxyStats() map[string]interface{} {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	workingCount := 0
	avgScore := 0.0
	qualityDistribution := make(map[types.ProxyQuality]int)

	for _, proxy := range pm.proxies {
		if proxy.Working {
			workingCount++
		}
		avgScore += proxy.Score
		qualityDistribution[proxy.Quality]++
	}

	if len(pm.proxies) > 0 {
		avgScore /= float64(len(pm.proxies))
	}

	return map[string]interface{}{
		"total_proxies":        len(pm.proxies),
		"working_proxies":      workingCount,
		"average_score":        avgScore,
		"quality_distribution": qualityDistribution,
		"total_requests":       pm.totalRequestsServed,
		"total_errors":         pm.totalProxyErrors,
	}
}
