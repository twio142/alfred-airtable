package main

import (
	"log"
	"testing"
	"time"
)

func TestInit(t *testing.T) {
	cache := &Cache{File: "airtable.db"}
	err := cache.init()
	if err != nil {
		t.Errorf("init() error = %v", err)
	}
	log.Println(cache.LastSyncedAt)
}

func TestGetLinks(t *testing.T) {
	cache := &Cache{File: ":memory:"}
	err := cache.init()
	if err != nil {
		t.Errorf("init() error = %v", err)
	}

	now := time.Now()
	link := Link{
		Name:         stringPtr("Test Link"),
		Note:         stringPtr("Test Note"),
		URL:          stringPtr("http://example.com"),
		Category:     stringPtr("Test Category"),
		Tags:         []string{"Test Tag"},
		Created:      &now,
		LastModified: &now,
		RecordURL:    stringPtr("http://example.com/record"),
		ID:           stringPtr("Test Link ID"),
		Done:         false,
		ListIDs:      []string{"Test List ID"},
	}

	err = cache.saveLinks([]Link{link})
	if err != nil {
		t.Errorf("saveLinks() error = %v", err)
	}

	links, err := cache.getLinks(nil, nil)
	if err != nil {
		t.Errorf("getLinks() error = %v", err)
	}

	if len(links) != 1 {
		t.Errorf("getLinks() returned %d links, expected 1", len(links))
	}

	if *links[0].Name != "Test Link" {
		t.Errorf("getLinks() returned link with name %s, expected 'Test Link'", *links[0].Name)
	}
}

func TestGetLists(t *testing.T) {
	cache := &Cache{File: ":memory:"}
	_ = cache.init()

	now := time.Now()
	list := List{
		Name:         stringPtr("Test List"),
		Note:         stringPtr("Test Note"),
		Created:      &now,
		LastModified: &now,
		RecordURL:    stringPtr("http://example.com/record"),
		ID:           stringPtr("Test List ID"),
	}

	err := cache.saveLists([]List{list})
	if err != nil {
		t.Errorf("saveLists() error = %v", err)
	}

	lists, err := cache.getLists(nil)
	if err != nil {
		t.Errorf("getLists() error = %v", err)
	}

	if len(lists) != 1 {
		t.Errorf("getLists() returned %d lists, expected 1", len(lists))
	}

	if *lists[0].Name != "Test List" {
		t.Errorf("getLists() returned list with name %s, expected 'Test List'", *lists[0].Name)
	}
}

func TestSaveLinks(t *testing.T) {
	cache := &Cache{File: ":memory:"}
	_ = cache.init()

	now := time.Now()
	link := Link{
		Name:         stringPtr("Test Link"),
		Note:         stringPtr("Test Note"),
		URL:          stringPtr("http://example.com"),
		Category:     stringPtr("Test Category"),
		Tags:         []string{"Test Tag"},
		Created:      &now,
		LastModified: &now,
		RecordURL:    stringPtr("http://example.com/record"),
		ID:           stringPtr("Test Link ID"),
		Done:         false,
		ListIDs:      []string{"Test List ID"},
	}

	err := cache.saveLinks([]Link{link})
	if err != nil {
		t.Errorf("saveLinks() error = %v", err)
	}

	links, err := cache.getLinks(nil, nil)
	if err != nil {
		t.Errorf("getLinks() error = %v", err)
	}

	if len(links) != 1 {
		t.Errorf("getLinks() returned %d links, expected 1", len(links))
	}

	if *links[0].Name != "Test Link" {
		t.Errorf("getLinks() returned link with name %s, expected 'Test Link'", *links[0].Name)
	}
}

func TestSaveLists(t *testing.T) {
	cache := &Cache{File: ":memory:"}
	_ = cache.init()

	now := time.Now()
	list := List{
		Name:         stringPtr("Test List"),
		Note:         stringPtr("Test Note"),
		Created:      &now,
		LastModified: &now,
		RecordURL:    stringPtr("http://example.com/record"),
		ID:           stringPtr("Test List ID"),
	}

	err := cache.saveLists([]List{list})
	if err != nil {
		t.Errorf("saveLists() error = %v", err)
	}

	lists, err := cache.getLists(nil)
	if err != nil {
		t.Errorf("getLists() error = %v", err)
	}

	if len(lists) != 1 {
		t.Errorf("getLists() returned %d lists, expected 1", len(lists))
	}

	if *lists[0].Name != "Test List" {
		t.Errorf("getLists() returned list with name %s, expected 'Test List'", *lists[0].Name)
	}
}

