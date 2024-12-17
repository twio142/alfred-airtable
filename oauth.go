package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"golang.org/x/oauth2"
)

type Auth struct {
	*oauth2.Token
	RefreshExpiry *time.Time
}

type OAuth struct {
	authorizationCache map[string]string
	authComplete       chan Auth
	config             *oauth2.Config
}

func (a *Auth) isValid() bool {
	return a.Token != nil && a.Expiry.After(time.Now())
}

func (a *Auth) isRefreshValid() bool {
	return a.RefreshToken != "" && a.RefreshExpiry != nil && a.RefreshExpiry.After(time.Now())
}

func (a *Auth) read(c *Cache) {
	if accessToken, err := c.getData("AccessToken"); err == nil {
		a.AccessToken = *accessToken
	}
	if expiresAt, err := c.getData("Expiry"); err == nil {
		expiresAtInt, err := strconv.ParseInt(*expiresAt, 10, 64)
		if err == nil {
			a.Expiry = time.Unix(expiresAtInt, 0)
		}
	}
	if refreshToken, err := c.getData("RefreshToken"); err == nil {
		a.RefreshToken = *refreshToken
	}
	if RefreshExpiry, err := c.getData("RefreshExpiry"); err == nil {
		RefreshExpiryInt, err := strconv.ParseInt(*RefreshExpiry, 10, 64)
		RefreshExpiryTime := time.Unix(RefreshExpiryInt, 0)
		if err == nil {
			a.RefreshExpiry = &RefreshExpiryTime
		}
	}
}

func (a *Auth) write(c *Cache) {
	if a.AccessToken != "" {
		c.setData("AccessToken", a.AccessToken)
	}
	if !a.Expiry.IsZero() {
		c.setData("Expiry", strconv.FormatInt(a.Expiry.Unix(), 10))
	}
	if a.RefreshToken != "" {
		c.setData("RefreshToken", a.RefreshToken)
	}
	if !a.RefreshExpiry.IsZero() {
		c.setData("RefreshExpiry", strconv.FormatInt(a.RefreshExpiry.Unix(), 10))
	}
}

func (o *OAuth) init() {
	o.config = &oauth2.Config{
		ClientID:    os.Getenv("CLIENT_ID"),
		RedirectURL: os.Getenv("REDIRECT_URI"),
		Scopes:      []string{"data.records:read", "data.records:write", "schema.bases:read", "schema.bases:write"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://www.airtable.com/oauth2/v1/authorize",
			TokenURL: "https://www.airtable.com/oauth2/v1/token",
		},
	}
}

func (o *OAuth) handleRoot(w http.ResponseWriter, r *http.Request) {
	state, err := randomString(100)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	codeVerifier, err := randomString(96)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	codeChallenge := createCodeChallenge(codeVerifier)
	o.authorizationCache[state] = codeVerifier

	authCodeURL := o.config.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.SetAuthURLParam("code_challenge", codeChallenge), oauth2.SetAuthURLParam("code_challenge_method", "S256"))

	http.Redirect(w, r, authCodeURL, http.StatusFound)
}

