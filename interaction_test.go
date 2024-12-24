package main

import (
	"os"
	"testing"
)

func TestListLists(t *testing.T) {
	airtable := &Airtable{
		baseURL: "https://api.airtable.com/v0",
		baseID:  os.Getenv("BASE_ID"),
		dbPath:  "airtable.db",
	}
	err := airtable.init(true)
	if err != nil {
		t.Errorf("init() error = %v", err)
	}
	airtable.listLists()
}

func TestListLinks(t *testing.T) {
	airtable := &Airtable{
		baseURL: "https://api.airtable.com/v0",
		baseID:  os.Getenv("BASE_ID"),
		dbPath:  "airtable.db",
	}
	err := airtable.init(true)
	if err != nil {
		t.Errorf("init() error = %v", err)
	}
	lists, err := airtable.cache.getLists(nil)
	if err != nil {
		t.Errorf("getLists() error = %v", err)
	}
	airtable.listLinks(&lists[0])
}

func TestEditLink(t *testing.T) {
	airtable := &Airtable{
		baseURL: "https://api.airtable.com/v0",
		baseID:  os.Getenv("BASE_ID"),
		dbPath:  "airtable.db",
	}
	err := airtable.init(true)
	if err != nil {
		t.Errorf("init() error = %v", err)
	}
	os.Setenv("listIDs", "recnzJ8FfXPKbz6Pn")

	airtable.editLink("[Test Title](https://example.com)")
}
