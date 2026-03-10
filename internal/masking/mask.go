package masking

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/rand"
	"strings"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func MaskValue(value string, maskType string, params map[string]interface{}) string {
	switch maskType {
	case "full":
		return fullMask(value)
	case "partial":
		return partialMask(value, params)
	case "email":
		return emailMask(value)
	case "name":
		return nameMask(value)
	case "random_string":
		return randomString(params)
	case "random_number":
		return randomNumber(params)
	case "null":
		return ""
	case "hash":
		return hashMask(value)
	default:
		return value
	}
}

func fullMask(_ string) string {
	return "****"
}

func partialMask(val string, params map[string]interface{}) string {
	if len(val) <= 2 {
		return strings.Repeat("*", len(val))
	}
	return string(val[0]) + strings.Repeat("*", len(val)-2) + string(val[len(val)-1])
}

func emailMask(val string) string {
	parts := strings.Split(val, "@")
	if len(parts) != 2 {
		return val
	}
	name := parts[0]
	domain := parts[1]
	if len(name) <= 1 {
		return "*" + "@" + domain
	}
	return string(name[0]) + strings.Repeat("*", len(name)-1) + "@" + domain
}

func nameMask(val string) string {
	// Split name by spaces to handle first and last names
	words := strings.Fields(val)
	if len(words) == 0 {
		return val
	}

	maskedWords := make([]string, len(words))
	for i, word := range words {
		if len(word) == 0 {
			maskedWords[i] = word
		} else if len(word) == 1 {
			maskedWords[i] = word
		} else {
			// Keep first letter, mask the rest
			maskedWords[i] = string(word[0]) + strings.Repeat("*", len(word)-1)
		}
	}

	return strings.Join(maskedWords, " ")
}

func randomString(params map[string]interface{}) string {
	length := 8
	if v, ok := params["length"].(float64); ok {
		length = int(v)
	}
	letters := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	b := make([]rune, length)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func randomNumber(params map[string]interface{}) string {
	max := 1000
	if v, ok := params["max"].(float64); ok {
		max = int(v)
	}
	return fmt.Sprintf("%d", rand.Intn(max))
}

func hashMask(val string) string {
	h := sha256.Sum256([]byte(val))
	return hex.EncodeToString(h[:])
}
