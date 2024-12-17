package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "strconv"
    "log"
    "net/http"
    "net/url"
    "os"
    "os/exec"
    "time"
)

type Auth struct {
	AccessToken *string
	ExpiresAt   *int64
	RefreshToken *string
	RefreshExpiresAt *int64
}

func (a *Auth) isValid() bool {
    return a.AccessToken != nil && a.ExpiresAt != nil && *a.ExpiresAt > time.Now().Unix()
}

func (a *Auth) isRefreshValid() bool {
    return a.RefreshToken != nil && a.RefreshExpiresAt != nil && *a.RefreshExpiresAt > time.Now().Unix()
}

func (a *Auth) read(c *Cache) {
    if accessToken, err := c.getData("AccessToken"); err == nil {
        a.AccessToken = accessToken
    }
    if expiresAt, err := c.getData("ExpiresAt"); err == nil {
        expiresAtInt, err := strconv.ParseInt(*expiresAt, 10, 64)
        if err == nil {
            a.ExpiresAt = &expiresAtInt
        }
    }
    if refreshToken, err := c.getData("RefreshToken"); err == nil {
        a.RefreshToken = refreshToken
    }
    if refreshExpiresAt, err := c.getData("RefreshExpiresAt"); err == nil {
        refreshExpiresAtInt, err := strconv.ParseInt(*refreshExpiresAt, 10, 64)
        if err == nil {
            a.RefreshExpiresAt = &refreshExpiresAtInt
        }
    }
}

func (a *Auth) write(c *Cache) {
    if a.AccessToken != nil {
        c.setData("AccessToken", *a.AccessToken)
    }
    if a.ExpiresAt != nil {
        c.setData("ExpiresAt", strconv.FormatInt(*a.ExpiresAt, 10))
    }
    if a.RefreshToken != nil {
        c.setData("RefreshToken", *a.RefreshToken)
    }
    if a.RefreshExpiresAt != nil {
        c.setData("RefreshExpiresAt", strconv.FormatInt(*a.RefreshExpiresAt, 10))
    }
}

type OAuth struct {
    clientId           string
    clientSecret       string
    port               string
    redirectUri        string
    scope              string
    airtableUrl        string
    authorizationCache map[string]string
    authComplete       chan Auth
}

