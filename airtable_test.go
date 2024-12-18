package main

import (
	"log"
	"os"
	"sync"
	"testing"
	"time"
)

func TestCacheLinks(t *testing.T) {
	airtable := &Airtable{
		BaseURL: "https://api.airtable.com/v0",
		BaseID:  os.Getenv("BASE_ID"),
		DBPath:  "cache_test.db",
	}
	err := airtable.init()
	if err != nil {
		t.Errorf("init() error = %v", err)
	}
	_, err = airtable.fetchLinks()
	if err != nil {
		t.Errorf("cacheLinks() error = %v", err)
	}
	links, err := airtable.Cache.getLinks(nil, nil)
	if err != nil {
		t.Errorf("getLinks() error = %v", err)
	}
	log.Println("cached", len(links), "links")
	airtable.Cache.DB.Close()
}

func TestCacheLists(t *testing.T) {
	airtable := &Airtable{
		BaseURL: "https://api.airtable.com/v0",
		BaseID:  os.Getenv("BASE_ID"),
		DBPath:  "cache_test.db",
	}
	err := airtable.init()
	if err != nil {
		t.Errorf("init() error = %v", err)
	}
	_ = airtable.Cache.setData("CachedAt", "2000-01-01T00:00:00Z")
	_, err = airtable.fetchLists()
	if err != nil {
		t.Errorf("cacheLists() error = %v", err)
	}
	lists, err := airtable.Cache.getLists(nil)
	if err != nil {
		t.Errorf("getLists() error = %v", err)
	}
	log.Println("cached", len(lists), "lists")
	airtable.Cache.DB.Close()
}

func TestCreateLink(t *testing.T) {
	airtable := &Airtable{
		BaseURL: "https://api.airtable.com/v0",
		BaseID:  os.Getenv("BASE_ID"),
		DBPath:  "cache_test.db",
	}
	err := airtable.init()
	if err != nil {
		t.Errorf("init() error = %v", err)
	}

	link := &Link{
		Name: stringPtr("Test Link"),
		Note: stringPtr("Test Note"),
		URL:  stringPtr("http://example.com"),
	}
	link, err = airtable.createLink(link)
	if err != nil {
		t.Errorf("createLink() error = %v", err)
	}
	log.Println("created link", link.ID)
	airtable.Cache.DB.Close()
}

func TestCreateList(t *testing.T) {
	airtable := &Airtable{
		BaseURL: "https://api.airtable.com/v0",
		BaseID:  os.Getenv("BASE_ID"),
		DBPath:  "cache_test.db",
	}
	err := airtable.init()
	if err != nil {
		t.Errorf("init() error = %v", err)
	}

	list := &List{
		Name: stringPtr("Test List"),
		Note: stringPtr("Test Note"),
	}
	links := []Link{
		{
			Name: stringPtr("Test Link"),
			Note: stringPtr("Test Note"),
			URL:  stringPtr("http://example.com"),
		},
	}
	list, err = airtable.createList(list, &links)
	if err != nil {
		t.Errorf("createList() error = %v", err)
	}
	log.Println("created list", list.ID)
	if len(list.LinkIDs) != 1 {
		t.Errorf("createList() error = %v", "link not added")
	}
	airtable.Cache.DB.Close()
}

func TestUpdateLink(t *testing.T) {
	airtable := &Airtable{
		BaseURL: "https://api.airtable.com/v0",
		BaseID:  os.Getenv("BASE_ID"),
		DBPath:  "cache_test.db",
	}
	err := airtable.init()
	if err != nil {
		t.Errorf("init() error = %v", err)
	}

	links, err := airtable.fetchLinks()
	if err != nil {
		t.Errorf("cacheLinks() error = %v", err)
	}

	var link *Link
	for _, l := range links {
		if *l.Name == "Test Link" {
			link = &l
			break
		}
	}

	link.Name = stringPtr("Updated Link")

	link, err = airtable.updateLink(link)
	if err != nil {
		t.Errorf("updateLink() error = %v", err)
	}

	log.Println("updated link", link.ID, *link.Name)
	airtable.Cache.DB.Close()
}

func TestDeleteLink(t *testing.T) {
	airtable := &Airtable{
		BaseURL: "https://api.airtable.com/v0",
		BaseID:  os.Getenv("BASE_ID"),
		DBPath:  "cache_test.db",
	}
	err := airtable.init()
	if err != nil {
		t.Errorf("init() error = %v", err)
	}

	links, err := airtable.Cache.getLinks(nil, nil)
	if err != nil {
		t.Errorf("getLinks() error = %v", err)
	}
	log.Println("got", len(links), "links")

	var link *Link
	for _, l := range links {
		if *l.Name == "Updated Link" {
			link = &l
			break
		}
	}

	err = airtable.deleteLink(link)
	if err != nil {
		t.Errorf("deleteLink() error = %v", err)
	}

	_, _ = airtable.fetchLinks()
	_ = airtable.Cache.setData("CachedAt", time.Now().Format(time.RFC3339))
	airtable.Cache.DB.Close()
}

func TestDeleteList(t *testing.T) {
	airtable := &Airtable{
		BaseURL: "https://api.airtable.com/v0",
		BaseID:  os.Getenv("BASE_ID"),
		DBPath:  "cache_test.db",
	}
	err := airtable.init()
	if err != nil {
		t.Errorf("init() error = %v", err)
	}
	airtable.Cache.LastCachedAt = time.Time{}

	lists, _ := airtable.fetchLists()
	var list *List
	for _, l := range lists {
		if *l.Name == "Test List" {
			list = &l
			break
		}
	}
	if list == nil {
		t.Errorf("list not found")
		return
	}

	log.Println("deleting list", *list.Name, list.ID)

	err = airtable.deleteList(list, true)
	if err != nil {
		t.Errorf("deleteList() error = %v", err)
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		_, _ = airtable.fetchLinks()
	}()

	go func() {
		defer wg.Done()
		_, _ = airtable.fetchLists()
	}()

	wg.Wait()
	_ = airtable.Cache.setData("CachedAt", time.Now().Format(time.RFC3339))
	airtable.Cache.DB.Close()
}

func TestCacheData(t *testing.T) {
	airtable := &Airtable{
		BaseURL: "https://api.airtable.com/v0",
		BaseID:  os.Getenv("BASE_ID"),
		DBPath:  "cache_test.db",
	}
	err := airtable.init()
	if err != nil {
		t.Errorf("init() error = %v", err)
	}
	log.Println(airtable.Cache.LastCachedAt)
	err = airtable.cacheData()
	if err != nil {
		t.Errorf("cacheData() error = %v", err)
	}
	log.Println(airtable.Cache.LastCachedAt)
	airtable.Cache.DB.Close()
}

func TestListToLinkCopier(t *testing.T) {
	airtable := &Airtable{
		BaseURL: "https://api.airtable.com/v0",
		BaseID:  os.Getenv("BASE_ID"),
		DBPath:  "cache_test.db",
	}
	err := airtable.init()
	if err != nil {
		t.Errorf("init() error = %v", err)
	}
	lists, err := airtable.Cache.getLists(nil)
	if err != nil {
		t.Errorf("getLists() error = %v", err)
	}
	if len(lists) == 0 {
		t.Errorf("getLists() error = %v", "no lists")
	}
	log.Println(*lists[0].Name)

	lc, err := airtable.listToLinkCopier(&lists[0])
	if err != nil {
		t.Errorf("listToLinkCopier() error = %v", err)
	}
	log.Println(*lc)
}
