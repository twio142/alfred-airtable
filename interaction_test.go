package main

import (
	"os"
	"testing"
)

func TestListLists(t *testing.T) {
	airtable := &Airtable{
		BaseURL: "https://api.airtable.com/v0",
		BaseID:  os.Getenv("BASE_ID"),
		DBPath:  "airtable.db",
	}
	err := airtable.init(true)
	if err != nil {
		t.Errorf("init() error = %v", err)
	}
	airtable.listLists()
}

func TestListLinks(t *testing.T) {
	airtable := &Airtable{
		BaseURL: "https://api.airtable.com/v0",
		BaseID:  os.Getenv("BASE_ID"),
		DBPath:  "airtable.db",
	}
	err := airtable.init(true)
	if err != nil {
		t.Errorf("init() error = %v", err)
	}
	lists, err := airtable.Cache.getLists(nil)
	if err != nil {
		t.Errorf("getLists() error = %v", err)
	}
	airtable.listLinks(&lists[0])
}

func TestEditLink(t *testing.T) {
	airtable := &Airtable{
		BaseURL: "https://api.airtable.com/v0",
		BaseID:  os.Getenv("BASE_ID"),
		DBPath:  "airtable.db",
	}
	err := airtable.init(true)
	if err != nil {
		t.Errorf("init() error = %v", err)
	}

	airtable.editLink("[Test Title](https://example.com)")
}
