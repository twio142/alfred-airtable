package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"
)

// Handle user interactions through Alfred

func (l *Link) format() Item {
	subtitle := ""
	subParts := []string{}
	largeParts := []string{
		*l.Name,
		"􀉣 " + *l.URL,
	}
	icon := Icon{Path: stringPtr("media/link.png")}
	if l.Done {
		subtitle = "􀃲 "
		icon.Path = stringPtr("media/link-done.png")
	}
	if len(l.Tags) > 0 {
		tags := []string{}
		for _, tag := range l.Tags {
			tags = append(tags, "􀆃"+tag)
		}
		subParts = append(subParts, strings.Join(tags, " "))
		largeParts = append(largeParts, strings.Join(tags, " "))
	}
	if len(l.ListNames) > 0 {
		lists := "􀈕 " + strings.Join(l.ListNames, ", ")
		subParts = append(subParts, lists)
		largeParts = append(largeParts, lists)
	}
	if l.Category != nil {
		subParts = append(subParts, "􀈭 "+*l.Category)
		largeParts = append(largeParts, "􀈭 "+*l.Category)
	}
	if l.Note != nil {
		subParts = append(subParts, "􀓕 "+*l.Note)
		largeParts = append(largeParts, "􀓕 "+*l.Note)
	}

	arg := fmt.Sprintf("[%s](%s)", *l.Name, *l.URL)

	item := Item{
		Title:        *l.Name,
		Subtitle:     subtitle + strings.Join(subParts, "  ·  "),
		Arg:          arg,
		Type:         stringPtr("file:skipcheck"),
		Match:        l.match(),
		QuickLookURL: l.URL,
		Action: struct {
			Text *string `json:"text,omitempty"`
			File *string `json:"file,omitempty"`
			URL  *string `json:"url,omitempty"`
		}{
			Text: &arg,
		},
		Text: struct {
			Copy      *string `json:"copy,omitempty"`
			LargeType *string `json:"largetype,omitempty"`
		}{
			Copy:      l.URL,
			LargeType: stringPtr(strings.Join(largeParts, "\n")),
		},
		Icon: &icon,
		Variables: map[string]string{
			"URL":  *l.URL,
			"ID":   *l.ID,
			"rURL": *l.RecordURL,
		},
		Mods: &map[string]Mod{
			"alt": {
				Subtitle: "Edit record",
				Icon:     &Icon{Path: stringPtr("media/edit.png")},
				Variables: map[string]string{
					"ID":   *l.ID,
					"mode": "edit-link",
				},
			},
			"shift": {
				Subtitle:  "Send to link copier",
				Arg:       arg,
				Variables: map[string]string{"mod": "save"},
			},
			"alt+shift": {
				Subtitle:  "Open record",
				Variables: map[string]string{"URL": *l.RecordURL},
			},
			"ctrl": {
				Subtitle: "Delete link",
				Icon:     &Icon{Path: stringPtr("media/delete.png")},
				Variables: map[string]string{
					"ID":   *l.ID,
					"mode": "delete-link",
				},
			},
			"fn": {
				Subtitle: "Rebuild cache",
				Icon:     &Icon{Path: stringPtr("media/reload.png")},
				Variables: map[string]string{
					"mode": "sync",
				},
			},
		},
	}

	if !l.Done {
		(*item.Mods)["cmd"] = Mod{
			Subtitle: "Mark as done 􀃲 ",
			Icon:     &Icon{Path: stringPtr("media/checked.png")},
			Variables: map[string]string{
				"ID":   *l.ID,
				"mode": "complete-link",
			},
		}
	}
	return item
}