func TestClearDeletedRecords(t *testing.T) {
	cache := &Cache{File: ":memory:"}
	_ = cache.init()

	now := time.Now()
	link := Link{
		Name:         stringPtr("Test Link"),
		Note:         stringPtr("Test Note"),
		URL:          stringPtr("http://example.com"),
		Category:     stringPtr("Test Category"),
		Tags:         []string{"Test Tag"},
		Created:      &now,
		LastModified: &now,
		RecordURL:    stringPtr("http://example.com/record"),
		ID:           stringPtr("Test Link ID"),
		Done:         false,
		ListIDs:      []string{"Test List ID"},
	}

	err := cache.saveLinks([]Link{link})
	if err != nil {
		t.Errorf("saveLinks() error = %v", err)
	}

	links, err := cache.getLinks(nil, nil)
	if err != nil {
		t.Errorf("getLinks() error = %v", err)
	}

	if len(links) != 1 {
		t.Errorf("getLinks() returned %d links, expected 1", len(links))
	}

	err = cache.clearDeletedRecords("Links", []string{"Test Link ID"})
	if err != nil {
		t.Errorf("clearDeletedRecords() error = %v", err)
	}

	links, err = cache.getLinks(nil, nil)
	if err != nil {
		t.Errorf("getLinks() error = %v", err)
	}

	if len(links) != 1 {
		t.Errorf("getLinks() returned %d links, expected 1", len(links))
	}

	err = cache.clearDeletedRecords("Links", []string{"Test Link ID 2"})
	if err != nil {
		t.Errorf("clearDeletedRecords() error = %v", err)
	}

	links, err = cache.getLinks(nil, nil)
	if err != nil {
		t.Errorf("getLinks() error = %v", err)
	}

	if len(links) != 0 {
		t.Errorf("getLinks() returned %d links, expected 0", len(links))
	}
}

func TestSetData(t *testing.T) {
	cache := &Cache{File: ":memory:"}
	_ = cache.init()

	err := cache.setData("TestKey", "TestValue")
	if err != nil {
		t.Errorf("setData() error = %v", err)
	}

	value, err := cache.getData("TestKey")
	if err != nil {
		t.Errorf("getData() error = %v", err)
	}

	if *value != "TestValue" {
		t.Errorf("getData() returned %s, expected 'TestValue'", *value)
	}
}

func TestGetData(t *testing.T) {
	cache := &Cache{File: ":memory:"}
	_ = cache.init()

	err := cache.setData("TestKey", "TestValue")
	if err != nil {
		t.Errorf("setData() error = %v", err)
	}

	value, err := cache.getData("TestKey")
	if err != nil {
		t.Errorf("getData() error = %v", err)
	}

	if *value != "TestValue" {
		t.Errorf("getData() returned %s, expected 'TestValue'", *value)
	}
}

func TestClearCache(t *testing.T) {
	cache := &Cache{File: ":memory:"}
	_ = cache.init()

	now := time.Now()
	link := Link{
		Name:         stringPtr("Test Link"),
		Note:         stringPtr("Test Note"),
		URL:          stringPtr("http://example.com"),
		Category:     stringPtr("Test Category"),
		Tags:         []string{"Test Tag"},
		Created:      &now,
		LastModified: &now,
		RecordURL:    stringPtr("http://example.com/record"),
		ID:           stringPtr("Test Link ID"),
		Done:         false,
		ListIDs:      []string{"Test List ID"},
	}

	err := cache.saveLinks([]Link{link})
	if err != nil {
		t.Errorf("saveLinks() error = %v", err)
	}

	err = cache.clearCache()
	if err != nil {
		t.Errorf("clearCache() error = %v", err)
	}

	links, err := cache.getLinks(nil, nil)
	if err != nil {
		t.Errorf("getLinks() error = %v", err)
	}

	if len(links) != 0 {
		t.Errorf("getLinks() returned %d links, expected 0", len(links))
	}
}
