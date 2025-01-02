package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

// Interact with the Airtable database

func (a *Airtable) fetchLinks() ([]Link, error) {
	params := map[string]interface{}{
		"filterByFormula": fmt.Sprintf("IS_AFTER(LAST_MODIFIED_TIME(),'%s')", a.cache.lastSyncedAt.Format(time.RFC3339)),
		"fields":          []string{"Name", "Note", "URL", "Category", "Tags", "Last Modified", "Record URL", "Done", "Lists"},
	}
	records, err := a.fetchRecords("Links", params)
	if err != nil {
		return nil, err
	}
	links := make([]Link, len(records))
	for i, record := range records {
		links[i] = *record.toLink()
	}
	return links, nil
}

func (a *Airtable) fetchLists() ([]List, error) {
	params := map[string]interface{}{
		"filterByFormula": fmt.Sprintf("IS_AFTER(LAST_MODIFIED_TIME(),'%s')", a.cache.lastSyncedAt.Format(time.RFC3339)),
		"fields":          []string{"Name", "Note", "Last Modified", "Record URL", "Links"},
	}
	records, err := a.fetchRecords("Lists", params)
	if err != nil {
		return nil, err
	}
	lists := make([]List, len(records))
	for i, record := range records {
		lists[i] = *record.toList()
	}
	logMessage("INFO", "Fetched %d lists", len(lists))
	return lists, nil
}

func (a *Airtable) fetchAllIDs(table string) ([]string, error) {
	params := map[string]interface{}{
		"fields": []string{"Name"},
	}
	records, err := a.fetchRecords(table, params)
	if err != nil {
		return []string{}, err
	}
	IDs := make([]string, len(records))
	for i, record := range records {
		IDs[i] = *record.ID
	}
	logMessage("INFO", "Fetched %d IDs in %s", len(IDs), table)
	return IDs, nil
}

