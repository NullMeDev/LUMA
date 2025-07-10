package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"universal-checker/pkg/types"
	"gopkg.in/yaml.v3"
)

// Parser handles parsing of different configuration formats
type Parser struct{}

// NewParser creates a new configuration parser
func NewParser() *Parser {
	return &Parser{}
}

// ParseConfig parses a configuration file based on its extension
func (p *Parser) ParseConfig(filePath string) (*types.Config, error) {
	ext := strings.ToLower(filepath.Ext(filePath))
	
	switch ext {
	case ".opk":
		return p.parseOPK(filePath)
	case ".svb":
		return p.parseSVB(filePath)
	case ".loli":
		return p.parseLoli(filePath)
	default:
		return nil, fmt.Errorf("unsupported config format: %s", ext)
	}
}

// parseOPK parses OpenBullet .opk configuration files
func (p *Parser) parseOPK(filePath string) (*types.Config, error) {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var opkConfig map[string]interface{}
	if err := json.Unmarshal(data, &opkConfig); err != nil {
		return nil, err
	}

	config := &types.Config{
		Name:            p.getStringValue(opkConfig, "name", filepath.Base(filePath)),
		Type:            types.ConfigTypeOPK,
		Method:          "GET", // Default
		Headers:         make(map[string]string),
		Data:            make(map[string]interface{}),
		Cookies:         make(map[string]string),
		Timeout:         p.getIntValue(opkConfig, "timeout", 30),
		FollowRedirects: p.getBoolValue(opkConfig, "followRedirects", true),
		CPM:             p.getIntValue(opkConfig, "cpm", 300),
		Delay:           p.getIntValue(opkConfig, "delay", 0),
		Retries:         p.getIntValue(opkConfig, "retries", 3),
		UseProxy:        p.getBoolValue(opkConfig, "useProxy", true),
		RawConfig:       opkConfig,
	}

	// Parse script blocks for OpenBullet configs
	if script, ok := opkConfig["script"].([]interface{}); ok {
		p.parseOBScript(script, config)
	} else {
		// Fallback to simple structure parsing
		config.Method = p.getStringValue(opkConfig, "method", "POST")
		if url, ok := opkConfig["url"].(string); ok {
			config.URL = url
		}
		
		// Extract headers
		if headers, ok := opkConfig["headers"].(map[string]interface{}); ok {
			for k, v := range headers {
				if str, ok := v.(string); ok {
					config.Headers[k] = str
				}
			}
		}
		
		// Extract data/payload
		if data, ok := opkConfig["data"].(map[string]interface{}); ok {
			config.Data = data
		}
		
		// Extract success/failure conditions
		if conditions, ok := opkConfig["conditions"].(map[string]interface{}); ok {
			if success, ok := conditions["success"].([]interface{}); ok {
				for _, s := range success {
					if str, ok := s.(string); ok {
						config.SuccessStrings = append(config.SuccessStrings, str)
					}
				}
			}
			if failure, ok := conditions["failure"].([]interface{}); ok {
				for _, f := range failure {
					if str, ok := f.(string); ok {
						config.FailureStrings = append(config.FailureStrings, str)
					}
				}
			}
		}
	}

	return config, nil
}

// parseOBScript parses OpenBullet script blocks
func (p *Parser) parseOBScript(script []interface{}, config *types.Config) {
	for _, block := range script {
		if blockMap, ok := block.(map[string]interface{}); ok {
			blockType := p.getStringValue(blockMap, "type", "")
			
			switch blockType {
			case "REQUEST":
				p.parseOBRequest(blockMap, config)
			case "KEYCHECK":
				p.parseOBKeyCheck(blockMap, config)
			case "PARSE":
				p.parseOBParse(blockMap, config)
			}
		}
	}
}

// parseOBRequest parses OpenBullet REQUEST blocks
func (p *Parser) parseOBRequest(block map[string]interface{}, config *types.Config) {
	if url, ok := block["url"].(string); ok {
		config.URL = url
	}
	if method, ok := block["method"].(string); ok {
		config.Method = method
	}
	
	// Parse headers
	if headers, ok := block["headers"].(map[string]interface{}); ok {
		for k, v := range headers {
			if str, ok := v.(string); ok {
				config.Headers[k] = str
			}
		}
	}
	
	// Parse post data
	if postData, ok := block["postData"].(string); ok {
		p.parseFormData(postData, config)
	} else if data, ok := block["data"].(map[string]interface{}); ok {
		config.Data = data
	}
}

// parseOBKeyCheck parses OpenBullet KEYCHECK blocks
func (p *Parser) parseOBKeyCheck(block map[string]interface{}, config *types.Config) {
	condition := p.getStringValue(block, "condition", "")
	keyCheckType := p.getStringValue(block, "keyCheckType", "")
	
	if keyCheckType == "SUCCESS" {
		config.SuccessStrings = append(config.SuccessStrings, condition)
	} else if keyCheckType == "FAILURE" {
		config.FailureStrings = append(config.FailureStrings, condition)
	}
}

