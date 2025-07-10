package checker

import (
	"testing"
	"time"
	"universal-checker/pkg/types"
)

func TestAdvancedProxyManager(t *testing.T) {
	manager := NewAdvancedProxyManager(StrategyBestScore)

	// Add dummy proxies
	proxy1 := types.Proxy{Host: "192.168.1.1", Port: 8080, Type: types.ProxyTypeHTTP}
	proxy2 := types.Proxy{Host: "192.168.1.2", Port: 8080, Type: types.ProxyTypeHTTPS}
	
	if err := manager.AddProxy(proxy1); err != nil {
		t.Errorf("Failed to add proxy1: %v", err)
	}

	if err := manager.AddProxy(proxy2); err != nil {
		t.Errorf("Failed to add proxy2: %v", err)
	}

	// Test best score selection
	proxy, err := manager.GetBestProxy()
	if err != nil || proxy == nil {
		t.Error("Expected to find the best proxy, but none was found")
	}

	// Test geo-preferred selection (no preferred country set yet)
	manager.SetPreferredCountries([]string{"US"})

	proxy, err = manager.GetBestProxy()
	if err != nil || proxy == nil {
		t.Error("Expected to find the best proxy (geo preferred), but none was found")
	}

	// Blacklist a proxy
	manager.BlacklistIP("192.168.1.1")
	
	// Test that blacklisted proxy is not returned
	proxy, err = manager.GetBestProxy()
	if err != nil {
		t.Error("Expected to find the best proxy, but got error instead")
	}

	if proxy != nil && proxy.Host == "192.168.1.1" {
		t.Error("Blacklisted proxy was incorrectly returned")
	}
}

func TestProxyHealthMonitor(t *testing.T) {
	manager := NewAdvancedProxyManager(StrategyBestScore)
	healthMonitor := NewProxyHealthMonitor(manager)

	proxy := types.Proxy{Host: "192.168.1.3", Port: 8080, Type: types.ProxyTypeHTTP}
	_ = manager.AddProxy(proxy)

	healthMonitor.SetCheckInterval(1 * time.Second)
	healthMonitor.SetMaxConcurrent(2)

	healthMonitor.Start()
	defer healthMonitor.Stop()

	// Wait for a couple of health checks
	time.Sleep(3 * time.Second)

	// Check statistics
	stats := healthMonitor.GetHealthCheckStats()
	if stats["total_checks"].(int64) == 0 {
		t.Error("Expected some health checks to have occurred by now")
	}

	// Check recent results
	results := healthMonitor.GetRecentResults(10)
	if len(results) == 0 {
		t.Error("Expected to have some recent health check results")
	}
}
