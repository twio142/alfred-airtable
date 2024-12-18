package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
)

// Interact with the Airtable database

func (a *Airtable) cacheLinks() ([]Link, error) {
	cachedAtStr, _ := a.Cache.getData("CachedAt")
	var cachedAt time.Time
	if cachedAtStr != nil {
		cachedAt, _ = time.Parse(time.RFC3339, *cachedAtStr)
	} else {
		cachedAt = time.Time{}
	}
	params := map[string]interface{}{
		"filterByFormula": fmt.Sprintf("IS_AFTER(LAST_MODIFIED_TIME(),'%s')", cachedAt.Format(time.RFC3339)),
		"fields":          []string{"Name", "Note", "URL", "Category", "Tags", "Last Modified", "Record URL", "Done", "Lists"},
	}
	records, err := a.fetchRecords("Links", params)
	if err != nil {
		return nil, err
	}
	links := []Link{}
	for _, record := range records {
		link := Link{
			Name:         getStringField(*record.Fields, "Name"),
			Note:         getStringField(*record.Fields, "Note"),
			URL:          getStringField(*record.Fields, "URL"),
			Category:     getStringField(*record.Fields, "Category"),
			Tags:         getStringSliceField(*record.Fields, "Tags"),
			Created:      *record.CreatedTime,
			LastModified: getTimeField(*record.Fields, "Last Modified"),
			RecordURL:    getStringField(*record.Fields, "Record URL"),
			ID:           *record.ID,
			Done:         getBoolField(*record.Fields, "Done"),
			ListIDs:      getStringSliceField(*record.Fields, "Lists"),
		}
		links = append(links, link)
	}
	a.Cache.saveLinks(links)
	return links, nil
}

func (a *Airtable) clearDeletedLinks() error {
	params := map[string]interface{}{
		"fields": []string{"Name"},
	}
	records, err := a.fetchRecords("Links", params)
	if err != nil {
		return err
	}
	linkIDs := []string{}
	for _, record := range records {
		linkIDs = append(linkIDs, *record.ID)
	}
	return a.Cache.clearDeletedRecords("Links", linkIDs)
}

func (a *Airtable) cacheLists() ([]List, error) {
	cachedAtStr, _ := a.Cache.getData("CachedAt")
	var cachedAt time.Time
	if cachedAtStr != nil {
		cachedAt, _ = time.Parse(time.RFC3339, *cachedAtStr)
	} else {
		cachedAt = time.Time{}
	}
	params := map[string]interface{}{
		"filterByFormula": fmt.Sprintf("IS_AFTER(LAST_MODIFIED_TIME(),'%s')", cachedAt.Format(time.RFC3339)),
		"fields":          []string{"Name", "Note", "Last Modified", "Record URL", "Links"},
	}
	records, err := a.fetchRecords("Lists", params)
	if err != nil {
		return nil, err
	}
	lists := []List{}
	for _, record := range records {
		list := List{
			Name:         (*record.Fields)["Name"].(string),
			Note:         (*record.Fields)["Note"].(string),
			Created:      (*record.CreatedTime),
			LastModified: (*record.Fields)["Last Modified"].(time.Time),
			RecordURL:    (*record.Fields)["Record URL"].(string),
			ID:           (*record.ID),
			LinkIDs:      (*record.Fields)["Links"].([]string),
		}
		lists = append(lists, list)
	}
	a.Cache.saveLists(lists)
	return lists, nil
}

func (a *Airtable) clearDeletedLists() error {
	params := map[string]interface{}{
		"fields": []string{"Name"},
	}
	records, err := a.fetchRecords("Lists", params)
	if err != nil {
		return err
	}
	listIDs := []string{}
	for _, record := range records {
		listIDs = append(listIDs, *record.ID)
	}
	return a.Cache.clearDeletedRecords("Lists", listIDs)
}

