package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/time/rate"
)

var (
	rateLimiter = rate.NewLimiter(5, 1)
	ctx         = context.Background()
)

func throttle() error {
	return rateLimiter.Wait(ctx)
}

// Interact with the Airtable API

type Airtable struct {
	baseURL string
	baseID  string
	auth    *Auth
	dbPath  string
	cache   *Cache
}

type Record struct {
	ID          *string                 `json:"id,omitempty"`
	CreatedTime *time.Time              `json:"createdTime,omitempty"`
	Fields      *map[string]interface{} `json:"fields,omitempty"`
}

type Response struct {
	Records []Record `json:"records"`
	Offset  *string  `json:"offset,omitempty"`
}

func (a *Airtable) init(skipAuth ...bool) error {
	a.cache = &Cache{file: a.dbPath}
	if err := a.cache.init(); err != nil {
		return err
	}
	if len(skipAuth) > 0 && skipAuth[0] {
	} else if err := a.getAuth(); err != nil {
		return err
	}
	return nil
}

func (a *Airtable) fetchRecords(tableName string, params map[string]interface{}) ([]Record, error) {
	if err := throttle(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf("%s/%s/%s", a.baseURL, a.baseID, tableName)
	searchParams := []string{}
	for key, value := range params {
		if str, ok := value.(string); ok {
			searchParams = append(searchParams, fmt.Sprintf("%s=%s", url.QueryEscape(key), url.QueryEscape(str)))
		} else if slice, ok := value.([]string); ok {
			for _, str := range slice {
				searchParams = append(searchParams, fmt.Sprintf("%s[]=%s", url.QueryEscape(key), url.QueryEscape(str)))
			}
		}
	}
	u = u + "?" + strings.Join(searchParams, "&")
	client := &http.Client{}
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", "Bearer "+a.auth.AccessToken)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logMessage("ERROR", "Failed to fetch records: %s", resp.Status)
		return nil, fmt.Errorf("failed to fetch records: %s", resp.Status)
	}

	var response Response
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	if response.Offset != nil {
		params["offset"] = *response.Offset
		if records, err := a.fetchRecords(tableName, params); err == nil {
			response.Records = append(response.Records, records...)
		} else {
			logMessage("ERROR", "Failed to fetch additional records: %s", err)
			return nil, err
		}
	}

	logMessage("INFO", "Fetched %d records", len(response.Records))
	return response.Records, nil
}

func (a *Airtable) fetchSchema() (*[]string, *[]string, error) {
	if err := throttle(); err != nil {
		return nil, nil, err
	}

	u := fmt.Sprintf("https://api.airtable.com/v0/meta/bases/%s/tables", a.baseID)
	client := &http.Client{}
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Add("Authorization", "Bearer "+a.auth.AccessToken)

	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("failed to fetch schema: %s", resp.Status)
	}

	type MetaResponse struct {
		Tables []struct {
			Name   string `json:"name"`
			Fields []struct {
				Name    string `json:"name"`
				Options *struct {
					Choices *[]struct {
						Name string `json:"name"`
					} `json:"choices"`
				} `json:"options"`
			} `json:"fields"`
		} `json:"tables"`
	}

	var response MetaResponse
	if err = json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, nil, err
	}

	tags := []string{}
	categories := []string{}

	for _, table := range response.Tables {
		if table.Name == "Links" {
			for _, field := range table.Fields {
				switch field.Name {
				case "Tags":
					if field.Options != nil {
						for _, choice := range *field.Options.Choices {
							tags = append(tags, choice.Name)
						}
					}
				case "Category":
					if field.Options != nil {
						for _, choice := range *field.Options.Choices {
							categories = append(categories, choice.Name)
						}
					}
				}
			}
		}
	}

	logMessage("INFO", "Fetched %d tags and %d categories", len(tags), len(categories))
	return &tags, &categories, nil
}

func (a *Airtable) createRecords(tableName string, records *[]*Record) error {
	if err := throttle(); err != nil {
		return err
	}

	u := fmt.Sprintf("%s/%s/%s", a.baseURL, a.baseID, tableName)
	client := &http.Client{}

	data := map[string]interface{}{
		"records": records,
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", u, strings.NewReader(string(jsonData)))
	if err != nil {
		return err
	}
	req.Header.Add("Authorization", "Bearer "+a.auth.AccessToken)
	req.Header.Add("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to create records: %s", resp.Status)
	}

	var response Response
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return err
	}

	*records = make([]*Record, len(response.Records))
	for i, record := range response.Records {
		(*records)[i] = &record
	}

	logMessage("INFO", "Created %d records", len(*records))
	return nil
}

func (a *Airtable) updateRecords(tableName string, records *[]*Record) error {
	if err := throttle(); err != nil {
		return err
	}

	for _, record := range *records {
		if record == nil || record.ID == nil {
			return fmt.Errorf("record with an ID is required for update")
		}
		record.CreatedTime = nil
	}
	u := fmt.Sprintf("%s/%s/%s", a.baseURL, a.baseID, tableName)
	client := &http.Client{}

	data := map[string]interface{}{
		"records": records,
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("PATCH", u, strings.NewReader(string(jsonData)))
	if err != nil {
		return err
	}
	req.Header.Add("Authorization", "Bearer "+a.auth.AccessToken)
	req.Header.Add("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to update records: %s", resp.Status)
	}

	var response Response
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return err
	}

	*records = make([]*Record, len(response.Records))
	for i, record := range response.Records {
		(*records)[i] = &record
	}

	logMessage("INFO", "Updated %d records", len(*records))
	return nil
}

func (a *Airtable) deleteRecords(tableName string, records *[]*Record) error {
	if err := throttle(); err != nil {
		return err
	}

	u := fmt.Sprintf("%s/%s/%s", a.baseURL, a.baseID, tableName)
	searchParams := []string{}
	for _, record := range *records {
		if record == nil || record.ID == nil {
			return fmt.Errorf("record with an ID is required for delete")
		}
		searchParams = append(searchParams, fmt.Sprintf("records[]=%s", *record.ID))
	}
	u = u + "?" + strings.Join(searchParams, "&")
	client := &http.Client{}
	req, err := http.NewRequest("DELETE", u, nil)
	if err != nil {
		return err
	}
	req.Header.Add("Authorization", "Bearer "+a.auth.AccessToken)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to delete record: %s", resp.Status)
	}

	logMessage("INFO", "Deleted %d records", len(*records))
	return nil
}