func (l *List) format() Item {
	subtitle := fmt.Sprintf("􀉣 %d/%d", *l.LinksDone, len(l.LinkIDs))
	largetype := ""
	if l.Note != nil {
		subtitle = subtitle + "  ·  􀓕 " + *l.Note
		largetype = *l.Note + "\n\n"
	}
	for _, linkName := range l.LinkNames {
		largetype = largetype + "- " + linkName + "\n"
	}
	item := Item{
		Title:    *l.Name,
		Subtitle: subtitle,
		Match:    l.match(),
		Text: struct {
			Copy      *string `json:"copy,omitempty"`
			LargeType *string `json:"largetype,omitempty"`
		}{
			Copy:      l.RecordURL,
			LargeType: &largetype,
		},
		Icon: &Icon{Path: stringPtr("media/list.png")},
		Variables: map[string]string{
			"listID": *l.ID,
			"rURL":   *l.RecordURL,
		},
		Mods: &map[string]Mod{
			"cmd": {
				Subtitle: "Add link to list",
				Icon:     &Icon{Path: stringPtr("media/add.png")},
				Variables: map[string]string{
					"mode":    "edit-link",
					"listIDs": *l.ID,
				},
			},
			"shift": {
				Subtitle: "Send to link copier",
				Icon:     &Icon{Path: stringPtr("media/clip.png")},
				Variables: map[string]string{
					"mode":   "list2lc",
					"listID": *l.ID,
				},
			},
			"ctrl": {
				Subtitle: "Delete list",
				Icon:     &Icon{Path: stringPtr("media/delete.png")},
				Variables: map[string]string{
					"mode":   "delete-list-links",
					"listID": *l.ID,
				},
			},
			"ctrl+alt": {
				Subtitle: "Delete list but keep links",
				Icon:     &Icon{Path: stringPtr("media/delete.png")},
				Variables: map[string]string{
					"mode":   "delete-list",
					"listID": *l.ID,
				},
			},
			"alt+shift": {
				Subtitle: "Open record",
				Arg:      *l.RecordURL,
				Variables: map[string]string{
					"url": *l.RecordURL,
				},
			},
			"fn": {
				Subtitle: "Rebuild cache",
				Icon:     &Icon{Path: stringPtr("media/reload.png")},
				Variables: map[string]string{
					"mode": "sync",
				},
			},
		},
	}

	return item
}

// list all links or links in a list
func (a *Airtable) listLinks(list *List) {
	wf := Workflow{}
	links, err := a.cache.getLinks(list, nil)
	if err != nil {
		wf.warnEmpty("Error: " + err.Error())
	} else {
		if len(links) == 0 {
			wf.warnEmpty("No Links Found")
		} else {
			for _, link := range links {
				wf.addItem(link.format())
			}
		}
		if list != nil {
			wf.addItem(Item{
				Title: "Go Back",
				Arg:   "__BACK__",
				Icon:  &Icon{Path: stringPtr("media/back.png")},
			})
		}
	}
	wf.output()
}

// list all lists
func (a *Airtable) listLists() {
	wf := Workflow{}
	lists, err := a.cache.getLists(nil)
	if err != nil {
		wf.warnEmpty("Error: " + err.Error())
	} else {
		if len(lists) == 0 {
			wf.warnEmpty("No Lists Found")
		} else {
			for _, list := range lists {
				wf.addItem(list.format())
			}
		}
	}
	wf.output()
}

