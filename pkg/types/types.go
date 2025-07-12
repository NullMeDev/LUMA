package types

import (
	"net/http"
	"time"
)

// ConfigType represents the type of configuration file
type ConfigType string

const (
	ConfigTypeOPK  ConfigType = "opk"
	ConfigTypeSVB  ConfigType = "svb"
	ConfigTypeLoli ConfigType = "loli"
)

// ProxyType represents the type of proxy
type ProxyType string

const (
	ProxyTypeHTTP   ProxyType = "http"
	ProxyTypeHTTPS  ProxyType = "https"
	ProxyTypeSOCKS4 ProxyType = "socks4"
	ProxyTypeSOCKS5 ProxyType = "socks5"
)

// ProxyQuality represents the quality rating of a proxy
type ProxyQuality string

const (
	ProxyQualityExcellent ProxyQuality = "excellent" // < 100ms, 99%+ uptime
	ProxyQualityGood      ProxyQuality = "good"      // < 300ms, 95%+ uptime
	ProxyQualityAverage   ProxyQuality = "average"   // < 1000ms, 90%+ uptime
	ProxyQualityPoor      ProxyQuality = "poor"      // > 1000ms, < 90% uptime
	ProxyQualityDead      ProxyQuality = "dead"      // Not responding
)

// ProxyLocation represents geographical information about a proxy
type ProxyLocation struct {
	Country     string  `json:"country"`
	CountryCode string  `json:"country_code"`
	City        string  `json:"city"`
	Region      string  `json:"region"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
	Timezone    string  `json:"timezone"`
	ISP         string  `json:"isp"`
}

// ProxyMetrics represents performance and reliability metrics
type ProxyMetrics struct {
	TotalRequests    int     `json:"total_requests"`
	SuccessfulReqs   int     `json:"successful_requests"`
	FailedRequests   int     `json:"failed_requests"`
	AverageLatency   int     `json:"average_latency"`   // in milliseconds
	MinLatency       int     `json:"min_latency"`       // in milliseconds
	MaxLatency       int     `json:"max_latency"`       // in milliseconds
	Uptime           float64 `json:"uptime"`            // percentage
	LastSuccessTime  time.Time `json:"last_success_time"`
	ConsecutiveFails int     `json:"consecutive_fails"`
	BanDetected      bool    `json:"ban_detected"`
}

// Proxy represents a proxy server with advanced features
type Proxy struct {
	Host     string    `json:"host"`
	Port     int       `json:"port"`
	Type     ProxyType `json:"type"`
	Username string    `json:"username,omitempty"`
	Password string    `json:"password,omitempty"`
	
	// Status and performance
	Working     bool         `json:"working"`
	Latency     int          `json:"latency"` // current latency in milliseconds
	LastTest    time.Time    `json:"last_test"`
	Quality     ProxyQuality `json:"quality"`
	Score       float64      `json:"score"` // 0-100 composite score
	
	// Geographic and network info
	Location *ProxyLocation `json:"location,omitempty"`
	Metrics  *ProxyMetrics  `json:"metrics,omitempty"`
	
	// Advanced features
	SupportsHTTPS bool      `json:"supports_https"`
	SupportsUDP   bool      `json:"supports_udp"`
	Anonymity     string    `json:"anonymity"` // "transparent", "anonymous", "elite"
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// Combo represents a username:password combination
type Combo struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email,omitempty"`
	Line     string `json:"line"` // original line format
}

// Config represents a checker configuration
type Config struct {
	Name        string                 `json:"name"`
	Type        ConfigType             `json:"type"`
	URL         string                 `json:"url"`
	Method      string                 `json:"method"`
	Headers     map[string]string      `json:"headers"`
	Data        map[string]interface{} `json:"data"`
	Cookies     map[string]string      `json:"cookies"`
	Timeout     int                    `json:"timeout"`
	FollowRedirects bool              `json:"follow_redirects"`

	// Success/Failure detection
	SuccessStrings []string `json:"success_strings"`
	FailureStrings []string `json:"failure_strings"`
	SuccessStatus  []int    `json:"success_status"`
	FailureStatus  []int    `json:"failure_status"`

	// Rate limiting
	CPM         int `json:"cpm"`          // checks per minute
	Delay       int `json:"delay"`        // delay between requests in ms
	Retries     int `json:"retries"`      // number of retries on failure

	// Proxy settings
	UseProxy      bool      `json:"use_proxy"`
	ProxyType     ProxyType `json:"proxy_type"`
	RequiresProxy bool      `json:"requires_proxy"` // Intelligent proxy detection result

	// Raw config data for different formats
	RawConfig map[string]interface{} `json:"raw_config"`
}

// BotStatus represents the status of a bot/worker
type BotStatus string

