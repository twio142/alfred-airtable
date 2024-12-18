package main

import (
	// "bytes"
	"encoding/json"
	"log"

	// "net/http"
	// "net/http/httptest"
	"context"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"golang.org/x/oauth2"
)

func TestAuth_isValid(t *testing.T) {
	airtable := &Airtable{
		BaseURL: "https://api.airtable.com/v0",
		BaseID:  os.Getenv("BASE_ID"),
		DBPath:  "cache_test.db",
	}
	airtable.Cache = &Cache{File: airtable.DBPath}
	err := airtable.Cache.init()
	if err != nil {
		t.Errorf("init() error = %v", err)
	}

	auth := Auth{
		Token: &oauth2.Token{},
	}
	auth.read(airtable.Cache)
	airtable.Auth = &auth

	if !airtable.Auth.Valid() {
		t.Errorf("Expected token to be valid")
	}

	airtable.Auth.Expiry = time.Now().Add(-time.Hour)
	if auth.Valid() {
		t.Errorf("Expected token to be invalid")
	}
}

func TestAuth_isRefreshValid(t *testing.T) {
	airtable := &Airtable{
		BaseURL: "https://api.airtable.com/v0",
		BaseID:  os.Getenv("BASE_ID"),
		DBPath:  "cache_test.db",
	}
	airtable.Cache = &Cache{File: airtable.DBPath}
	err := airtable.Cache.init()
	if err != nil {
		t.Errorf("init() error = %v", err)
	}

	auth := Auth{
		Token: &oauth2.Token{},
	}
	auth.read(airtable.Cache)
	airtable.Auth = &auth

	if !auth.refreshValid() {
		t.Errorf("Expected refresh token to be valid")
	}

	auth.RefreshExpiry = &[]time.Time{time.Now().Add(-time.Hour)}[0]
	if auth.refreshValid() {
		t.Errorf("Expected refresh token to be invalid")
	}
}

func TestAuth_read(t *testing.T) {
	cache := &Cache{File: ":memory:"}
	cache.init()
	cache.setData("AccessToken", "test_token")
	cache.setData("Expiry", strconv.FormatInt(time.Now().Add(time.Hour).Unix(), 10))
	cache.setData("RefreshToken", "test_refresh_token")
	cache.setData("RefreshExpiry", strconv.FormatInt(time.Now().Add(time.Hour).Unix(), 10))

	auth := &Auth{}
	auth.read(cache)

	if auth.AccessToken != "test_token" {
		t.Errorf("Expected AccessToken to be 'test_token', got '%s'", auth.AccessToken)
	}
	if auth.RefreshToken != "test_refresh_token" {
		t.Errorf("Expected RefreshToken to be 'test_refresh_token', got '%s'", auth.RefreshToken)
	}
}

func TestAuth_write(t *testing.T) {
	cache := &Cache{File: ":memory:"}
	cache.init()

	auth := &Auth{
		Token: &oauth2.Token{
			AccessToken:  "test_token",
			Expiry:       time.Now().Add(time.Hour),
			RefreshToken: "test_refresh_token",
		},
		RefreshExpiry: &[]time.Time{time.Now().Add(time.Hour)}[0],
	}
	auth.write(cache)

	accessToken, _ := cache.getData("AccessToken")
	if *accessToken != "test_token" {
		t.Errorf("Expected AccessToken to be 'test_token', got '%s'", *accessToken)
	}
	refreshToken, _ := cache.getData("RefreshToken")
	if *refreshToken != "test_refresh_token" {
		t.Errorf("Expected RefreshToken to be 'test_refresh_token', got '%s'", *refreshToken)
	}
}

func TestAuth(t *testing.T) {
	a := &Airtable{
		Auth:  &Auth{},
		Cache: &Cache{File: ":memory:"},
	}
	a.Cache.init()

	o := &OAuth{}
	o.init()

	redirectURL := o.config.RedirectURL
	u, err := url.Parse(redirectURL)
	if err != nil {
		t.Fatal(err)
	}
	baseURL := u.Scheme + "://" + u.Host

	auth := Auth{}
	// read tokens from .creds.json
	file, err := os.Open(".creds.json")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&auth); err != nil {
		t.Fatal(err)
	}

	// auth.read(a.Cache)

	server := o.startServer()
	cmd := exec.Command("open", baseURL)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}

	newAuth := <-o.authComplete
	newAuth.write(a.Cache)
	a.Auth = &newAuth

	// Shutdown the server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server Shutdown Failed:%+v", err)
	}
	log.Println("Server exited properly")
}

func TestRefresh(t *testing.T) {
	a := &Airtable{
		Auth:  &Auth{},
		Cache: &Cache{File: ":memory:"},
	}
	a.Cache.init()

	o := &OAuth{}
	o.init()

	redirectURL := o.config.RedirectURL
	u, err := url.Parse(redirectURL)
	if err != nil {
		t.Fatal(err)
	}
	baseURL := u.Scheme + "://" + u.Host

	auth := Auth{}
	// read tokens from .creds.json
	file, err := os.Open(".creds.json")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&auth); err != nil {
		t.Fatal(err)
	}
	// auth.read(a.Cache)

	if auth.refreshValid() {
		server := o.startServer()
		defer server.Shutdown(context.Background())

		if err := exec.Command("curl", baseURL+"/refresh?refresh_token="+auth.Token.RefreshToken).Run(); err != nil {
			t.Fatal(err)
		}

		select {
		case newAuth := <-o.authComplete:
			a.Auth = &newAuth
			newAuth.write(a.Cache)
		case <-time.After(10 * time.Second):
			t.Fatal("Timeout waiting for new authentication")
		}
		return
	}

	go o.startServer()
	cmd := exec.Command("open", baseURL)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}

	newAuth := <-o.authComplete
	newAuth.write(a.Cache)
	a.Auth = &newAuth
}

func TestMain(m *testing.M) {
	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		log.Fatalf("Error loading .env file")
	}

	// Run tests
	os.Exit(m.Run())
}
