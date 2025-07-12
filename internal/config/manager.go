package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"universal-checker/pkg/types"
)

// ConfigManager handles config loading, verification, and management
type ConfigManager struct {
	parser      *Parser
	loadedConfigs []types.Config
	proxyConfigs  []types.Config
	proxylessConfigs []types.Config
	stats       ConfigStats
}

// ConfigStats holds statistics about loaded configs
type ConfigStats struct {
	TotalConfigs    int
	ProxyConfigs    int
	ProxylessConfigs int
	SupportedFormats map[string]int
	LoadErrors      []string
	LastScan        time.Time
}

// NewConfigManager creates a new config manager
func NewConfigManager() *ConfigManager {
	return &ConfigManager{
		parser: NewParser(),
		stats: ConfigStats{
			SupportedFormats: make(map[string]int),
		},
	}
}

// LoadConfigsFromDrop handles drag-and-drop config loading
func (cm *ConfigManager) LoadConfigsFromDrop(filePaths []string) (*ConfigLoadResult, error) {
	result := &ConfigLoadResult{
		LoadedConfigs:    []types.Config{},
		ProxyConfigs:     []types.Config{},
		ProxylessConfigs: []types.Config{},
		Errors:          []string{},
		Stats:           ConfigLoadStats{},
	}

	for _, filePath := range filePaths {
		if err := cm.processDroppedPath(filePath, result); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Error processing %s: %v", filepath.Base(filePath), err))
		}
	}

	// Update manager state
	cm.loadedConfigs = result.LoadedConfigs
	cm.proxyConfigs = result.ProxyConfigs
	cm.proxylessConfigs = result.ProxylessConfigs
	cm.updateStats(result)

	return result, nil
}

// ConfigLoadResult contains the results of config loading
type ConfigLoadResult struct {
	LoadedConfigs    []types.Config    `json:"loaded_configs"`
	ProxyConfigs     []types.Config    `json:"proxy_configs"`
	ProxylessConfigs []types.Config    `json:"proxyless_configs"`
	Errors          []string          `json:"errors"`
	Stats           ConfigLoadStats   `json:"stats"`
}

// ConfigLoadStats contains statistics about the loading process
type ConfigLoadStats struct {
	TotalProcessed   int            `json:"total_processed"`
	SuccessfulLoads  int            `json:"successful_loads"`
	FailedLoads      int            `json:"failed_loads"`
	ProxyRequired    int            `json:"proxy_required"`
	ProxyOptional    int            `json:"proxy_optional"`
	FormatBreakdown  map[string]int `json:"format_breakdown"`
	ProcessingTime   time.Duration  `json:"processing_time"`
}

// processDroppedPath processes a single dropped file or directory
func (cm *ConfigManager) processDroppedPath(path string, result *ConfigLoadResult) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	if info.IsDir() {
		return cm.processDirectory(path, result)
	}

	return cm.processFile(path, result)
}

// processDirectory recursively processes a directory
func (cm *ConfigManager) processDirectory(dirPath string, result *ConfigLoadResult) error {
	return filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && cm.isConfigFile(path) {
			if err := cm.processFile(path, result); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("Error processing %s: %v", filepath.Base(path), err))
			}
		}
		return nil
	})
}

// processFile processes a single config file
func (cm *ConfigManager) processFile(filePath string, result *ConfigLoadResult) error {
	start := time.Now()
	result.Stats.TotalProcessed++

	// Detect and validate file format
	if !cm.isConfigFile(filePath) {
		return fmt.Errorf("unsupported file format")
	}

	// Parse the config
	config, err := cm.parser.ParseConfig(filePath)
	if err != nil {
		result.Stats.FailedLoads++
		return err
	}

	// Update format statistics
	format := strings.ToLower(filepath.Ext(filePath))
	if result.Stats.FormatBreakdown == nil {
		result.Stats.FormatBreakdown = make(map[string]int)
	}
	result.Stats.FormatBreakdown[format]++

	// Add to appropriate lists
	result.LoadedConfigs = append(result.LoadedConfigs, *config)
	result.Stats.SuccessfulLoads++

	if config.RequiresProxy {
		result.ProxyConfigs = append(result.ProxyConfigs, *config)
		result.Stats.ProxyRequired++
	} else {
		result.ProxylessConfigs = append(result.ProxylessConfigs, *config)
		result.Stats.ProxyOptional++
	}

	result.Stats.ProcessingTime += time.Since(start)
	return nil
}

// isConfigFile checks if a file is a supported config format
func (cm *ConfigManager) isConfigFile(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	supportedExts := []string{".opk", ".svb", ".loli", ".anom", ".json", ".yaml", ".yml"}
	
	for _, supportedExt := range supportedExts {
		if ext == supportedExt {
			return true
		}
	}
	return false
}

// VerifyConfig performs detailed verification of a single config
func (cm *ConfigManager) VerifyConfig(config *types.Config) *ConfigVerificationResult {
	result := &ConfigVerificationResult{
		ConfigName:     config.Name,
		IsValid:        true,
		RequiresProxy:  config.RequiresProxy,
		Issues:        []string{},
		Recommendations: []string{},
		Compatibility:  cm.checkCompatibility(config),
	}

	// Verify essential fields
	if config.URL == "" {
		result.Issues = append(result.Issues, "Missing target URL")
		result.IsValid = false
	}

	if len(config.SuccessStrings) == 0 && len(config.SuccessStatus) == 0 {
		result.Issues = append(result.Issues, "No success conditions defined")
		result.Recommendations = append(result.Recommendations, "Add success strings or status codes")
	}

	if config.CPM > 1000 {
		result.Recommendations = append(result.Recommendations, "High CPM detected - consider rate limiting")
	}

	// Check proxy requirements vs settings
	if config.RequiresProxy && !config.UseProxy {
		result.Issues = append(result.Issues, "Config requires proxy but UseProxy is disabled")
		result.IsValid = false
	}

	return result
}

