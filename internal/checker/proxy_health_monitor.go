package checker

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"universal-checker/pkg/types"
	"universal-checker/pkg/utils"
)

// HealthCheckResult represents the result of a health check
type HealthCheckResult struct {
	ProxyID     string                 `json:"proxy_id"`
	Success     bool                   `json:"success"`
	Latency     int                    `json:"latency"`
	Error       string                 `json:"error,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
	AdditionalInfo map[string]interface{} `json:"additional_info,omitempty"`
}

// ProxyHealthMonitor continuously monitors proxy health
type ProxyHealthMonitor struct {
	proxyManager    *AdvancedProxyManager
	checkInterval   time.Duration
	timeout         time.Duration
	maxConcurrent   int
	ctx             context.Context
	cancel          context.CancelFunc
	wg              sync.WaitGroup
	
	// Health check results storage
	recentResults   []HealthCheckResult
	resultsMutex    sync.RWMutex
	maxResultsStore int
	
	// Statistics
	totalChecks     int64
	successfulChecks int64
	failedChecks    int64
	statsMutex      sync.RWMutex
	
	// Error recovery and circuit breaker
	errorHistory    []error
	errorHistoryMutex sync.RWMutex
	maxErrorHistory int
	circuitBreakerThreshold int
	circuitBreakerWindow time.Duration
}

// NewProxyHealthMonitor creates a new health monitor
func NewProxyHealthMonitor(proxyManager *AdvancedProxyManager) *ProxyHealthMonitor {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &ProxyHealthMonitor{
		proxyManager:    proxyManager,
		checkInterval:   30 * time.Second, // Check every 30 seconds
		timeout:         10 * time.Second,
		maxConcurrent:   10,
		ctx:             ctx,
		cancel:          cancel,
		recentResults:   make([]HealthCheckResult, 0),
		maxResultsStore: 1000, // Store last 1000 results
		
		// Error recovery and circuit breaker configuration
		errorHistory:    make([]error, 0),
		maxErrorHistory: 100, // Store last 100 errors
		circuitBreakerThreshold: 10, // Trip circuit breaker after 10 consecutive errors
		circuitBreakerWindow: 5 * time.Minute, // Circuit breaker window
	}
}

// Start begins the health monitoring process
func (hm *ProxyHealthMonitor) Start() {
	log.Println("[INFO] Starting proxy health monitor...")
	
	hm.wg.Add(1)
	go hm.healthCheckLoop()
}

// Stop stops the health monitoring process
func (hm *ProxyHealthMonitor) Stop() {
	log.Println("[INFO] Stopping proxy health monitor...")
	hm.cancel()
	hm.wg.Wait()
	log.Println("[INFO] Proxy health monitor stopped")
}

// healthCheckLoop is the main loop for health checking
func (hm *ProxyHealthMonitor) healthCheckLoop() {
	defer hm.wg.Done()
	
	ticker := time.NewTicker(hm.checkInterval)
	defer ticker.Stop()
	
	// Run initial health check
	hm.performHealthChecks()
	
	for {
		select {
		case <-hm.ctx.Done():
			return
		case <-ticker.C:
			hm.performHealthChecks()
		}
	}
}

// performHealthChecks runs health checks on all proxies
func (hm *ProxyHealthMonitor) performHealthChecks() {
	hm.proxyManager.mutex.RLock()
	proxies := make([]types.Proxy, len(hm.proxyManager.proxies))
	copy(proxies, hm.proxyManager.proxies)
	hm.proxyManager.mutex.RUnlock()
	
	if len(proxies) == 0 {
		return
	}
	
	log.Printf("[INFO] Starting health check for %d proxies", len(proxies))
	
	// Create a semaphore to limit concurrent checks
	semaphore := make(chan struct{}, hm.maxConcurrent)
	var checkWg sync.WaitGroup
	
	for i := range proxies {
		checkWg.Add(1)
		go func(proxy *types.Proxy) {
			defer checkWg.Done()
			
			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			
			hm.checkSingleProxy(proxy)
		}(&proxies[i])
	}
	
	checkWg.Wait()
	log.Printf("[INFO] Health check completed")
}

// checkSingleProxy performs a health check on a single proxy with comprehensive logging
func (hm *ProxyHealthMonitor) checkSingleProxy(proxy *types.Proxy) {
	correlationID := utils.GenerateCorrelationID()
	ctx, cancel := context.WithTimeout(hm.ctx, 30*time.Second)
	defer cancel()

	start := time.Now()
	proxyID := fmt.Sprintf("%s:%d", proxy.Host, proxy.Port)

	// Log health check start
	log.Printf("[DEBUG] Starting health check for proxy %s [CID:%s]", proxyID, correlationID)

	ch := make(chan error, 1)
	go func() {
		ch <- hm.proxyManager.TestProxyWithContext(ctx, proxy)
	}()

	select {
	case err := <-ch:
		// Proxy check completed within timeout
		latency := time.Since(start)
		latencyMs := int(latency.Milliseconds())

		// Create health check result
		result := HealthCheckResult{
			ProxyID:   proxyID,
			Success:   err == nil,
			Latency:   latencyMs,
			Timestamp: time.Now(),
			AdditionalInfo: map[string]interface{}{
				"proxy_type":     string(proxy.Type),
				"score":          proxy.Score,
				"quality":        string(proxy.Quality),
				"correlation_id": correlationID,
				"timeout_used":   "30s",
			},
		}

		if err != nil {
			result.Error = err.Error()
			hm.storeError(err) // Store error for analysis
			log.Printf("[WARN] Proxy %s health check failed [CID:%s] [%dms]: %s", proxyID, correlationID, latencyMs, err.Error())
		} else {
			log.Printf("[INFO] Proxy %s health check successful [CID:%s] [%dms]", proxyID, correlationID, latencyMs)
		}

		// Add location info if available
		if proxy.Location != nil {
			result.AdditionalInfo["country"] = proxy.Location.Country
			result.AdditionalInfo["city"] = proxy.Location.City
		}

		// Store the result
		hm.storeHealthCheckResult(result)

		// Update statistics
		hm.updateHealthCheckStats(result.Success)

		// Log performance warnings
		if result.Success && latencyMs > 5000 {
			log.Printf("[WARN] Proxy %s is slow [CID:%s]: %dms (threshold: 5000ms)", proxyID, correlationID, latencyMs)
		}

		// Auto-blacklist consistently failing proxies
		if proxy.Metrics != nil && proxy.Metrics.ConsecutiveFails >= 5 {
			log.Printf("[WARN] Auto-blacklisting proxy %s [CID:%s] (5+ consecutive failures)", proxyID, correlationID)
			hm.proxyManager.BlacklistIP(proxy.Host)
		}

	case <-ctx.Done():
		// Timeout exceeded
		log.Printf("[ERROR] Proxy %s health check timed out [CID:%s] (30s timeout)", proxyID, correlationID)
		hm.storeHealthCheckResult(HealthCheckResult{
			ProxyID:   proxyID,
			Success:   false,
			Error:     "health check timeout (30s)",
			Timestamp: time.Now(),
			AdditionalInfo: map[string]interface{}{
				"correlation_id": correlationID,
				"timeout_used":   "30s",
				"timeout_reason": "hard_timeout",
			},
		})
		hm.updateHealthCheckStats(false)
		hm.proxyManager.BlacklistIP(proxy.Host)
	}
}

// storeHealthCheckResult stores a health check result
func (hm *ProxyHealthMonitor) storeHealthCheckResult(result HealthCheckResult) {
	hm.resultsMutex.Lock()
	defer hm.resultsMutex.Unlock()
	
	hm.recentResults = append(hm.recentResults, result)
	
	// Keep only the most recent results
	if len(hm.recentResults) > hm.maxResultsStore {
		hm.recentResults = hm.recentResults[len(hm.recentResults)-hm.maxResultsStore:]
	}
}

// updateHealthCheckStats updates the health check statistics
func (hm *ProxyHealthMonitor) updateHealthCheckStats(success bool) {
	hm.statsMutex.Lock()
	defer hm.statsMutex.Unlock()
	
	hm.totalChecks++
	if success {
		hm.successfulChecks++
	} else {
		hm.failedChecks++
	}
}

// GetRecentResults returns recent health check results
func (hm *ProxyHealthMonitor) GetRecentResults(limit int) []HealthCheckResult {
	hm.resultsMutex.RLock()
	defer hm.resultsMutex.RUnlock()
	
	if limit <= 0 || limit > len(hm.recentResults) {
		limit = len(hm.recentResults)
	}
	
	// Return the most recent results
	start := len(hm.recentResults) - limit
	if start < 0 {
		start = 0
	}
	
	results := make([]HealthCheckResult, limit)
	copy(results, hm.recentResults[start:])
	
	return results
}

// GetHealthCheckStats returns health check statistics
func (hm *ProxyHealthMonitor) GetHealthCheckStats() map[string]interface{} {
	hm.statsMutex.RLock()
	defer hm.statsMutex.RUnlock()
	
	successRate := 0.0
	if hm.totalChecks > 0 {
		successRate = float64(hm.successfulChecks) / float64(hm.totalChecks) * 100
	}
	
	return map[string]interface{}{
		"total_checks":     hm.totalChecks,
		"successful_checks": hm.successfulChecks,
		"failed_checks":    hm.failedChecks,
		"success_rate":     successRate,
		"check_interval":   hm.checkInterval.String(),
		"max_concurrent":   hm.maxConcurrent,
	}
}

// GetProxyHealthSummary returns a summary of proxy health by region/quality
func (hm *ProxyHealthMonitor) GetProxyHealthSummary() map[string]interface{} {
	hm.proxyManager.mutex.RLock()
	defer hm.proxyManager.mutex.RUnlock()
	
	summary := map[string]interface{}{
		"by_quality":  make(map[types.ProxyQuality]int),
		"by_country":  make(map[string]int),
		"by_type":     make(map[types.ProxyType]int),
		"total_working": 0,
		"total_proxies": len(hm.proxyManager.proxies),
	}
	
	qualityCount := make(map[types.ProxyQuality]int)
	countryCount := make(map[string]int)
	typeCount := make(map[types.ProxyType]int)
	workingCount := 0
	
	for _, proxy := range hm.proxyManager.proxies {
		qualityCount[proxy.Quality]++
		typeCount[proxy.Type]++
		
		if proxy.Working {
			workingCount++
		}
		
		if proxy.Location != nil && proxy.Location.Country != "" {
			countryCount[proxy.Location.Country]++
		}
	}
	
	summary["by_quality"] = qualityCount
	summary["by_country"] = countryCount
	summary["by_type"] = typeCount
	summary["total_working"] = workingCount
	
	return summary
}

// SetCheckInterval changes the health check interval
func (hm *ProxyHealthMonitor) SetCheckInterval(interval time.Duration) {
	hm.checkInterval = interval
	log.Printf("[INFO] Health check interval changed to %s", interval.String())
}

// SetMaxConcurrent changes the maximum concurrent health checks
func (hm *ProxyHealthMonitor) SetMaxConcurrent(max int) {
	if max > 0 {
		hm.maxConcurrent = max
		log.Printf("[INFO] Max concurrent health checks changed to %d", max)
	}
}

// TriggerImmediateCheck triggers an immediate health check for all proxies
func (hm *ProxyHealthMonitor) TriggerImmediateCheck() {
	log.Println("[INFO] Triggering immediate health check...")
	go hm.performHealthChecks()
}

// GetFailingProxies returns proxies that are currently failing
func (hm *ProxyHealthMonitor) GetFailingProxies() []types.Proxy {
	hm.proxyManager.mutex.RLock()
	defer hm.proxyManager.mutex.RUnlock()
	
	var failing []types.Proxy
	for _, proxy := range hm.proxyManager.proxies {
		if !proxy.Working || proxy.Metrics.ConsecutiveFails > 0 {
			failing = append(failing, proxy)
		}
	}
	
	return failing
}

// storeError stores an error in the error history for analysis
func (hm *ProxyHealthMonitor) storeError(err error) {
	if err == nil {
		return
	}
	
	hm.errorHistoryMutex.Lock()
	defer hm.errorHistoryMutex.Unlock()
	
	hm.errorHistory = append(hm.errorHistory, err)
	
	// Keep only the most recent errors
	if len(hm.errorHistory) > hm.maxErrorHistory {
		hm.errorHistory = hm.errorHistory[len(hm.errorHistory)-hm.maxErrorHistory:]
	}
}

// GetErrorHistory returns recent errors for analysis
func (hm *ProxyHealthMonitor) GetErrorHistory() []error {
	hm.errorHistoryMutex.RLock()
	defer hm.errorHistoryMutex.RUnlock()
	
	errorsCopy := make([]error, len(hm.errorHistory))
	copy(errorsCopy, hm.errorHistory)
	
	return errorsCopy
}

// isCircuitBreakerTripped checks if the circuit breaker should be tripped
func (hm *ProxyHealthMonitor) isCircuitBreakerTripped() bool {
	hm.errorHistoryMutex.RLock()
	defer hm.errorHistoryMutex.RUnlock()
	
	if len(hm.errorHistory) < hm.circuitBreakerThreshold {
		return false
	}
	
	// Check if the last N errors occurred within the circuit breaker window
	recentErrors := hm.errorHistory[len(hm.errorHistory)-hm.circuitBreakerThreshold:]
	now := time.Now()
	
	for _, err := range recentErrors {
		// For this simple implementation, we assume all errors are recent
		// In a real implementation, you'd store timestamps with errors
		_ = err
		if now.Sub(now) < hm.circuitBreakerWindow {
			return true
		}
	}
	
	return false
}

// RecoverFromErrors attempts to recover from critical errors
func (hm *ProxyHealthMonitor) RecoverFromErrors() {
	log.Println("[INFO] Attempting to recover from health check errors...")
	
	// Clear error history
	hm.errorHistoryMutex.Lock()
	hm.errorHistory = make([]error, 0)
	hm.errorHistoryMutex.Unlock()
	
	// Reset consecutive failures for all proxies
	hm.proxyManager.mutex.Lock()
	for i := range hm.proxyManager.proxies {
		if hm.proxyManager.proxies[i].Metrics != nil {
			hm.proxyManager.proxies[i].Metrics.ConsecutiveFails = 0
		}
	}
	hm.proxyManager.mutex.Unlock()
	
	log.Println("[INFO] Health check error recovery completed")
}