func (o *OAuth) handleAirtableOAuth(w http.ResponseWriter, r *http.Request) {
	state := r.URL.Query().Get("state")
	codeVerifier, ok := o.authorizationCache[state]
	if !ok {
		http.Error(w, "This request was not from Airtable!", http.StatusBadRequest)
		return
	}
	delete(o.authorizationCache, state)

	if err := r.URL.Query().Get("error"); err != "" {
		errorDescription := r.URL.Query().Get("error_description")
		fmt.Fprintf(w, "There was an error authorizing this request.\nError: \"%s\"\nError Description: \"%s\"", err, errorDescription)
		return
	}

	code := r.URL.Query().Get("code")
	data := url.Values{}
	data.Set("client_id", o.config.ClientID)
	data.Set("code_verifier", codeVerifier)
	data.Set("redirect_uri", o.config.RedirectURL)
	data.Set("code", code)
	data.Set("grant_type", "authorization_code")

	req, err := http.NewRequest("POST", o.config.Endpoint.TokenURL, bytes.NewBufferString(data.Encode()))
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if o.config.ClientSecret != "" {
		req.SetBasicAuth(o.config.ClientID, o.config.ClientSecret)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	var responseData map[string]interface{}
	if err = json.Unmarshal(body, &responseData); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	expiry := time.Now().Add(time.Duration(responseData["expires_in"].(float64)) * time.Second)
	refreshExpiry := time.Now().Add(time.Duration(responseData["refresh_expires_in"].(float64)) * time.Second)

	newAuth := &Auth{
		Token: &oauth2.Token{
			AccessToken:  responseData["access_token"].(string),
			TokenType:    responseData["token_type"].(string),
			RefreshToken: responseData["refresh_token"].(string),
			Expiry:       expiry,
		},
		RefreshExpiry: &refreshExpiry,
	}

	credentials, err := json.Marshal(newAuth)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if err := os.WriteFile(".credentials.json", credentials, 0644); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	o.authComplete <- *newAuth
}

func (o *OAuth) handleRefresh(w http.ResponseWriter, r *http.Request) {
	refreshToken := r.URL.Query().Get("refresh_token")
	data := url.Values{}
	data.Set("client_id", o.config.ClientID)
	data.Set("refresh_token", refreshToken)
	data.Set("scope", strings.Join(o.config.Scopes, " "))
	data.Set("grant_type", "refresh_token")

	req, err := http.NewRequest("POST", o.config.Endpoint.TokenURL, bytes.NewBufferString(data.Encode()))
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if o.config.ClientSecret != "" {
		req.SetBasicAuth(o.config.ClientID, o.config.ClientSecret)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	var responseData map[string]interface{}
	if err = json.Unmarshal(body, &responseData); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	expiry := time.Now().Add(time.Duration(responseData["expires_in"].(float64)) * time.Second)
	refreshExpiry := time.Now().Add(time.Duration(responseData["refresh_expires_in"].(float64)) * time.Second)

	newAuth := &Auth{
		Token: &oauth2.Token{
			AccessToken:  responseData["access_token"].(string),
			TokenType:    responseData["token_type"].(string),
			RefreshToken: responseData["refresh_token"].(string),
			Expiry:       expiry,
		},
		RefreshExpiry: &refreshExpiry,
	}

	credentials, err := json.Marshal(newAuth)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if err := os.WriteFile(".credentials.json", credentials, 0644); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	o.authComplete <- *newAuth
}

func (o *OAuth) startServer() {
	http.HandleFunc("/", o.handleRoot)
	http.HandleFunc("/airtable-oauth", o.handleAirtableOAuth)
	http.HandleFunc("/refresh", o.handleRefresh)

	redirectURL := o.config.RedirectURL
	u, err := url.Parse(redirectURL)
	if err != nil {
		log.Fatal(err)
		return
	}
	port := u.Port()
	log.Printf("Server listening on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}

func (a *Airtable) getAuth() error {
	o := OAuth{}
	o.init()

	redirectURL := o.config.RedirectURL
	u, err := url.Parse(redirectURL)
	if err != nil {
		return err
	}
	baseURL := u.Scheme + "://" + u.Host

	auth := Auth{}
	auth.read(a.Cache)
	if auth.isValid() {
		return nil
	} else if auth.isRefreshValid() {
		go o.startServer()
		resp, err := http.Get(fmt.Sprintf("%s/refresh?refresh_token=%s", baseURL, auth.RefreshToken))
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		var responseData map[string]interface{}
		if err := json.Unmarshal(body, &responseData); err != nil {
			return err
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

		return nil
	}

	go o.startServer()
	cmd := exec.Command("open", baseURL)
	if err := cmd.Start(); err != nil {
		return err
	}

	newAuth := <-o.authComplete
	newAuth.write(a.Cache)
	a.Auth = &newAuth

	return nil
}
