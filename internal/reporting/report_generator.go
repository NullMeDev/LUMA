package reporting

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"universal-checker/pkg/types"
)

// Report represents a detailed operation report
type Report struct {
	GeneratedAt     time.Time          `json:"generated_at"`
	SessionID       string             `json:"session_id"`
	TotalCombos     int                `json:"total_combos"`
	ValidCombos     int                `json:"valid_combos"`
	InvalidCombos   int                `json:"invalid_combos"`
	ErrorCombos     int                `json:"error_combos"`
	ProxiesUsed     int                `json:"proxies_used"`
	TotalRequests   int                `json:"total_requests"`
	AverageLatency  float64            `json:"average_latency"`
	Results         []types.CheckResult `json:"results"`
	Statistics      map[string]interface{}  `json:"statistics"`
}

// GenerateReport generates a JSON report from a Checker's statistics and results
func GenerateReport(filename string, sessionID string, checkerStats types.CheckerStats, results []types.CheckResult) error {
	// Summarize results
	proxiesUsed := 0
	latencySum := 0
	
	for _, result := range results {
		if result.Proxy != nil {
			proxiesUsed++
		}
		latencySum += result.Latency
	}
	
	averageLatency := 0.0
	if len(results) > 0 {
		averageLatency = float64(latencySum) / float64(len(results))
	}

	// Construct report
	report := Report{
		GeneratedAt:     time.Now(),
		SessionID:       sessionID,
		TotalCombos:     checkerStats.TotalCombos,
		ValidCombos:     checkerStats.ValidCombos,
		InvalidCombos:   checkerStats.InvalidCombos,
		ErrorCombos:     checkerStats.ErrorCombos,
		ProxiesUsed:     proxiesUsed,
		TotalRequests:   checkerStats.ValidCombos + checkerStats.InvalidCombos + checkerStats.ErrorCombos,
		AverageLatency:  averageLatency,
		Results:         results,
		Statistics: map[string]interface{}{
			"current_cpm": checkerStats.CurrentCPM,
			"average_cpm": checkerStats.AverageCPM,
		},
	}

	// Prepare directory
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Write report
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}