// parseOBParse parses OpenBullet PARSE blocks (for variable extraction)
func (p *Parser) parseOBParse(block map[string]interface{}, config *types.Config) {
	// For now, we'll store parse blocks in raw config for potential future use
	if config.RawConfig["parseBlocks"] == nil {
		config.RawConfig["parseBlocks"] = make([]interface{}, 0)
	}
	config.RawConfig["parseBlocks"] = append(config.RawConfig["parseBlocks"].([]interface{}), block)
}

// parseSVB parses SilverBullet .svb configuration files
func (p *Parser) parseSVB(filePath string) (*types.Config, error) {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	// Try to detect if it's LoliScript format (common in SB configs)
	if p.isLoliScript(string(data)) {
		return p.parseLoliScript(filePath, data)
	}

	var svbConfig map[string]interface{}
	if err := yaml.Unmarshal(data, &svbConfig); err != nil {
		// Try JSON format as fallback
		if err := json.Unmarshal(data, &svbConfig); err != nil {
			return nil, err
		}
	}

	config := &types.Config{
		Name:            p.getStringValue(svbConfig, "name", filepath.Base(filePath)),
		Type:            types.ConfigTypeSVB,
		Method:          "GET", // Default
		Headers:         make(map[string]string),
		Data:            make(map[string]interface{}),
		Cookies:         make(map[string]string),
		Timeout:         p.getIntValue(svbConfig, "timeout", 30),
		FollowRedirects: p.getBoolValue(svbConfig, "followRedirects", true),
		CPM:             p.getIntValue(svbConfig, "cpm", 300),
		Delay:           p.getIntValue(svbConfig, "delay", 0),
		Retries:         p.getIntValue(svbConfig, "retries", 3),
		UseProxy:        p.getBoolValue(svbConfig, "useProxy", true),
		RawConfig:       svbConfig,
	}

	// Parse script if available (LoliScript in SB)
	if script, ok := svbConfig["script"].(string); ok {
		p.parseLoliScriptString(script, config)
	} else {
		// Standard structure parsing
		config.Method = p.getStringValue(svbConfig, "method", "POST")
		
		// Extract URL
		if url, ok := svbConfig["url"].(string); ok {
			config.URL = url
		}
		
		// Extract request configuration
		if request, ok := svbConfig["request"].(map[string]interface{}); ok {
			if headers, ok := request["headers"].(map[string]interface{}); ok {
				for k, v := range headers {
					if str, ok := v.(string); ok {
						config.Headers[k] = str
					}
				}
			}
			if data, ok := request["data"].(map[string]interface{}); ok {
				config.Data = data
			}
		}
		
		// Extract response conditions
		if response, ok := svbConfig["response"].(map[string]interface{}); ok {
			if success, ok := response["success"].([]interface{}); ok {
				for _, s := range success {
					if str, ok := s.(string); ok {
						config.SuccessStrings = append(config.SuccessStrings, str)
					}
				}
			}
			if failure, ok := response["failure"].([]interface{}); ok {
				for _, f := range failure {
					if str, ok := f.(string); ok {
						config.FailureStrings = append(config.FailureStrings, str)
					}
				}
			}
		}
	}

	return config, nil
}
// parseLoliScript parses .loli script string
func (p *Parser) parseLoliScript(filePath string, data []byte) (*types.Config, error) {
	lines := strings.Split(string(data), "\n")
	
	config := &types.Config{
		Name:            filepath.Base(filePath),
		Type:            types.ConfigTypeLoli,
		Method:          "GET", // Default
		Headers:         make(map[string]string),
		Data:            make(map[string]interface{}),
		Cookies:         make(map[string]string),
		Timeout:         30,
		FollowRedirects: true,
		CPM:             300,
		Delay:           0,
		Retries:         3,
		UseProxy:        true,
		RawConfig:       make(map[string]interface{}),
	}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse different directive types
		if strings.HasPrefix(line, "REQUEST") {
			config.URL = p.extractURL(line)
		} else if strings.HasPrefix(line, "HEADERS") {
			p.parseLoliHeaders(line, config)
		} else if strings.HasPrefix(line, "POSTDATA") {
			p.parseLoliPostData(line, config)
		} else if strings.HasPrefix(line, "KEYCHECK") {
			p.parseLoliKeyCheck(line, config)
		} else if strings.HasPrefix(line, "CPM") {
		if cpm := p.extractNumber(line); cpm > 0 {
				config.CPM = cpm
			}
		}
	}

	return config, nil
}

