package main

import (
	"log"
	"os"
	"testing"

	// "golang.org/x/oauth2"
)

func TestFetchRecords(t *testing.T) {
	airtable := &Airtable{
		BaseURL: "https://api.airtable.com/v0",
		BaseID:  os.Getenv("BASE_ID"),
		DBPath:  "cache_test.db",
	}
	err := airtable.init()
	if err != nil {
		t.Errorf("init() error = %v", err)
	}

	params := map[string]interface{}{
		"filterByFormula": "IS_AFTER(LAST_MODIFIED_TIME(),'2024-12-01T00:00:00Z')",
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
		BaseID:  os.Getenv("BASE_ID"),
		DBPath:  "cache_test.db",
	}
	err := airtable.init()
	if err != nil {
		t.Errorf("init() error = %v", err)
	}

	err = airtable.fetchSchema()
	if err != nil {
		t.Errorf("fetchSchema() error = %v", err)
	}

	tags, _ := airtable.Cache.getData("Tags")
	if tags == nil || *tags == "" {
		t.Errorf("fetchSchema() did not cache tags")
	}
	log.Println(tags)

	categories, _ := airtable.Cache.getData("Categories")
	if categories == nil || *categories == "" {
		t.Errorf("fetchSchema() did not cache categories")
	}
	log.Println(categories)
}

func TestCreateRecords(t *testing.T) {
	airtable := &Airtable{
		BaseURL: "https://api.airtable.com/v0",
		BaseID:  os.Getenv("BASE_ID"),
		DBPath:  "cache_test.db",
	}
	err := airtable.init()
	if err != nil {
		t.Errorf("init() error = %v", err)
	}

	records := []*Record{
		{
			Fields: &map[string]interface{}{
				"Name": "Test Link",
				"Note": "Test Note",
				"URL":  "http://example.com",
				// "Category": "",
				"Tags":  []string{},
				"Done":  false,
				"Lists": []string{},
			},
		},
	}

	records, err = airtable.createRecords("Links", records)
	if err != nil {
		t.Errorf("createRecords() error = %v", err)
	}
	record := records[0]
	log.Println("test", *record.ID)
}

func TestUpdateRecords(t *testing.T) {
	airtable := &Airtable{
		BaseURL: "https://api.airtable.com/v0",
		BaseID:  os.Getenv("BASE_ID"),
		DBPath:  "cache_test.db",
	}
	err := airtable.init()
	if err != nil {
		t.Errorf("init() error = %v", err)
	}

	records := []*Record{
		{
			Fields: &map[string]interface{}{
				"Name": "Test Link",
				"Note": "Test Note",
				"URL":  "http://example.com",
				// "Category": "",
				"Tags":  []string{},
				"Done":  false,
				"Lists": []string{},
			},
		},
	}

	records, err = airtable.createRecords("Links", records)
	if err != nil {
		t.Errorf("createRecords() error = %v", err)
	}

	record := records[0]
	record.Fields = &map[string]interface{}{}
	(*record.Fields)["Name"] = "Updated Link"
	(*record.Fields)["Note"] = "Updated Note"

	log.Println("test", *record.ID, record.Fields)

	_, err = airtable.updateRecords("Links", records)
	if err != nil {
		t.Errorf("updateRecords() error = %v", err)
	}
}

func TestDeleteRecords(t *testing.T) {
	airtable := &Airtable{
		BaseURL: "https://api.airtable.com/v0",
		BaseID:  os.Getenv("BASE_ID"),
		DBPath:  "cache_test.db",
	}
	err := airtable.init()
	if err != nil {
		t.Errorf("init() error = %v", err)
	}

	records := []*Record{
		{
			ID: stringPtr("recFDMPdTXU6jkLJu"),
		},
	}

	err = airtable.deleteRecords("Links", records)
	if err != nil {
		t.Errorf("deleteRecords() error = %v", err)
	}
}
