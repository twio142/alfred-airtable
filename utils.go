package main

import (
	// "encoding/json"
	// "io"
	// "io/fs"
	// "net/http"
	// "os"
	// "strings"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"time"
)

func notify(title, message string) {
	// Placeholder for notification
}

func match() {
}

func stringPtr(s string) *string {
	return &s
}

func getStringField(fields map[string]interface{}, key string) string {
	if value, ok := fields[key]; ok {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return ""
}

func getStringSliceField(fields map[string]interface{}, key string) []string {
	if value, ok := fields[key]; ok {
		if slice, ok := value.([]interface{}); ok {
			strSlice := make([]string, len(slice))
			for i, v := range slice {
				strSlice[i] = fmt.Sprintf("%v", v)
			}
			return strSlice
		}
	}
	return []string{}
}

func getTimeField(fields map[string]interface{}, key string) time.Time {
	if value, ok := fields[key]; ok {
		if str, ok := value.(string); ok {
			t, err := time.Parse(time.RFC3339, str)
			if err == nil {
				return t
			}
		}
	}
	return time.Time{}
}

func getBoolField(fields map[string]interface{}, key string) bool {
	if value, ok := fields[key]; ok {
		if b, ok := value.(bool); ok {
			return b
		}
	}
	return false
}

func randomString(length int) (string, error) {
	b := make([]byte, length)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func createCodeChallenge(codeVerifier string) string {
	hash := sha256.Sum256([]byte(codeVerifier))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}
