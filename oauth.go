package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

// Implement OAuth2 authentication and token management
// Start a HTTP server to handle the OAuth2 flow

const (
	clientID     = "your-client-id"
	clientSecret = "your-client-secret"
	redirectURI  = "http://localhost:8080/callback"
	authURL      = "https://airtable.com/oauth2/v1/authorize"
	tokenURL     = "https://airtable.com/oauth2/v1/token"
)

var (
	tokenFile = "token.json"
)

type Token struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	Expiry       int64  `json:"expiry"`
}

func handleOAuth() {
	http.HandleFunc("/callback", handleCallback)
	http.HandleFunc("/refresh", handleRefresh)
	go http.ListenAndServe(":8080", nil)

	url := fmt.Sprintf("%s?response_type=code&client_id=%s&redirect_uri=%s", authURL, clientID, redirectURI)
	fmt.Printf("Please visit the following URL to authenticate:\n%s\n", url)
}

func handleCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Code not found", http.StatusBadRequest)
		return
	}

	token, err := exchangeCodeForToken(code)
	if err != nil {
		http.Error(w, "Failed to exchange code for token", http.StatusInternalServerError)
		return
	}

	saveToken(token)
	fmt.Fprintf(w, "Authentication successful! You can close this window.")
}

func exchangeCodeForToken(code string) (*Token, error) {
	req, err := http.NewRequest("POST", tokenURL, nil)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(clientID, clientSecret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.URL.RawQuery = fmt.Sprintf("grant_type=authorization_code&code=%s&redirect_uri=%s", code, redirectURI)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.Reader(resp.Body))
	if err != nil {
		return nil, err
	}

	var token Token
	if err := json.Unmarshal(body, &token); err != nil {
		return nil, err
	}

	token.Expiry = time.Now().Add(time.Hour).Unix()
	return &token, nil
}

func saveToken(token *Token) {
	file, err := os.Create(tokenFile)
	if err != nil {
		log.Fatalf("Failed to create token file: %v", err)
	}
	defer file.Close()

	json.NewEncoder(file).Encode(token)
}

func handleRefresh(w http.ResponseWriter, r *http.Request) {
	token, err := loadToken()
	if err != nil {
		http.Error(w, "Failed to load token", http.StatusInternalServerError)
		return
	}

	if token.RefreshToken == "" {
		http.Error(w, "No refresh token available", http.StatusBadRequest)
		return
	}

	newToken, err := refreshAccessToken(token.RefreshToken)
	if err != nil {
		http.Error(w, "Failed to refresh access token", http.StatusInternalServerError)
		return
	}

	saveToken(newToken)
	fmt.Fprintf(w, "Token refreshed successfully!")
}

func loadToken() (*Token, error) {
	file, err := os.Open(tokenFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var token Token
	if err := json.NewDecoder(file).Decode(&token); err != nil {
		return nil, err
	}

	return &token, nil
}

func refreshAccessToken(refreshToken string) (*Token, error) {
	req, err := http.NewRequest("POST", tokenURL, nil)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(clientID, clientSecret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.URL.RawQuery = fmt.Sprintf("grant_type=refresh_token&refresh_token=%s", refreshToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.Reader(resp.Body))
	if err != nil {
		return nil, err
	}

	var token Token
	if err := json.Unmarshal(body, &token); err != nil {
		return nil, err
	}

	token.Expiry = time.Now().Add(time.Hour).Unix()
	return &token, nil
}