func (a *Airtable) editLink(input string) {
	wf := Workflow{}
	variables := map[string]string{
		"ID":       os.Getenv("ID"),
		"title":    os.Getenv("title"),
		"URL":      os.Getenv("URL"),
		"note":     os.Getenv("note"),
		"category": os.Getenv("category"),
		"tags":     os.Getenv("tags"),
		"listIDs":  os.Getenv("listIDs"),
		"done":     os.Getenv("done"),
	}

	re := regexp.MustCompile(`^(- )?\[(.+)\]\((.+?)\)$`)
	if variables["URL"] == "" {
		if matches := re.FindStringSubmatch(os.Getenv("input")); matches != nil {
			variables["title"] = matches[2]
			variables["URL"] = matches[3]
		}
	}

	link := Link{}
	if variables["ID"] != "" {
		if links, _ := a.cache.getLinks(nil, stringPtr(variables["ID"])); len(links) > 0 {
			link = links[0]
		}
	}

	if link.ID == nil && variables["URL"] == "" {
		if matches := re.FindStringSubmatch(input); matches != nil {
			if testURL(matches[3]) {
				item := Item{
					Title:        "Save the Link to Airtable",
					Variables:    variables,
					QuickLookURL: &matches[3],
					Icon:         &Icon{Path: stringPtr("media/save.png")},
				}
				item.setVar("title", matches[2])
				item.setVar("URL", matches[3])
				item.setVar("mode", "save-link")
				altMod := Mod{
					Subtitle:  "Edit record",
					Icon:      &Icon{Path: stringPtr("media/edit.png")},
					Variables: variables,
				}
				altMod.setVar("mode", "edit-link")
				item.Mods = &map[string]Mod{"alt": altMod}
				wf.addItem(item)
				wf.addItem(Item{
					Title: matches[3],
					Icon:  &Icon{Path: stringPtr("media/link.png")},
					Valid: boolPtr(false),
				})
				wf.addItem(Item{
					Title: matches[2],
					Icon:  &Icon{Path: stringPtr("media/title.png")},
					Valid: boolPtr(false),
				})
				wf.output()
				return
			}
		}
		wf.addItem(Item{
			Title:    "Save a Link to Airtable",
			Subtitle: input,
			Valid:    boolPtr(false),
			Icon:     &Icon{Path: stringPtr("media/save.png")},
		})
		wf.output()
		return
	}

	// Save changes
	saveItem := Item{
		Title:     "Save the Link to Airtable",
		Icon:      &Icon{Path: stringPtr("media/save.png")},
		Variables: variables,
	}
	saveItem.setVar("mode", "save-link")
	wf.addItem(saveItem)

	// URL
	currentURL := variables["URL"]
	if currentURL == "" && link.URL != nil {
		currentURL = *link.URL
	}
	if testURL(input) {
		// Edit URL
		item := Item{
			Title:        "Edit URL: " + input,
			Subtitle:     "Current: " + currentURL,
			AutoComplete: &currentURL,
			QuickLookURL: &input,
			Valid:        boolPtr(input != currentURL),
			Icon:         &Icon{Path: stringPtr("media/link.png")},
			Variables:    variables,
		}
		item.setVar("URL", input)
		wf.addItem(item, true)
	} else {
		// Show current URL
		wf.addItem(Item{
			Title:        currentURL,
			Subtitle:     "Edit URL",
			AutoComplete: &currentURL,
			QuickLookURL: &currentURL,
			Icon:         &Icon{Path: stringPtr("media/link.png")},
			Valid:        boolPtr(false),
		})
	}

	// Title
	currentTitle := variables["title"]
	if currentTitle == "" {
		if link.Name != nil {
			currentTitle = *link.Name
		} else {
			variables["title"] = variables["URL"]
			currentTitle = variables["URL"]
		}
	}
	if input != "" {
		// Edit title
		item := Item{
			Title:        fmt.Sprintf("Edit Title: '%s'", input),
			Subtitle:     fmt.Sprintf("Current: '%s'", currentTitle),
			AutoComplete: &currentTitle,
			Valid:        boolPtr(input != currentTitle),
			Icon:         &Icon{Path: stringPtr("media/title.png")},
			Variables:    variables,
		}
		item.setVar("title", input)
		wf.addItem(item, true)
	} else {
		// Show current title
		wf.addItem(Item{
			Title:        currentTitle,
			Subtitle:     "Edit Title",
			AutoComplete: &currentTitle,
			Icon:         &Icon{Path: stringPtr("media/title.png")},
			Valid:        boolPtr(false),
		})
	}

	// Note
	currentNote := variables["note"]
	if currentNote == "__NONE__" {
		currentNote = ""
	} else if variables["note"] == "" && link.Note != nil {
		currentNote = *link.Note
	}
	if input != "" {
		// Edit note
		item := Item{
			Title:        "Edit Note: " + input,
			Subtitle:     "Current: " + currentNote,
			AutoComplete: &currentNote,
			Valid:        boolPtr(input != currentNote),
			Icon:         &Icon{Path: stringPtr("media/note.png")},
			Variables:    variables,
		}
		item.setVar("note", input)
		wf.addItem(item, true)
	} else if currentNote != "" {
		// Show current note
		wf.addItem(Item{
			Title:        currentNote,
			Subtitle:     "Edit Note",
			AutoComplete: &currentNote,
			Icon:         &Icon{Path: stringPtr("media/note.png")},
			Valid:        boolPtr(false),
		})
	}

	// Tags
	currentTags := strings.Split(variables["tags"], ",")
	if variables["tags"] == "__NONE__" {
		currentTags = []string{}
	} else if variables["tags"] == "" && link.Tags != nil {
		currentTags = link.Tags
	}
	tagRe := regexp.MustCompile(`^#(\w*)$`)
	tagsRe := regexp.MustCompile(`^(#\w+, *)*#\w+$`)

	if matches := tagRe.FindStringSubmatch(input); matches != nil {
		match := matches[1]
		if match != "" {
			// Create a new tag
			item := Item{
				Title:     "Create Tag: " + match,
				Variables: variables,
				Icon:      &Icon{Path: stringPtr("media/tag-new.png")},
			}
			item.setVar("tags", strings.Join(append(currentTags, match), ","))
			wf.addItem(item, true)
		}

		// Add an existing tag
		match = strings.ToLower(match)
		tagsMap := make(map[string]bool)
		if tags, _ := a.cache.getData("tags"); tags != nil {
			for _, tag := range strings.Split(*tags, ",") {
				tagsMap[tag] = true
			}
			for _, tag := range currentTags {
				tagsMap[tag] = false
			}
		}
		for tag := range tagsMap {
			if match == "" || strings.HasPrefix(strings.ToLower(tag), match) {
				item := Item{
					Title:        "Add Tag: " + tag,
					AutoComplete: stringPtr("#" + tag),
					Variables:    variables,
					Icon:         &Icon{Path: stringPtr("media/tag.png")},
				}
				item.setVar("tags", strings.Join(append(currentTags, tag), ","))
				wf.addItem(item, true)
			}
		}

	} else if tagsRe.FindStringSubmatch(input) != nil {
		// Edit all tags
		tagsMap := map[string]bool{}
		for _, part := range strings.Split(input, ",") {
			tag := strings.TrimSpace(part)
			tag = strings.TrimPrefix(tag, "#")
			tagsMap[tag] = true
		}
		tags := []string{}
		parts := []string{}
		for tag := range tagsMap {
			tags = append(tags, tag)
			parts = append(parts, "#"+tag)
		}
		newTagEdit := strings.Join(parts, ", ")
		parts = []string{}
		for _, tag := range currentTags {
			parts = append(parts, "#"+tag)
		}
		currentTagEdit := strings.Join(parts, ", ")
		item := Item{
			Title:        "Edit Tags: " + newTagEdit,
			Subtitle:     fmt.Sprintf("Current: %s", currentTagEdit),
			AutoComplete: stringPtr(currentTagEdit),
			Variables:    variables,
		}
		item.setVar("tags", strings.Join(tags, ","))
		wf.addItem(item, true)

	} else if len(currentTags) > 0 {
		// Show current tags
		parts := []string{}
		for _, tag := range currentTags {
			parts = append(parts, "#"+tag)
		}
		item := Item{
			Title:        strings.Join(parts, ", "),
			Subtitle:     "Edit Tags",
			Icon:         &Icon{Path: stringPtr("media/tag.png")},
			AutoComplete: stringPtr(strings.Join(parts, ", ")),
			Valid:        boolPtr(false),
		}
		cmdMod := Mod{
			Subtitle:  "Remove all tags",
			Icon:      &Icon{Path: stringPtr("media/tag-delete.png")},
			Valid:     boolPtr(true),
			Variables: variables,
		}
		cmdMod.setVar("tags", "__NONE__")
		item.Mods = &map[string]Mod{"cmd": cmdMod}
		wf.addItem(item)
	}

	// Edit Category
	currentCategory := variables["category"]
	if currentCategory == "__NONE__" {
		currentCategory = ""
	} else if currentCategory == "" && link.Category != nil {
		currentCategory = *link.Category
	}
	categoryRe := regexp.MustCompile(`^/(\w*)$`)
	if matches := categoryRe.FindStringSubmatch(input); matches != nil {
		match := strings.ToLower(matches[1])
		if categories, _ := a.cache.getData("categories"); categories != nil {
			// Set a category
			for _, category := range strings.Split(*categories, ",") {
				if category == currentCategory {
					continue
				}
				if match == "" || strings.HasPrefix(strings.ToLower(category), match) {
					item := Item{
						Title:        "Set Category: " + category,
						AutoComplete: stringPtr("/" + category),
						Icon:         &Icon{Path: stringPtr("media/category.png")},
						Variables:    variables,
					}
					item.setVar("category", category)
					wf.addItem(item, true)
				}
			}
		}
	} else if currentCategory != "" {
		// Show current category
		item := Item{
			Title:        currentCategory,
			Subtitle:     "Edit Category",
			AutoComplete: stringPtr("/" + currentCategory),
			Icon:         &Icon{Path: stringPtr("media/category.png")},
			Valid:        boolPtr(false),
		}
		cmdMod := Mod{
			Subtitle:  "Remove category",
			Icon:      &Icon{Path: stringPtr("media/category-delete.png")},
			Valid:     boolPtr(true),
			Variables: variables,
		}
		cmdMod.setVar("category", "__NONE__")
		item.Mods = &map[string]Mod{"cmd": cmdMod}
		wf.addItem(item)
	}

	// Edit Lists
	currentListIDs := strings.Split(variables["listIDs"], ",")
	if variables["listIDs"] == "__NONE__" {
		currentListIDs = []string{}
	} else if variables["listIDs"] == "" && link.ListIDs != nil {
		currentListIDs = link.ListIDs
	}
	if strings.HasPrefix(input, "@") || len(currentListIDs) > 0 {
		listsMap := make(map[string]bool)
		for _, listID := range currentListIDs {
			listsMap[listID] = true
		}
		listNamesMap := make(map[string]string)
		if lists, _ := a.cache.getLists(nil); lists != nil {
			for _, list := range lists {
				listNamesMap[*list.ID] = *list.Name
			}
		}

		if strings.HasPrefix(input, "@") {
			match := strings.TrimPrefix(input, "@")
			// Add to an existing list
			match = strings.ToLower(match)
			for listID, listName := range listNamesMap {
				if listsMap[listID] {
					continue
				}
				if match == "" || strings.Contains(strings.ToLower(listName), match) {
					item := Item{
						Title:     "Add to List: " + listName,
						Variables: variables,
						Icon:      &Icon{Path: stringPtr("media/list.png")},
					}
					item.setVar("listIDs", strings.Join(append(currentListIDs, listID), ","))
					wf.addItem(item, true)
				}
			}
		}

		// Show current lists
		for listID, listName := range listNamesMap {
			if !listsMap[listID] {
				continue
			}
			item := Item{
				Title: listName,
				Icon:  &Icon{Path: stringPtr("media/list.png")},
				Valid: boolPtr(false),
			}
			cmdMod := Mod{
				Subtitle:  "Remove from list",
				Icon:      &Icon{Path: stringPtr("media/list-delete.png")},
				Valid:     boolPtr(true),
				Variables: variables,
			}
			listIDs := []string{}
			for _, id := range currentListIDs {
				if id != listID {
					listIDs = append(listIDs, id)
				}
			}
			cmdMod.setVar("listIDs", strings.Join(listIDs, ","))
			item.Mods = &map[string]Mod{"cmd": cmdMod}
			wf.addItem(item)
		}
	}

	// Done
	currentDone := variables["done"] == "true"
	if variables["done"] == "" {
		currentDone = link.Done
	}
	if currentDone {
		item := Item{
			Title: "Done",
			Icon:  &Icon{Path: stringPtr("media/checked.png")},
			Valid: boolPtr(false),
		}
		cmdMod := Mod{
			Subtitle:  "Mark as not done",
			Icon:      &Icon{Path: stringPtr("media/unchecked.png")},
			Valid:     boolPtr(true),
			Variables: variables,
		}
		cmdMod.setVar("done", "false")
		item.Mods = &map[string]Mod{"cmd": cmdMod}
		wf.addItem(item)
	} else if input == ".d" {
		item := Item{
			Title:     "Mark as done",
			Icon:      &Icon{Path: stringPtr("media/checked.png")},
			Variables: variables,
		}
		item.setVar("done", "true")
		wf.addItem(item, true)
	}

	wf.setVar("mode", "edit-link")
	wf.output()
}