func (a *Airtable) createLink(link *Link) error {
	record := Record{
		Fields: &map[string]interface{}{
			"Name":     link.Name,
			"Note":     link.Note,
			"Url":      link.URL,
			"Category": link.Category,
			"Tags":     link.Tags,
			"Done":     link.Done,
			"Lists":    link.ListIDs,
		},
	}
	records := []*Record{&record}
	_, err := a.createRecords("Links", records)
	return err
}

func (a *Airtable) createList(list *List, links *[]Link) error {
	listRecord := Record{
		Fields: &map[string]interface{}{
			"Name": list.Name,
			"Note": list.Note,
		},
	}
	// TODO: Check if list already exists
	listRecords := []*Record{&listRecord}
	listRecords, err := a.createRecords("Lists", listRecords)
	if err != nil {
		return err
	}
	list.ID = *listRecords[0].ID

	linkRecords := []*Record{}
	for _, link := range *links {
		linkRecord := &Record{
			Fields: &map[string]interface{}{
				"Name":     link.Name,
				"Note":     link.Note,
				"Url":      link.URL,
				"Category": link.Category,
				"Tags":     link.Tags,
				"Done":     link.Done,
				"Lists":    []string{list.ID},
			},
		}
		linkRecords = append(linkRecords, linkRecord)
	}
	_, err = a.createRecords("Links", linkRecords)
	return err
}

func (a *Airtable) updateLink(link *Link) error {
	record := Record{
		ID: &link.ID,
		Fields: &map[string]interface{}{
			"Name":     link.Name,
			"Note":     link.Note,
			"Url":      link.URL,
			"Category": link.Category,
			"Tags":     link.Tags,
			"Done":     link.Done,
			"Lists":    link.ListIDs,
		},
	}
	records := []*Record{&record}
	_, err := a.updateRecords("Links", records)
	return err
}

func (a *Airtable) updateList(list *List) error {
	record := Record{
		ID: &list.ID,
		Fields: &map[string]interface{}{
			"Name":  list.Name,
			"Note":  list.Note,
			"Links": list.LinkIDs,
		},
	}
	records := []*Record{&record}
	_, err := a.updateRecords("Lists", records)
	return err
}

func (a *Airtable) deleteLink(link *Link) error {
	record := Record{
		ID: &link.ID,
	}
	records := []*Record{&record}
	return a.deleteRecords("Links", records)
}

func (a *Airtable) deleteList(list *List, deleteLinks bool) error {
	if deleteLinks && len(list.LinkIDs) > 0 {
		records := []*Record{}
		for _, linkID := range list.LinkIDs {
			record := Record{
				ID: &linkID,
			}
			records = append(records, &record)
		}
		err := a.deleteRecords("Links", records)
		if err != nil {
			return err
		}
	}
	record := Record{
		ID: &list.ID,
	}
	records := []*Record{&record}
	err := a.deleteRecords("Lists", records)
	if err != nil {
		return err
	}
	return nil
}

func (a *Airtable) listToLinkCopier(list *List) error {
	name := list.Name
	links, err := a.Cache.getLinks(&list.ID)
	if err != nil {
		return err
	}
	lines := []string{}
	for _, link := range links {
		line := fmt.Sprintf("- [%s](%s)", link.Name, link.URL)
		lines = append(lines, line)
	}
	text := strings.Join(lines, "\n")
	outputFile := fmt.Sprintf("%s.md", name) // TODO: save to links directory; prevent overwriting
	os.WriteFile(outputFile, []byte(text), 0644)
	return nil
}

func (a *Airtable) linkCopierToList(file string) (List, error) {
	name := strings.TrimSuffix(file, ".md")
	text, err := os.ReadFile(file)
	if err != nil {
		return List{}, err
	}
	lines := strings.Split(string(text), "\n")
	links := []Link{}
	for _, line := range lines {
		link := Link{}
		re := regexp.MustCompile(`^- \[(.+)\]\((.+?)\)$`)
		matches := re.FindStringSubmatch(line)
		if len(matches) == 3 {
			link.Name = matches[1]
			link.URL = matches[2]
			links = append(links, link)
		}
	}
	list := List{Name: name}
	err = a.createList(&list, &links)
	if err != nil {
		return List{}, err
	}
	return list, nil
}
