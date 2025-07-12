package utils

import (
	"fmt"
	"math/rand"
	"time"
)

// GenerateCorrelationID generates a unique correlation ID for request tracing
func GenerateCorrelationID() string {
	timestamp := time.Now().UnixNano()
	random := rand.Intn(9999)
	return fmt.Sprintf("CID-%d-%04d", timestamp, random)
}

// GenerateTaskID generates a unique task ID
func GenerateTaskID(prefix string) string {
	timestamp := time.Now().UnixNano()
	random := rand.Intn(9999)
	return fmt.Sprintf("%s-%d-%04d", prefix, timestamp, random)
}

// GenerateSessionID generates a unique session ID
func GenerateSessionID() string {
	timestamp := time.Now().Unix()
	random := rand.Intn(9999)
	return fmt.Sprintf("SID-%d-%04d", timestamp, random)
}

func init() {
	rand.Seed(time.Now().UnixNano())
}
