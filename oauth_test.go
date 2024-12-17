package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

func TestAuth_isValid(t *testing.T) {
	auth := &Auth{
		Token: &oauth2.Token{
			AccessToken: "test_token",
			Expiry:      time.Now().Add(time.Hour),
		},
	}
	if !auth.isValid() {
		t.Errorf("Expected token to be valid")
	}

	auth.Expiry = time.Now().Add(-time.Hour)
	if auth.isValid() {
		t.Errorf("Expected token to be invalid")
	}
}

func TestAuth_isRefreshValid(t *testing.T) {
	auth := &Auth{
		RefreshToken:  "test_refresh_token",
		RefreshExpiry: &[]time.Time{time.Now().Add(time.Hour)}[0],
	}
	if !auth.isRefreshValid() {
		t.Errorf("Expected refresh token to be valid")
	}

	auth.RefreshExpiry = &[]time.Time{time.Now().Add(-time.Hour)}[0]
	if auth.isRefreshValid() {
		t.Errorf("Expected refresh token to be invalid")
	}
}

func TestAuth_read(t *testing.T) {
	cache := &Cache{file: ":memory:"}
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
	cache := &Cache{file: ":memory:"}
	cache.init()

	auth := &Auth{
		Token: &oauth2.Token{
			AccessToken: "test_token",
			Expiry:      time.Now().Add(time.Hour),
		},
		RefreshToken:  "test_refresh_token",
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

func TestOAuth_init(t *testing.T) {
	o := &OAuth{}
	o.init()

	if o.config.ClientID != os.Getenv("CLIENT_ID") {
		t.Errorf("Expected ClientID to be '%s', got '%s'", os.Getenv("CLIENT_ID"), o.config.ClientID)
	}
	if o.config.RedirectURL != os.Getenv("REDIRECT_URI") {
		t.Errorf("Expected RedirectURL to be '%s', got '%s'", os.Getenv("REDIRECT_URI"), o.config.RedirectURL)
	}
}

func TestOAuth_handleRoot(t *testing.T) {
	o := &OAuth{authorizationCache: make(map[string]string)}
	o.init()

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(o.handleRoot)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusFound {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusFound)
	}

	location, err := rr.Result().Location()
	if err != nil {
		t.Fatal(err)
	}

	if location.Scheme != "https" || location.Host != "www.airtable.com" {
		t.Errorf("handler returned unexpected location: got %v", location)
	}
}

func TestOAuth_handleAirtableOAuth(t *testing.T) {
	o := &OAuth{
		authorizationCache: make(map[string]string),
		authComplete:       make(chan Auth, 1),
	}
	o.init()

	state := "test_state"
	codeVerifier := "test_code_verifier"
	o.authorizationCache[state] = codeVerifier

	data := url.Values{}
	data.Set("state", state)
	data.Set("code", "test_code")

	req, err := http.NewRequest("GET", "/airtable-oauth?"+data.Encode(), nil)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(o.handleAirtableOAuth)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
}

func TestOAuth_handleRefresh(t *testing.T) {
	o := &OAuth{authComplete: make(chan Auth, 1)}
	o.init()

	data := url.Values{}
	data.Set("refresh_token", "test_refresh_token")

	req, err := http.NewRequest("GET", "/refresh?"+data.Encode(), nil)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(o.handleRefresh)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
}

func TestOAuth_startServer(t *testing.T) {
	o := &OAuth{}
	o.init()

	go o.startServer()
	time.Sleep(1 * time.Second)

	resp, err := http.Get("http://localhost:" + os.Getenv("PORT"))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if status := resp.StatusCode; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
}

func TestAirtable_getAuth(t *testing.T) {
	a := &Airtable{
		Auth:  &Auth{},
		Cache: &Cache{file: ":memory:"},
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
	auth.read(a.Cache)
	if auth.isValid() {
		return
	} else if auth.isRefreshValid() {
		go o.startServer()
		resp, err := http.Get(baseURL + "/refresh?refresh_token=" + auth.RefreshToken)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}

		var responseData map[string]interface{}
		if err := json.Unmarshal(body, &responseData); err != nil {
			t.Fatal(err)
		}

		expiry := int64(responseData["expires_in"].(float64)) + time.Now().Unix()
		RefreshExpiry := int64(responseData["refresh_expires_in"].(float64)) + time.Now().Unix()
		RefreshExpiryTime := time.Unix(RefreshExpiry, 0)

		newAuth := Auth{
			Token: &oauth2.Token{
				AccessToken:  responseData["access_token"].(string),
				TokenType:    responseData["token_type"].(string),
				RefreshToken: responseData["refresh_token"].(string),
				Expiry:       time.Unix(expiry, 0),
			},
			RefreshExpiry: &RefreshExpiryTime,
		}
		a.Auth = &newAuth

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
