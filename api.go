package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Interact with the Airtable API

type Airtable struct {
	BaseURL      string
	BaseID       string
	Auth         *Auth
	DBPath       string
	Cache        *Cache
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
	a.Cache = &Cache{File: a.DBPath}
	if err := a.Cache.init(); err != nil {
		return err
	}
	if len(skipAuth) > 0 && skipAuth[0] {
	} else if err := a.getAuth(); err != nil {
		return err
	}
	return nil
}

func (a *Airtable) fetchRecords(tableName string, params map[string]interface{}) ([]Record, error) {
	u := fmt.Sprintf("%s/%s/%s", a.BaseURL, a.BaseID, tableName)
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
	req.Header.Add("Authorization", "Bearer "+a.Auth.AccessToken)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
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
		}
	}

	return response.Records, nil
}

func (a *Airtable) fetchSchema() (*[]string, *[]string, error) {
	u := fmt.Sprintf("https://api.airtable.com/v0/meta/bases/%s/tables", a.BaseID)
	client := &http.Client{}
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Add("Authorization", "Bearer "+a.Auth.AccessToken)

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

	return &tags, &categories, nil
}

func (a *Airtable) createRecords(tableName string, records *[]*Record) error {
	u := fmt.Sprintf("%s/%s/%s", a.BaseURL, a.BaseID, tableName)
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
	req.Header.Add("Authorization", "Bearer "+a.Auth.AccessToken)
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
	return nil
}

func (a *Airtable) updateRecords(tableName string, records *[]*Record) error {
	for _, record := range *records {
		if record == nil || record.ID == nil {
			return fmt.Errorf("record with an ID is required for update")
		}
		record.CreatedTime = nil
	}
	u := fmt.Sprintf("%s/%s/%s", a.BaseURL, a.BaseID, tableName)
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
	req.Header.Add("Authorization", "Bearer "+a.Auth.AccessToken)
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
	return nil
}

func (a *Airtable) deleteRecords(tableName string, records *[]*Record) error {
	u := fmt.Sprintf("%s/%s/%s", a.BaseURL, a.BaseID, tableName)
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
	req.Header.Add("Authorization", "Bearer "+a.Auth.AccessToken)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to delete record: %s", resp.Status)
	}

	return nil
}