func (a *Airtable) saveLink() error {
	link := Link{}
	if os.Getenv("ID") != "" {
		if links, _ := a.cache.getLinks(nil, stringPtr(os.Getenv("ID"))); len(links) > 0 {
			link = links[0]
		}
	}

	if os.Getenv("URL") != "" {
		link.URL = stringPtr(os.Getenv("URL"))
	}
	if !testURL(*link.URL) {
		return fmt.Errorf("invalid URL: %s", *link.URL)
	}
	if os.Getenv("title") != "" {
		link.Name = stringPtr(os.Getenv("title"))
	}
	if link.Name == nil || *link.Name == "" {
		link.Name = link.URL
	}
	if os.Getenv("note") != "" {
		if os.Getenv("note") == "__NONE__" {
			link.Note = nil
		} else {
			link.Note = stringPtr(os.Getenv("note"))
		}
	}
	if os.Getenv("category") != "" {
		if os.Getenv("category") == "__NONE__" {
			link.Category = nil
		} else if categories, _ := a.cache.getData("categories"); categories != nil {
			for _, category := range strings.Split(*categories, ",") {
				if category == os.Getenv("category") {
					link.Category = stringPtr(os.Getenv("category"))
					break
				}
			}
		}
	}
	if os.Getenv("tags") != "" {
		if os.Getenv("tags") == "__NONE__" {
			link.Tags = nil
		} else {
			link.Tags = strings.Split(os.Getenv("tags"), ",")
		}
	}
	if os.Getenv("listIDs") != "" {
		if os.Getenv("listIDs") == "__NONE__" {
			link.ListIDs = nil
		} else {
			link.ListIDs = strings.Split(os.Getenv("listIDs"), ",")
		}
	}
	if os.Getenv("done") != "" {
		link.Done = os.Getenv("done") == "true"
	}

	var wg sync.WaitGroup
	wg.Add(1)

	errChan := make(chan error, 1)

	if link.ID == nil {
		// create a new link
		go func() {
			defer wg.Done()
			err := a.createLink(&link)
			errChan <- err
		}()
	} else {
		// update the link
		go func() {
			defer wg.Done()
			err := a.updateLink(&link)
			errChan <- err
		}()
	}

	wg.Wait()
	err := <-errChan
	return err
}
