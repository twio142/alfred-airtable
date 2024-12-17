package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestFetchRecords(t *testing.T) {
	airtable := &Airtable{
		BaseURL: "https://api.airtable.com/v0",
		BaseID:  "app1234567890",
		Auth: &Auth{
			AccessToken: "test_token",
		},
	}

	params := map[string]interface{}{
		"filterByFormula": "IS_AFTER(LAST_MODIFIED_TIME(),'2022-01-01T00:00:00.000Z')",
		"fields":          []string{"Name", "Note", "URL", "Category", "Tags", "Last Modified", "Record URL", "Done", "Lists"},
	}

	records, err := airtable.fetchRecords("Links", params)
	if err != nil {
		t.Errorf("fetchRecords() error = %v", err)
	}

	if len(records) == 0 {
		t.Errorf("fetchRecords() returned no records")
	}
}

func TestFetchSchema(t *testing.T) {
	airtable := &Airtable{
		BaseURL: "https://api.airtable.com/v0",
		BaseID:  "app1234567890",
		Auth: &Auth{
			AccessToken: "test_token",
		},
		Cache: &Cache{},
	}

	err := airtable.fetchSchema()
	if err != nil {
		t.Errorf("fetchSchema() error = %v", err)
	}

	tags, _ := airtable.Cache.getData("Tags")
	if tags == nil || *tags == "" {
		t.Errorf("fetchSchema() did not cache tags")
	}

	categories, _ := airtable.Cache.getData("Categories")
	if categories == nil || *categories == "" {
		t.Errorf("fetchSchema() did not cache categories")
	}
}

func TestCreateRecords(t *testing.T) {
	airtable := &Airtable{
		BaseURL: "https://api.airtable.com/v0",
		BaseID:  "app1234567890",
		Auth: &Auth{
			AccessToken: "test_token",
		},
	}

	records := []*Record{
		{
			Fields: &map[string]interface{}{
				"Name":     "Test Link",
				"Note":     "Test Note",
				"URL":      "http://example.com",
				"Category": "Test Category",
				"Tags":     []string{"Test Tag"},
				"Done":     false,
				"Lists":    []string{"Test List ID"},
			},
		},
	}

	err := airtable.createRecords("Links", records)
	if err != nil {
		t.Errorf("createRecords() error = %v", err)
	}
}

func TestUpdateRecords(t *testing.T) {
	airtable := &Airtable{
		BaseURL: "https://api.airtable.com/v0",
		BaseID:  "app1234567890",
		Auth: &Auth{
			AccessToken: "test_token",
		},
	}

	records := []*Record{
		{
			ID: stringPtr("rec1234567890"),
			Fields: &map[string]interface{}{
				"Name":     "Updated Test Link",
				"Note":     "Updated Test Note",
				"URL":      "http://example.com",
				"Category": "Updated Test Category",
				"Tags":     []string{"Updated Test Tag"},
				"Done":     false,
				"Lists":    []string{"Updated Test List ID"},
			},
		},
	}

	err := airtable.updateRecords("Links", records)
	if err != nil {
		t.Errorf("updateRecords() error = %v", err)
	}
}

func TestDeleteRecords(t *testing.T) {
	airtable := &Airtable{
		BaseURL: "https://api.airtable.com/v0",
		BaseID:  "app1234567890",
		Auth: &Auth{
			AccessToken: "test_token",
		},
	}

	records := []*Record{
		{
			ID: stringPtr("rec1234567890"),
		},
	}

	err := airtable.deleteRecords("Links", records)
	if err != nil {
		t.Errorf("deleteRecords() error = %v", err)
	}
}
