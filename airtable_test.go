package main

import (
	"testing"
	// "time"
)

func TestCacheLinks(t *testing.T) {
	airtable := &Airtable{
		Cache: &Cache{},
	}
	_, err := airtable.cacheLinks()
	if err != nil {
		t.Errorf("cacheLinks() error = %v", err)
	}
}

func TestClearDeletedLinks(t *testing.T) {
	airtable := &Airtable{
		Cache: &Cache{},
	}
	err := airtable.clearDeletedLinks()
	if err != nil {
		t.Errorf("clearDeletedLinks() error = %v", err)
	}
}

func TestCacheLists(t *testing.T) {
	airtable := &Airtable{
		Cache: &Cache{},
	}
	_, err := airtable.cacheLists()
	if err != nil {
		t.Errorf("cacheLists() error = %v", err)
	}
}

func TestClearDeletedLists(t *testing.T) {
	airtable := &Airtable{
		Cache: &Cache{},
	}
	err := airtable.clearDeletedLists()
	if err != nil {
		t.Errorf("clearDeletedLists() error = %v", err)
	}
}

func TestCreateLink(t *testing.T) {
	airtable := &Airtable{
		Cache: &Cache{},
	}
	link := &Link{
		Name:     "Test Link",
		Note:     "Test Note",
		URL:      "http://example.com",
		Category: "Test Category",
		Tags:     []string{"Test Tag"},
		Done:     false,
		ListIDs:  []string{"Test List ID"},
	}
	err := airtable.createLink(link)
	if err != nil {
		t.Errorf("createLink() error = %v", err)
	}
}

func TestCreateList(t *testing.T) {
	airtable := &Airtable{
		Cache: &Cache{},
	}
	list := &List{
		Name: "Test List",
		Note: "Test Note",
	}
	links := []Link{
		{
			Name:     "Test Link",
			Note:     "Test Note",
			URL:      "http://example.com",
			Category: "Test Category",
			Tags:     []string{"Test Tag"},
			Done:     false,
			ListIDs:  []string{"Test List ID"},
		},
	}
	err := airtable.createList(list, &links)
	if err != nil {
		t.Errorf("createList() error = %v", err)
	}
}

func TestUpdateLink(t *testing.T) {
	airtable := &Airtable{
		Cache: &Cache{},
	}
	link := &Link{
		ID:       "Test Link ID",
		Name:     "Updated Test Link",
		Note:     "Updated Test Note",
		URL:      "http://example.com",
		Category: "Updated Test Category",
		Tags:     []string{"Updated Test Tag"},
		Done:     false,
		ListIDs:  []string{"Updated Test List ID"},
	}
	err := airtable.updateLink(link)
	if err != nil {
		t.Errorf("updateLink() error = %v", err)
	}
}

func TestUpdateList(t *testing.T) {
	airtable := &Airtable{
		Cache: &Cache{},
	}
	list := &List{
		ID:       "Test List ID",
		Name:     "Updated Test List",
		Note:     "Updated Test Note",
		LinkIDs:  []string{"Updated Test Link ID"},
	}
	err := airtable.updateList(list)
	if err != nil {
		t.Errorf("updateList() error = %v", err)
	}
}

func TestDeleteLink(t *testing.T) {
	airtable := &Airtable{
		Cache: &Cache{},
	}
	link := &Link{
		ID: "Test Link ID",
	}
	err := airtable.deleteLink(link)
	if err != nil {
		t.Errorf("deleteLink() error = %v", err)
	}
}

func TestDeleteList(t *testing.T) {
	airtable := &Airtable{
		Cache: &Cache{},
	}
	list := &List{
		ID: "Test List ID",
	}
	err := airtable.deleteList(list, true)
	if err != nil {
		t.Errorf("deleteList() error = %v", err)
	}
}

func TestListToLinkCopier(t *testing.T) {
	airtable := &Airtable{
		Cache: &Cache{},
	}
	list := &List{
		ID:   "Test List ID",
		Name: "Test List",
	}
	err := airtable.listToLinkCopier(list)
	if err != nil {
		t.Errorf("listToLinkCopier() error = %v", err)
	}
}