// parseLoli parses .loli configuration files
func (p *Parser) parseLoli(filePath string) (*types.Config, error) {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	return p.parseLoliScript(filePath, data)
}

// isLoliScript detects if content is LoliScript format
func (p *Parser) isLoliScript(content string) bool {
	return strings.Contains(content, "REQUEST") || strings.Contains(content, "KEYCHECK") || strings.Contains(content, "POSTDATA")
}

// parseLoliScriptString parses LoliScript from string
func (p *Parser) parseLoliScriptString(script string, config *types.Config) {
	lines := strings.Split(script, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		
		if strings.HasPrefix(line, "REQUEST") {
			config.URL = p.extractURL(line)
		} else if strings.HasPrefix(line, "HEADERS") {
			p.parseLoliHeaders(line, config)
		} else if strings.HasPrefix(line, "POSTDATA") {
			p.parseLoliPostData(line, config)
		} else if strings.HasPrefix(line, "KEYCHECK") {
			p.parseLoliKeyCheck(line, config)
		}
	}
}

// parseFormData parses form data string into config
func (p *Parser) parseFormData(formData string, config *types.Config) {
	pairs := strings.Split(formData, "&")
	for _, pair := range pairs {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			config.Data[key] = value
		}
	}
}

// Helper functions
func (p *Parser) getStringValue(data map[string]interface{}, key, defaultValue string) string {
	if val, ok := data[key].(string); ok {
		return val
	}
	return defaultValue
}

func (p *Parser) getIntValue(data map[string]interface{}, key string, defaultValue int) int {
	if val, ok := data[key].(float64); ok {
		return int(val)
	}
	if val, ok := data[key].(int); ok {
		return val
	}
	return defaultValue
}

func (p *Parser) getBoolValue(data map[string]interface{}, key string, defaultValue bool) bool {
	if val, ok := data[key].(bool); ok {
		return val
	}
	return defaultValue
}

func (p *Parser) extractURL(line string) string {
	re := regexp.MustCompile(`REQUEST\s+([^\s]+)\s+(.+)`)
	matches := re.FindStringSubmatch(line)
	if len(matches) >= 3 {
		return matches[2]
	}
	return ""
}

func (p *Parser) parseLoliHeaders(line string, config *types.Config) {
	re := regexp.MustCompile(`HEADERS\s+(.+)`)
	matches := re.FindStringSubmatch(line)
	if len(matches) >= 2 {
		headerPairs := strings.Split(matches[1], ";")
		for _, pair := range headerPairs {
			parts := strings.SplitN(pair, ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				config.Headers[key] = value
			}
		}
	}
}

func (p *Parser) parseLoliPostData(line string, config *types.Config) {
	re := regexp.MustCompile(`POSTDATA\s+(.+)`)
	matches := re.FindStringSubmatch(line)
	if len(matches) >= 2 {
		data := matches[1]
		// Parse form data
		pairs := strings.Split(data, "&")
		for _, pair := range pairs {
			parts := strings.SplitN(pair, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				config.Data[key] = value
			}
		}
	}
}

func (p *Parser) parseLoliKeyCheck(line string, config *types.Config) {
	re := regexp.MustCompile(`KEYCHECK\s+(.+)`)
	matches := re.FindStringSubmatch(line)
	if len(matches) >= 2 {
		keycheck := matches[1]
		// Parse keycheck conditions
		if strings.Contains(keycheck, "SUCCESS") {
			parts := strings.Split(keycheck, "SUCCESS")
			if len(parts) > 1 {
				condition := strings.TrimSpace(parts[0])
				// Remove "Contains" and quotes
				condition = strings.TrimPrefix(condition, "Contains")
				condition = strings.Trim(condition, ` "'`)
				config.SuccessStrings = append(config.SuccessStrings, condition)
			}
		}
		if strings.Contains(keycheck, "FAILURE") {
			parts := strings.Split(keycheck, "FAILURE")
			if len(parts) > 1 {
				condition := strings.TrimSpace(parts[0])
				// Remove "Contains" and quotes
				condition = strings.TrimPrefix(condition, "Contains")
				condition = strings.Trim(condition, ` "'`)
				config.FailureStrings = append(config.FailureStrings, condition)
			}
		}
	}
}


func (p *Parser) extractNumber(line string) int {
	re := regexp.MustCompile(`\d+`)
	match := re.FindString(line)
	if match != "" {
		if num, err := strconv.Atoi(match); err == nil {
			return num
		}
	}
	return 0
}

// DetectConfigType detects the configuration type based on file extension
func DetectConfigType(filePath string) types.ConfigType {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".opk":
		return types.ConfigTypeOPK
	case ".svb":
		return types.ConfigTypeSVB
	case ".loli":
		return types.ConfigTypeLoli
	default:
		return ""
	}
}
