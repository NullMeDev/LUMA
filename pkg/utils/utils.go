package utils

import (
	"net"
	"os"
	"regexp"
	"strconv"
)

// FileExists checks if a file exists
func FileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}

// IsValidIP checks if a string is a valid IP address
func IsValidIP(ip string) bool {
	return net.ParseIP(ip) != nil
}

// IsNumeric checks if a string contains only numeric characters
func IsNumeric(s string) bool {
	_, err := strconv.Atoi(s)
	return err == nil
}

// IsValidEmail checks if a string is a valid email address
func IsValidEmail(email string) bool {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}

// SanitizeFilename removes invalid characters from a filename
func SanitizeFilename(filename string) string {
	reg := regexp.MustCompile(`[<>:"/\\|?*]`)
	return reg.ReplaceAllString(filename, "_")
}

// CreateDirectory creates a directory if it doesn't exist
func CreateDirectory(path string) error {
	return os.MkdirAll(path, 0755)
}