func (o *OAuth) init() {
    o.clientId = os.Getenv("CLIENT_ID")
    o.clientSecret = os.Getenv("CLIENT_SECRET")
		o.port = os.Getenv("PORT")
 		o.redirectUri = os.Getenv("REDIRECT_URI")
    o.scope = os.Getenv("SCOPE")
    if o.scope == "" {
        o.scope = "data.records:read data.records:write schema.bases:read schema.bases:write"
    }
    o.airtableUrl = os.Getenv("AIRTABLE_URL")
    if o.airtableUrl == "" {
        o.airtableUrl = "https://www.airtable.com"
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

    authorizationUrl := fmt.Sprintf("%s/oauth2/v1/authorize?code_challenge=%s&code_challenge_method=S256&state=%s&client_id=%s&redirect_uri=%s&response_type=code&scope=%s",
        o.airtableUrl, codeChallenge, state, o.clientId, o.redirectUri, o.scope)

    http.Redirect(w, r, authorizationUrl, http.StatusFound)
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
    data.Set("client_id", o.clientId)
    data.Set("code_verifier", codeVerifier)
    data.Set("redirect_uri", o.redirectUri)
    data.Set("code", code)
    data.Set("grant_type", "authorization_code")

    req, err := http.NewRequest("POST", fmt.Sprintf("%s/oauth2/v1/token", o.airtableUrl), bytes.NewBufferString(data.Encode()))
    if err != nil {
        http.Error(w, "Internal Server Error", http.StatusInternalServerError)
        return
    }
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
    if o.clientSecret != "" {
        req.SetBasicAuth(o.clientId, o.clientSecret)
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
    if err := json.Unmarshal(body, &responseData); err != nil {
        http.Error(w, "Internal Server Error", http.StatusInternalServerError)
        return
    }

    responseData["expires_at"] = responseData["expires_in"].(float64) + float64(time.Now().Unix())
    responseData["refresh_expires_at"] = responseData["refresh_expires_in"].(float64) + float64(time.Now().Unix())

    credentials, err := json.Marshal(responseData)
    if err != nil {
        http.Error(w, "Internal Server Error", http.StatusInternalServerError)
        return
    }

    if err := os.WriteFile(".credentials.json", credentials, 0644); err != nil {
        http.Error(w, "Internal Server Error", http.StatusInternalServerError)
        return
    }

    expiresAt := int64(responseData["expires_in"].(float64)) + time.Now().Unix()
    refreshExpiresAt := int64(responseData["refresh_expires_in"].(float64)) + time.Now().Unix()

    newAuth := Auth{
        AccessToken:      stringPtr(responseData["access_token"].(string)),
        ExpiresAt:        &expiresAt,
        RefreshToken:     stringPtr(responseData["refresh_token"].(string)),
        RefreshExpiresAt: &refreshExpiresAt,
    }
    o.authComplete <- newAuth
}

func (o *OAuth) handleRefresh(w http.ResponseWriter, r *http.Request) {
    refreshToken := r.URL.Query().Get("refresh_token")
    data := url.Values{}
    data.Set("client_id", o.clientId)
    data.Set("refresh_token", refreshToken)
    data.Set("scope", o.scope)
    data.Set("grant_type", "refresh_token")

    req, err := http.NewRequest("POST", fmt.Sprintf("%s/oauth2/v1/token", o.airtableUrl), bytes.NewBufferString(data.Encode()))
    if err != nil {
        http.Error(w, "Internal Server Error", http.StatusInternalServerError)
        return
    }
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
    if o.clientSecret != "" {
        req.SetBasicAuth(o.clientId, o.clientSecret)
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
    if err := json.Unmarshal(body, &responseData); err != nil {
        http.Error(w, "Internal Server Error", http.StatusInternalServerError)
        return
    }

    responseData["expires_at"] = responseData["expires_in"].(float64) + float64(time.Now().Unix())
    responseData["refresh_expires_at"] = responseData["refresh_expires_in"].(float64) + float64(time.Now().Unix())

    credentials, err := json.Marshal(responseData)
    if err != nil {
        http.Error(w, "Internal Server Error", http.StatusInternalServerError)
        return
    }

    if err := os.WriteFile(".credentials.json", credentials, 0644); err != nil {
        http.Error(w, "Internal Server Error", http.StatusInternalServerError)
        return
    }

    expiresAt := int64(responseData["expires_in"].(float64)) + time.Now().Unix()
    refreshExpiresAt := int64(responseData["refresh_expires_in"].(float64)) + time.Now().Unix()

    newAuth := Auth{
        AccessToken:      stringPtr(responseData["access_token"].(string)),
        ExpiresAt:        &expiresAt,
        RefreshToken:     stringPtr(responseData["refresh_token"].(string)),
        RefreshExpiresAt: &refreshExpiresAt,
    }
    o.authComplete <- newAuth
}

func (o *OAuth) startServer() {
    http.HandleFunc("/", o.handleRoot)
    http.HandleFunc("/airtable-oauth", o.handleAirtableOAuth)
    http.HandleFunc("/refresh", o.handleRefresh)

    log.Printf("Server listening on port %s", o.port)
    if err := http.ListenAndServe(":"+o.port, nil); err != nil {
        log.Fatal(err)
    }
}

func getAuth(c *Cache) (Auth, error) {
    o := OAuth{}
    o.init()

    auth := Auth{}
    if auth.isValid() {
        return auth, nil
    } else if auth.isRefreshValid() {
        go o.startServer()
        resp, err := http.Get(fmt.Sprintf("http://localhost:%s/refresh?refresh_token=%s", o.port, *auth.RefreshToken))
        if err != nil {
            return Auth{}, err
        }
        defer resp.Body.Close()

        body, err := io.ReadAll(resp.Body)
        if err != nil {
            return Auth{}, err
        }

        var responseData map[string]interface{}
        if err := json.Unmarshal(body, &responseData); err != nil {
            return Auth{}, err
        }

        expiresAt := int64(responseData["expires_in"].(float64)) + time.Now().Unix()
        refreshExpiresAt := int64(responseData["refresh_expires_in"].(float64)) + time.Now().Unix()

        newAuth := Auth{
            AccessToken:      stringPtr(responseData["access_token"].(string)),
            ExpiresAt:        &expiresAt,
            RefreshToken:     stringPtr(responseData["refresh_token"].(string)),
            RefreshExpiresAt: &refreshExpiresAt,
        }
        return newAuth, nil
    }

    go o.startServer()
    cmd := exec.Command("open", fmt.Sprintf("http://localhost:%s", o.port))
    if err := cmd.Start(); err != nil {
        return Auth{}, err
    }

    newAuth := <-o.authComplete
    newAuth.write(c)
    return newAuth, nil
}
