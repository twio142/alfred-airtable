package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

const (
	baseURL = "https://api.airtable.com/v0/"
	apiKey  = "your_api_key"
	baseID  = "your_base_id"
)

type Record struct {
	ID     string                 `json:"id"`
	Fields map[string]interface{} `json:"fields"`
}

type Response struct {
	Records []Record `json:"records"`
}

func fetchRecords(tableName string) ([]Record, error) {
	url := fmt.Sprintf("%s%s/%s", baseURL, baseID, tableName)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var response Response
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}

	return response.Records, nil
}

func createRecord(tableName string, fields map[string]interface{}) (Record, error) {
	url := fmt.Sprintf("%s%s/%s", baseURL, baseID, tableName)
	data := map[string]interface{}{
		"fields": fields,
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		return Record{}, err
	}

	req, err := http.NewRequest("POST", url, strings.NewReader(string(jsonData)))
	if err != nil {
		return Record{}, err
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return Record{}, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return Record{}, err
	}

	var record Record
	if err := json.Unmarshal(body, &record); err != nil {
		return Record{}, err
	}

	return record, nil
}

func updateRecord(tableName, recordID string, fields map[string]interface{}) (Record, error) {
	url := fmt.Sprintf("%s%s/%s/%s", baseURL, baseID, tableName, recordID)
	data := map[string]interface{}{
		"fields": fields,
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		return Record{}, err
	}

	req, err := http.NewRequest("PATCH", url, strings.NewReader(string(jsonData)))
	if err != nil {
		return Record{}, err
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return Record{}, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return Record{}, err
	}

	var record Record
	if err := json.Unmarshal(body, &record); err != nil {
		return Record{}, err
	}

	return record, nil
}

func deleteRecord(tableName, recordID string) error {
	url := fmt.Sprintf("%s%s/%s/%s", baseURL, baseID, tableName, recordID)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	client := &http.Client{}
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