// ConfigVerificationResult contains the results of config verification
type ConfigVerificationResult struct {
	ConfigName      string            `json:"config_name"`
	IsValid         bool              `json:"is_valid"`
	RequiresProxy   bool              `json:"requires_proxy"`
	Issues          []string          `json:"issues"`
	Recommendations []string          `json:"recommendations"`
	Compatibility   CompatibilityInfo `json:"compatibility"`
}

// CompatibilityInfo contains compatibility information
type CompatibilityInfo struct {
	OriginalFormat  string   `json:"original_format"`
	SupportedBy     []string `json:"supported_by"`
	FeatureSupport  map[string]bool `json:"feature_support"`
}

// checkCompatibility checks config compatibility with different tools
func (cm *ConfigManager) checkCompatibility(config *types.Config) CompatibilityInfo {
	compat := CompatibilityInfo{
		OriginalFormat: string(config.Type),
		SupportedBy:    []string{},
		FeatureSupport: make(map[string]bool),
	}

	// Universal checker always supports
	compat.SupportedBy = append(compat.SupportedBy, "Universal-Checker")

	// Check OpenBullet compatibility
	if config.Type == types.ConfigTypeOPK || cm.hasOpenBulletFeatures(config) {
		compat.SupportedBy = append(compat.SupportedBy, "OpenBullet")
	}

	// Check SilverBullet compatibility
	if config.Type == types.ConfigTypeSVB || cm.hasSilverBulletFeatures(config) {
		compat.SupportedBy = append(compat.SupportedBy, "SilverBullet")
	}

	// Feature support analysis
	compat.FeatureSupport["proxy_support"] = config.UseProxy
	compat.FeatureSupport["custom_headers"] = len(config.Headers) > 0
	compat.FeatureSupport["post_data"] = len(config.Data) > 0
	compat.FeatureSupport["success_detection"] = len(config.SuccessStrings) > 0
	compat.FeatureSupport["failure_detection"] = len(config.FailureStrings) > 0

	return compat
}

// hasOpenBulletFeatures checks if config has OpenBullet-specific features
func (cm *ConfigManager) hasOpenBulletFeatures(config *types.Config) bool {
	// Check for OpenBullet-specific patterns in raw config
	if parseBlocks, ok := config.RawConfig["parseBlocks"]; ok && parseBlocks != nil {
		return true
	}
	return false
}

// hasSilverBulletFeatures checks if config has SilverBullet-specific features
func (cm *ConfigManager) hasSilverBulletFeatures(config *types.Config) bool {
	// Check for SilverBullet-specific fields
	if needsProxies, ok := config.RawConfig["NeedsProxies"]; ok && needsProxies != nil {
		return true
	}
	if _, ok := config.RawConfig["OnlySocks"]; ok {
		return true
	}
	return false
}

// updateStats updates the manager's statistics
func (cm *ConfigManager) updateStats(result *ConfigLoadResult) {
	cm.stats.TotalConfigs = len(result.LoadedConfigs)
	cm.stats.ProxyConfigs = len(result.ProxyConfigs)
	cm.stats.ProxylessConfigs = len(result.ProxylessConfigs)
	cm.stats.LoadErrors = result.Errors
	cm.stats.LastScan = time.Now()

	// Update format breakdown
	for format, count := range result.Stats.FormatBreakdown {
		cm.stats.SupportedFormats[format] = count
	}
}

// GetStats returns current statistics
func (cm *ConfigManager) GetStats() ConfigStats {
	return cm.stats
}

// GetProxyConfigs returns configs that require proxies
func (cm *ConfigManager) GetProxyConfigs() []types.Config {
	return cm.proxyConfigs
}

// GetProxylessConfigs returns configs that don't require proxies
func (cm *ConfigManager) GetProxylessConfigs() []types.Config {
	return cm.proxylessConfigs
}

// GetAllConfigs returns all loaded configs
func (cm *ConfigManager) GetAllConfigs() []types.Config {
	return cm.loadedConfigs
}

// ValidateConfigBatch validates multiple configs and returns a summary
func (cm *ConfigManager) ValidateConfigBatch(configs []types.Config) *BatchValidationResult {
	result := &BatchValidationResult{
		TotalConfigs:    len(configs),
		ValidConfigs:    0,
		InvalidConfigs:  0,
		ProxyRequired:   0,
		ProxyOptional:   0,
		ValidationResults: make([]ConfigVerificationResult, 0),
	}

	for _, config := range configs {
		verification := cm.VerifyConfig(&config)
		result.ValidationResults = append(result.ValidationResults, *verification)

		if verification.IsValid {
			result.ValidConfigs++
		} else {
			result.InvalidConfigs++
		}

		if verification.RequiresProxy {
			result.ProxyRequired++
		} else {
			result.ProxyOptional++
		}
	}

	return result
}

// BatchValidationResult contains results of batch validation
type BatchValidationResult struct {
	TotalConfigs      int                         `json:"total_configs"`
	ValidConfigs      int                         `json:"valid_configs"`
	InvalidConfigs    int                         `json:"invalid_configs"`
	ProxyRequired     int                         `json:"proxy_required"`
	ProxyOptional     int                         `json:"proxy_optional"`
	ValidationResults []ConfigVerificationResult  `json:"validation_results"`
}
