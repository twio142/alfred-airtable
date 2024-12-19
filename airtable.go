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
		"filterByFormula": fmt.Sprintf("IS_AFTER(LAST_MODIFIED_TIME(),'%s')", a.Cache.LastCachedAt.Format(time.RFC3339)),
		"fields":          []string{"Name", "Note", "URL", "Category", "Tags", "Last Modified", "Record URL", "Done", "Lists"},
	}
	records, err := a.fetchRecords("Links", params)
	if err != nil {
		return nil, err
	}
	links := []Link{}
	for _, record := range records {
		links = append(links, *record.toLink())
	}
	return links, nil
}

func (a *Airtable) fetchLists() ([]List, error) {
	params := map[string]interface{}{
		"filterByFormula": fmt.Sprintf("IS_AFTER(LAST_MODIFIED_TIME(),'%s')", a.Cache.LastCachedAt.Format(time.RFC3339)),
		"fields":          []string{"Name", "Note", "Last Modified", "Record URL", "Links"},
	}
	records, err := a.fetchRecords("Lists", params)
	if err != nil {
		return nil, err
	}
	lists := []List{}
	for _, record := range records {
		lists = append(lists, *record.toList())
	}
	return lists, nil
}

func (a *Airtable) fetchAllIDs(table string) ([]string, error) {
	IDs := []string{}
	params := map[string]interface{}{
		"fields": []string{"Name"},
	}
	records, err := a.fetchRecords(table, params)
	if err != nil {
		return IDs, err
	}
	for _, record := range records {
		IDs = append(IDs, *record.ID)
	}
	return IDs, nil
}

func (a *Airtable) cacheData() error {
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
		if err := a.Cache.clearDeletedRecords("Links", linkIDs); err != nil {
			return err
		}
		if err := a.Cache.clearDeletedRecords("Lists", listIDs); err != nil {
			return err
		}
		if err := a.Cache.saveLinks(links); err != nil {
			return err
		}
		if err := a.Cache.saveLists(lists); err != nil {
			return err
		}
		if tags := schema[0]; tags != nil {
			_ = a.Cache.setData("Tags", *tags)
		}
		if categories := schema[1]; categories != nil {
			_ = a.Cache.setData("Categories", *categories)
		}
		_ = a.Cache.setData("CachedAt", now.Format(time.RFC3339))
		a.Cache.LastCachedAt = now
	}

	return nil
}

func (a *Airtable) createLink(link *Link) error {
	record := link.toRecord()
	err := a.createRecords("Links", []*Record{&record})
	if err != nil {
		return err
	}
	link.ID = record.ID
	link.Created = record.CreatedTime
	return nil
}

func (a *Airtable) createList(list *List, links *[]Link) error {
	if list.ID == nil {
		lists, _ := a.Cache.getLists(list)
		if len(lists) > 0 {
			list.ID = lists[0].ID
		} else {
			listRecord := list.toRecord()
			err := a.createRecords("Lists", []*Record{&listRecord})
			if err != nil {
				return err
			}
			list.ID = listRecord.ID
		}
	}

	if links == nil || len(*links) == 0 {
		return nil
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
	err := a.createRecords("Links", linkRecords)
	if err != nil {
		return err
	}
	for _, linkRecord := range linkRecords {
		list.LinkIDs = append(list.LinkIDs, *linkRecord.ID)
	}
	return nil
}

func (a *Airtable) updateLink(link *Link) error {
	if link.ID == nil {
		return fmt.Errorf("Link ID is required")
	}
	record := link.toRecord()
	err := a.updateRecords("Links", []*Record{&record})
	if err != nil {
		return err
	}
	*link = *record.toLink()
	return nil
}

func (a *Airtable) updateList(list *List) error {
	if list.ID == nil {
		return fmt.Errorf("List ID is required")
	}
	record := list.toRecord()
	err := a.updateRecords("Lists", []*Record{&record})
	if err != nil {
		return err
	}
	*list = *record.toList()
	return nil
}

func (a *Airtable) deleteLink(link *Link) error {
	if link.ID == nil {
		return fmt.Errorf("Link ID is required")
	}
	return a.deleteRecords("Links", []*Record{{ID: link.ID}})
}

func (a *Airtable) deleteList(list *List, deleteLinks bool) error {
	if list.ID == nil {
		return fmt.Errorf("List ID is required")
	}
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
	err := a.deleteRecords("Lists", []*Record{{ID: list.ID}})
	if err != nil {
		return err
	}
	return nil
}

func (a *Airtable) listToLinkCopier(list *List) (*string, error) {
	name := "Untitled List"
	if list.Name != nil {
		name = *list.Name
	}
	links, err := a.Cache.getLinks(list, nil)
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