const (
	BotStatusNone    BotStatus = "NONE"
	BotStatusError   BotStatus = "ERROR"
	BotStatusSuccess BotStatus = "SUCCESS"
	BotStatusFail    BotStatus = "FAIL"
	BotStatusBan     BotStatus = "BAN"
	BotStatusRetry   BotStatus = "RETRY"
	BotStatusCustom  BotStatus = "CUSTOM"
)

// CheckResult represents the result of a combo check
type CheckResult struct {
	Combo     Combo     `json:"combo"`
	Config    string    `json:"config"`
	Status    BotStatus `json:"status"` // Enhanced bot status
	Response  string    `json:"response"`
	Proxy     *Proxy    `json:"proxy,omitempty"`
	Timestamp time.Time `json:"timestamp"`
	CPM       float64   `json:"cpm"`
	Latency   int       `json:"latency"`
	Error     string    `json:"error,omitempty"`
	
	// Enhanced data from parsing
	CapturedData map[string]interface{} `json:"captured_data,omitempty"`
	Variables    map[string]interface{} `json:"variables,omitempty"`
}

// CheckerStats represents statistics for the checker
type CheckerStats struct {
	TotalCombos    int       `json:"total_combos"`
	ValidCombos    int       `json:"valid_combos"`
	InvalidCombos  int       `json:"invalid_combos"`
	ErrorCombos    int       `json:"error_combos"`
	CurrentCPM     float64   `json:"current_cpm"`
	AverageCPM     float64   `json:"average_cpm"`
	StartTime      time.Time `json:"start_time"`
	ElapsedTime    int       `json:"elapsed_time"` // in seconds
	WorkingProxies int       `json:"working_proxies"`
	TotalProxies   int       `json:"total_proxies"`
	ActiveWorkers  int       `json:"active_workers"`
}

// CheckerConfig represents the global checker configuration
type CheckerConfig struct {
	MaxWorkers     int  `json:"max_workers"`
	ProxyTimeout   int  `json:"proxy_timeout"`
	RequestTimeout int  `json:"request_timeout"`
	RetryCount     int  `json:"retry_count"`
	ProxyRotation  bool `json:"proxy_rotation"`
	
	// Auto proxy scraping
	AutoScrapeProxies bool     `json:"auto_scrape_proxies"`
	ProxySources      []string `json:"proxy_sources"`
	
	// Output settings
	SaveValidOnly   bool   `json:"save_valid_only"`
	OutputFormat    string `json:"output_format"` // "txt", "json", "csv"
	OutputDirectory string `json:"output_directory"`
}

// HTTPClient represents an HTTP client with proxy support
type HTTPClient struct {
	Client *http.Client
	Proxy  *Proxy
}

// WorkerTask represents a task for a worker
type WorkerTask struct {
	Combo  Combo
	Config Config
	Proxy  *Proxy
}

// WorkerResult represents the result from a worker
type WorkerResult struct {
	Result CheckResult
	Error  error
}

// Global checker types for enhanced functionality

// GlobalWorkerTask represents a global task for testing a combo against all configs
type GlobalWorkerTask struct {
	TaskID  int
	Combo   Combo
	Configs []Config // ALL configs to test against
	Proxy   *Proxy
}

// GlobalWorkerResult represents the result from a global worker
type GlobalWorkerResult struct {
	TaskID           int
	Combo            Combo
	Results          []CheckResult // Results for each config
	OverallStatus    string        // "valid", "invalid", "error"
	ValidConfigCount int           // Number of configs that returned valid
	Timestamp        time.Time
	Latency          int // Total latency for all config tests
	WorkerID         int
	Proxy            *Proxy
}

// GlobalCheckerStats represents enhanced statistics for the global checker
type GlobalCheckerStats struct {
	TotalCombos    int       `json:"total_combos"`
	TotalConfigs   int       `json:"total_configs"`
	TotalTasks     int       `json:"total_tasks"`
	ProcessedTasks int       `json:"processed_tasks"`
	ValidCombos    int       `json:"valid_combos"`
	InvalidCombos  int       `json:"invalid_combos"`
	ErrorCombos    int       `json:"error_combos"`
	CurrentCPM     float64   `json:"current_cpm"`
	AverageCPM     float64   `json:"average_cpm"`
	StartTime      time.Time `json:"start_time"`
	ElapsedTime    int       `json:"elapsed_time"` // in seconds
	WorkingProxies int       `json:"working_proxies"`
	TotalProxies   int       `json:"total_proxies"`
	ActiveWorkers  int       `json:"active_workers"`
}

// LogEntry represents a log entry for GUI display
type LogEntry struct {
	Level     string    `json:"level"`     // "debug", "info", "warning", "error", "success"
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}