func (a *Airtable) syncData(force ...bool) error {
	forceSync := len(force) > 0 && force[0]
	if !forceSync && time.Since(a.cache.lastSyncedAt) < a.cache.maxAge {
		return nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var wg sync.WaitGroup
	wg.Add(5)

	errorChan := make(chan error, 1)
	linksChan := make(chan []Link, 1)
	listsChan := make(chan []List, 1)
	linkIDsChan := make(chan []string, 1)
	listIDsChan := make(chan []string, 1)
	schemaChan := make(chan []*string, 1)

	go func() {
		defer wg.Done()
		links, err := a.fetchLinks()
		if err != nil {
			select {
			case errorChan <- err:
			default:
			}
			cancel()
			return
		}
		select {
		case linksChan <- links:
		case <-ctx.Done():
		}
	}()

	go func() {
		defer wg.Done()
		lists, err := a.fetchLists()
		if err != nil {
			select {
			case errorChan <- err:
			default:
			}
			cancel()
			return
		}
		select {
		case listsChan <- lists:
		case <-ctx.Done():
		}
	}()

	go func() {
		defer wg.Done()
		linkIDs, err := a.fetchAllIDs("Links")
		if err != nil {
			select {
			case errorChan <- err:
			default:
			}
			cancel()
			return
		}
		select {
		case linkIDsChan <- linkIDs:
		case <-ctx.Done():
		}
	}()

	go func() {
		defer wg.Done()
		listIDs, err := a.fetchAllIDs("Lists")
		if err != nil {
			select {
			case errorChan <- err:
			default:
			}
			cancel()
			return
		}
		select {
		case listIDsChan <- listIDs:
		case <-ctx.Done():
		}
	}()

	go func() {
		defer wg.Done()
		tags, categories, err := a.fetchSchema()
		if err != nil {
			select {
			case errorChan <- err:
			default:
			}
			cancel()
			return
		}
		schema := []*string{nil, nil}
		if tags != nil {
			schema[0] = stringPtr(strings.Join(*tags, ","))
		}
		if categories != nil {
			schema[1] = stringPtr(strings.Join(*categories, ","))
		}
		select {
		case schemaChan <- schema:
		case <-ctx.Done():
		}
	}()

	go func() {
		wg.Wait()
		close(errorChan)
		close(linksChan)
		close(listsChan)
		close(linkIDsChan)
		close(listIDsChan)
		close(schemaChan)
	}()

	select {
	case err := <-errorChan:
		return err
	case links := <-linksChan:
		lists := <-listsChan
		linkIDs := <-linkIDsChan
		listIDs := <-listIDsChan
		schema := <-schemaChan

		now := time.Now()
		if err := a.cache.clearDeletedRecords("Links", linkIDs); err != nil {
			return err
		}
		if err := a.cache.clearDeletedRecords("Lists", listIDs); err != nil {
			return err
		}
		if err := a.cache.saveLinks(links); err != nil {
			return err
		}
		if err := a.cache.saveLists(lists); err != nil {
			return err
		}
		if tags := schema[0]; tags != nil {
			_ = a.cache.setData("Tags", *tags)
		}
		if categories := schema[1]; categories != nil {
			_ = a.cache.setData("Categories", *categories)
		}
		_ = a.cache.setData("LastSyncedAt", now.Format(time.RFC3339))
		a.cache.lastSyncedAt = now
	}

	return nil
}

func (a *Airtable) createLink(link *Link) error {
	if link == nil {
		return fmt.Errorf("Link is required")
	}
	record := link.toRecord()
	records := []*Record{&record}
	err := a.createRecords("Links", &records)
	if err != nil {
		return err
	}
	link.ID = records[0].ID
	link.Created = records[0].CreatedTime
	logMessage("INFO", "Created link %s", *link.Name)
	return nil
}

func (a *Airtable) createList(list *List, links *[]Link) error {
	if list == nil {
		return fmt.Errorf("List is required")
	}
	if list.ID == nil {
		lists, _ := a.cache.getLists(list)
		if len(lists) > 0 {
			list.ID = lists[0].ID
		} else {
			listRecord := list.toRecord()
			listRecords := []*Record{&listRecord}
			err := a.createRecords("Lists", &listRecords)
			if err != nil {
				return err
			}
			list.ID = listRecords[0].ID
		}
	}

	if links == nil || len(*links) == 0 {
		return nil
	}
	if list.LinkIDs == nil {
		list.LinkIDs = make([]string, len(*links))
	}

	linkRecords := make([]*Record, len(*links))
	for i, link := range *links {
		if link.ListIDs == nil {
			link.ListIDs = []string{*list.ID}
		} else {
			link.ListIDs = append(link.ListIDs, *list.ID)
		}
		record := link.toRecord()
		linkRecords[i] = &record
	}
	err := a.createRecords("Links", &linkRecords)
	if err != nil {
		return err
	}
	for i, linkRecord := range linkRecords {
		list.LinkIDs[i] = *linkRecord.ID
	}
	logMessage("INFO", "Created list %s", *list.Name)
	return nil
}

func (a *Airtable) updateLink(link *Link) error {
	if link == nil || link.ID == nil {
		return fmt.Errorf("Link with an ID is required")
	}
	record := link.toRecord()
	records := []*Record{&record}
	err := a.updateRecords("Links", &records)
	if err != nil {
		return err
	}
	*link = *records[0].toLink()
	logMessage("INFO", "Updated link %s", *link.Name)
	return nil
}

func (a *Airtable) updateList(list *List) error {
	if list == nil || list.ID == nil {
		return fmt.Errorf("List with an ID is required")
	}
	record := list.toRecord()
	records := []*Record{&record}
	err := a.updateRecords("Lists", &records)
	if err != nil {
		return err
	}
	*list = *records[0].toList()
	logMessage("INFO", "Updated list %s", *list.Name)
	return nil
}

func (a *Airtable) deleteLink(link *Link) error {
	if link == nil || link.ID == nil {
		return fmt.Errorf("Link with an ID is required")
	}
	logMessage("INFO", "Deleting link %s", *link.ID)
	return a.deleteRecords("Links", &[]*Record{{ID: link.ID}})
}

func (a *Airtable) deleteList(list *List, deleteLinks bool) error {
	if list == nil || list.ID == nil {
		return fmt.Errorf("List with an ID is required")
	}
	if deleteLinks && len(list.LinkIDs) > 0 {
		records := make([]*Record, len(list.LinkIDs))
		for i, linkID := range list.LinkIDs {
			record := Record{
				ID: &linkID,
			}
			records[i] = &record
		}
		err := a.deleteRecords("Links", &records)
		if err != nil {
			return err
		}
	}
	err := a.deleteRecords("Lists", &[]*Record{{ID: list.ID}})
	if err != nil {
		return err
	}
	logMessage("INFO", "Deleted list %s", *list.ID)
	return nil
}

func (a *Airtable) listToLinkCopier(list *List) (*string, error) {
	name := "Untitled List"
	if list.Name != nil {
		name = *list.Name
	}
	links, err := a.cache.getLinks(list, nil)
	if err != nil {
		return nil, err
	}
	if len(links) == 0 {
		return nil, fmt.Errorf("no links found in list")
	}
	lines := []string{}
	for _, link := range links {
		line := fmt.Sprintf("- [%s](%s)", *link.Name, *link.URL)
		lines = append(lines, line)
	}
	text := strings.Join(lines, "\n")

	lcDir := "link_copiers"
	if _, err := os.Stat(lcDir); os.IsNotExist(err) {
		_ = os.Mkdir(lcDir, 0755)
	}
	outputFile := fmt.Sprintf("%s/%s.md", lcDir, name)
	suffix := 1
	for {
		if _, err := os.Stat(outputFile); os.IsNotExist(err) {
			break
		}
		outputFile = fmt.Sprintf("%s/%s_%d.md", lcDir, name, suffix)
		suffix++
	}
	logMessage("INFO", "Saving list %s to %s", *list.Name, outputFile)
	return &outputFile, os.WriteFile(outputFile, []byte(text), 0644)
}

func (a *Airtable) linkCopierToList(file string) (*List, error) {
	name := filepath.Base(file)
	name = strings.TrimSuffix(name, ".md")
	text, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(text), "\n")
	links := []Link{}
	for _, line := range lines {
		link := Link{}
		re := regexp.MustCompile(`^- \[(.+)\]\((.+?)\)$`)
		matches := re.FindStringSubmatch(line)
		if len(matches) == 3 {
			link.Name = &matches[1]
			link.URL = &matches[2]
			links = append(links, link)
		}
	}
	list := List{Name: &name}
	err = a.createList(&list, &links)
	if err != nil {
		return nil, err
	}
	return &list, nil
}
