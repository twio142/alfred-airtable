package main

import (
	"bytes"
	"context"
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

func (a *Auth) refreshValid() bool {
	return a.RefreshToken != "" && a.RefreshExpiry != nil && a.RefreshExpiry.After(time.Now())
}

func (a *Auth) read(c *Cache) {
	if a.Token != nil && a.AccessToken != "" {
		return
	}
	if accessToken, err := c.getData("AccessToken"); err == nil {
		a.AccessToken = *accessToken
	}
	if expiry, err := c.getData("Expiry"); err == nil {
		expiryInt, err := strconv.ParseInt(*expiry, 10, 64)
		if err == nil {
			a.Expiry = time.Unix(expiryInt, 0)
		}
	}
	if refreshToken, err := c.getData("RefreshToken"); err == nil {
		a.RefreshToken = *refreshToken
	}
	if RefreshExpiry, err := c.getData("RefreshExpiry"); err == nil {
		RefreshExpiryInt, err := strconv.ParseInt(*RefreshExpiry, 10, 64)
		if err == nil {
			RefreshExpiryTime := time.Unix(RefreshExpiryInt, 0)
			a.RefreshExpiry = &RefreshExpiryTime
		}
	}
}

func (a *Auth) write(c *Cache) {
	if a.Token == nil {
		return
	}
	if a.AccessToken != "" {
		_ = c.setData("AccessToken", a.AccessToken)
	}
	if !a.Expiry.IsZero() {
		_ = c.setData("Expiry", strconv.FormatInt(a.Expiry.Unix(), 10))
	}
	if a.RefreshToken != "" {
		_ = c.setData("RefreshToken", a.RefreshToken)
	}
	if !a.RefreshExpiry.IsZero() {
		_ = c.setData("RefreshExpiry", strconv.FormatInt(a.RefreshExpiry.Unix(), 10))
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
	o.authComplete = make(chan Auth)
	o.authorizationCache = make(map[string]string)
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
	if o.authorizationCache == nil {
		o.authorizationCache = make(map[string]string)
	}
	o.authorizationCache[state] = codeVerifier

	authCodeURL := o.config.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.SetAuthURLParam("code_challenge", codeChallenge), oauth2.SetAuthURLParam("code_challenge_method", "S256"))

	http.Redirect(w, r, authCodeURL, http.StatusFound)
}

func (o *OAuth) handleAirtableOAuth(w http.ResponseWriter, r *http.Request) {
	state := r.URL.Query().Get("state")
	codeVerifier, ok := o.authorizationCache[state]
	if !ok {
		http.Error(w, "This request was not from Airtable!", http.StatusBadRequest)
		o.authComplete <- Auth{}
		return
	}
	delete(o.authorizationCache, state)

	if err := r.URL.Query().Get("error"); err != "" {
		errorDescription := r.URL.Query().Get("error_description")
		fmt.Fprintf(w, "There was an error authorizing this request.\nError: \"%s\"\nError Description: \"%s\"", err, errorDescription)
		o.authComplete <- Auth{}
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
		o.authComplete <- Auth{}
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
		o.authComplete <- Auth{}
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		o.authComplete <- Auth{}
		return
	}

	fmt.Fprintln(w, "Success! You can close this tab now.")

	var responseData map[string]interface{}
	if err = json.Unmarshal(body, &responseData); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		o.authComplete <- Auth{}
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

	o.authComplete <- *newAuth
}

func (o *OAuth) handleRefresh(w http.ResponseWriter, r *http.Request) {
    defer func() {
        if r := recover(); r != nil {
            log.Println("Recovered in handleRefresh:", r)
            o.authComplete <- Auth{}
        }
    }()

    refreshToken := r.URL.Query().Get("refresh_token")
    data := url.Values{}
    data.Set("client_id", o.config.ClientID)
    data.Set("refresh_token", refreshToken)
    data.Set("scope", strings.Join(o.config.Scopes, " "))
    data.Set("grant_type", "refresh_token")

    req, err := http.NewRequest("POST", o.config.Endpoint.TokenURL, bytes.NewBufferString(data.Encode()))
    if err != nil {
        http.Error(w, "Internal Server Error", http.StatusInternalServerError)
        o.authComplete <- Auth{}
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
        o.authComplete <- Auth{}
        return
    }
    defer resp.Body.Close()

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        http.Error(w, "Internal Server Error", http.StatusInternalServerError)
        o.authComplete <- Auth{}
        return
    }

    var responseData map[string]interface{}
    if err = json.Unmarshal(body, &responseData); err != nil {
        http.Error(w, "Internal Server Error", http.StatusInternalServerError)
        o.authComplete <- Auth{}
        return
    }

    if responseData["error"] != nil {
        errorDescription := responseData["error_description"].(string)
        fmt.Fprintf(w, "There was an error refreshing the token.\nError: \"%s\"\nError Description: \"%s\"", responseData["error"].(string), errorDescription)
        o.authComplete <- Auth{}
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

    o.authComplete <- *newAuth
}

func (o *OAuth) startServer() *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/", o.handleRoot)
	mux.HandleFunc("/airtable-oauth", o.handleAirtableOAuth)
	mux.HandleFunc("/refresh", o.handleRefresh)

	redirectURL := o.config.RedirectURL
	u, err := url.Parse(redirectURL)
	if err != nil {
		log.Fatal(err)
		return nil
	}
	port := u.Port()
	if u.Hostname() != "localhost" || port == "" {
		log.Fatal("Redirect URL must be http://localhost:port")
	}
	log.Printf("Server listening on port %s", port)

	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Could not listen on :%s: %v\n", port, err)
		}
	}()

	return server
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

	var server *http.Server

	auth := Auth{
		Token: &oauth2.Token{},
	}
	auth.read(a.cache)
	a.auth = &auth
	if a.auth.Valid() {
		logMessage("INFO", "Using cached auth")
		return nil
	} else if a.auth.refreshValid() {
		server = o.startServer()
		defer server.Shutdown(context.Background())

		if err := exec.Command("curl", baseURL+"/refresh?refresh_token="+a.auth.RefreshToken).Start(); err != nil {
			logMessage("ERROR", "Failed to refresh token: %v", err)
			return err
		}

		newAuth := <-o.authComplete

		if newAuth.Token != nil && newAuth.AccessToken != "" {
			a.auth = &newAuth
			newAuth.write(a.cache)
			logMessage("INFO", "Refreshed auth")
			return nil
		}
	}

	if server == nil {
		server = o.startServer()
    defer server.Shutdown(context.Background())
	}

	if err := exec.Command("open", baseURL).Start(); err != nil {
		return err
	}

	newAuth := <-o.authComplete
	newAuth.write(a.cache)
	a.auth = &newAuth

	if newAuth.Token == nil || newAuth.AccessToken == "" {
		logMessage("ERROR", "Failed to get auth")
		return fmt.Errorf("failed to get auth")
	}

	logMessage("INFO", "Got auth")
	return nil
}
