package checker

import (
	"encoding/base64"
	"strings"
	"crypto/sha256"
	"crypto/md5"
	"crypto/hmac"
	"fmt"
	"hash"
	"math/rand"
	"net/url"
	"strconv"
	"time"
)

// FunctionType represents the type of function to apply
type FunctionType string

const (
	FuncBase64Encode   FunctionType = "Base64Encode"
	FuncBase64Decode   FunctionType = "Base64Decode"
	FuncSHA256         FunctionType = "SHA256"
	FuncMD5           FunctionType = "MD5"
	FuncHMAC          FunctionType = "HMAC"
	FuncToUpper       FunctionType = "ToUpper"
	FuncToLower       FunctionType = "ToLower"
	FuncURLEncode     FunctionType = "URLEncode"
	FuncURLDecode     FunctionType = "URLDecode"
	FuncLength        FunctionType = "Length"
	FuncReplace       FunctionType = "Replace"
	FuncRandomNum     FunctionType = "RandomNum"
	FuncRandomString  FunctionType = "RandomString"
	FuncUnixTime      FunctionType = "UnixTime"
	FuncTrim          FunctionType = "Trim"
)

// FunctionBlock applies transformations to inputs
type FunctionBlock struct{}

// Apply applies a function block to the input based on FunctionType
func (f *FunctionBlock) Apply(funcType FunctionType, input string, params ...string) (string, error) {
	switch funcType {
	case FuncBase64Encode:
		return base64.StdEncoding.EncodeToString([]byte(input)), nil
	
	case FuncBase64Decode:
		decoded, err := base64.StdEncoding.DecodeString(input)
		if err != nil {
			return "", fmt.Errorf("base64 decode error: %v", err)
		}
		return string(decoded), nil

	case FuncSHA256:
		h := sha256.New()
		h.Write([]byte(input))
		return fmt.Sprintf("%x", h.Sum(nil)), nil

	case FuncHMAC:
		if len(params) == 0 {
			return "", fmt.Errorf("HMAC requires a key parameter")
		}
		return computeHMAC(input, params[0], sha256.New), nil

	case FuncToUpper:
		return strings.ToUpper(input), nil

	case FuncToLower:
		return strings.ToLower(input), nil

	case FuncMD5:
		h := md5.New()
		h.Write([]byte(input))
		return fmt.Sprintf("%x", h.Sum(nil)), nil

	case FuncURLEncode:
		return url.QueryEscape(input), nil

	case FuncURLDecode:
		decoded, err := url.QueryUnescape(input)
		if err != nil {
			return "", fmt.Errorf("URL decode error: %v", err)
		}
		return decoded, nil

	case FuncLength:
		return strconv.Itoa(len(input)), nil

	case FuncReplace:
		if len(params) < 2 {
			return "", fmt.Errorf("Replace requires old and new string parameters")
		}
		return strings.ReplaceAll(input, params[0], params[1]), nil

	case FuncRandomNum:
		length := 10 // default
		if len(params) > 0 {
			if l, err := strconv.Atoi(params[0]); err == nil {
				length = l
			}
		}
		return generateRandomNumber(length), nil

	case FuncRandomString:
		length := 10 // default
		if len(params) > 0 {
			if l, err := strconv.Atoi(params[0]); err == nil {
				length = l
			}
		}
		return generateRandomString(length), nil

	case FuncUnixTime:
		return strconv.FormatInt(time.Now().Unix(), 10), nil

	case FuncTrim:
		return strings.TrimSpace(input), nil

	default:
		return "", fmt.Errorf("unsupported function type: %s", funcType)
	}
}

func computeHMAC(input, key string, newHash func() hash.Hash) string {
	h := hmac.New(newHash, []byte(key))
	h.Write([]byte(input))
	return fmt.Sprintf("%x", h.Sum(nil))
}

func generateRandomNumber(length int) string {
	rand.Seed(time.Now().UnixNano())
	result := ""
	for i := 0; i < length; i++ {
		result += strconv.Itoa(rand.Intn(10))
	}
	return result
}

func generateRandomString(length int) string {
	rand.Seed(time.Now().UnixNano())
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[rand.Intn(len(charset))]
	}
	return string(result)
}
