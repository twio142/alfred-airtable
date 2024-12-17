package main

import (
	// "encoding/json"
	// "io"
	// "io/fs"
	// "net/http"
	// "os"
	// "strings"
	// "time"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
)

func notify(title, message string) {
	// Placeholder for notification
}

func match() {

}

func stringPtr(s string) *string {
    return &s
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
