package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

// Interact with the Airtable database

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
	return a.createRecords("Links", records)
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
	err := a.createRecords("Lists", listRecords)
	if err != nil {
		return err
	}

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
	return a.createRecords("Links", linkRecords)
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
	return a.updateRecords("Links", records)
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
	return a.updateRecords("Lists", records)
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

func (a *Airtable) listToLinkCopier(c *Cache, list *List) error {
	name := list.Name
	links, err := c.getLinks(&list.ID)
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
