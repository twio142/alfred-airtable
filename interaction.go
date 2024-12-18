package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"
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
		Match:        stringPtr(""), // TODO:
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
		Variables: &map[string]string{
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
				Subtitle:  "Rebuild cache",
				Arg:       "__CACHE__",
				Icon:      &Icon{Path: stringPtr("media/reload.png")},
				Variables: map[string]string{"mode": "cache"},
			},
		},
	}

	if !l.Done {
		(*item.Mods)["cmd"] = Mod{
			Subtitle: "Mark as done 􀃲 ",
			Icon:     &Icon{Path: stringPtr("media/checked.png")},
			Variables: map[string]string{
				"ID":   *l.ID,
				"mode": "complete",
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
		Match:    stringPtr(""), // TODO:
		Text: struct {
			Copy      *string `json:"copy,omitempty"`
			LargeType *string `json:"largetype,omitempty"`
		}{
			Copy:      l.RecordURL,
			LargeType: &largetype,
		},
		Icon: &Icon{Path: stringPtr("media/list.png")},
		Variables: &map[string]string{
			"listID": *l.ID,
			"rURL":   *l.RecordURL,
		},
		Mods: &map[string]Mod{
			"cmd": {
				Subtitle: "Add link to list",
				Icon:     &Icon{Path: stringPtr("media/add.png")},
				Variables: map[string]string{
					"mode":     "append",
					"listID":   *l.ID,
					"listName": *l.Name,
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
					"mode":   "delete-links",
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
				Arg:      "__CACHE__",
				Icon:     &Icon{Path: stringPtr("media/reload.png")},
				Variables: map[string]string{
					"mode": "cache",
				},
			},
		},
	}

	return item
}

// list all links or links in a list
func (a *Airtable) listLinks(list *List) {
	wf := Workflow{}
	links, err := a.Cache.getLinks(list, nil)
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
	lists, err := a.Cache.getLists(nil)
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
	link := Link{}
	variables := map[string]string{}
	if os.Getenv("ID") != "" {
		variables["ID"] = os.Getenv("ID")
		link.ID = stringPtr(os.Getenv("ID"))
		links, _ := a.Cache.getLinks(nil, link.ID)
		if len(links) > 0 {
			link = links[0]
		}
	}
	if os.Getenv("title") != "" {
		variables["Title"] = os.Getenv("title")
		link.Name = stringPtr(os.Getenv("title"))
	}
	if os.Getenv("URL") != "" {
		variables["URL"] = os.Getenv("URL")
		link.URL = stringPtr(os.Getenv("URL"))
	}
	if os.Getenv("tags") != "" {
		variables["Tags"] = os.Getenv("tags")
		if os.Getenv("tags") == "__NONE__" {
			link.Tags = []string{}
		} else {
			link.Tags = strings.Split(os.Getenv("tags"), ",")
		}
	}
	if os.Getenv("note") != "" {
		variables["Note"] = os.Getenv("note")
		if os.Getenv("note") == "  " { // two spaces
			link.Note = nil
		} else {
			link.Note = stringPtr(os.Getenv("note"))
		}
	}
	if os.Getenv("category") != "" {
		variables["Category"] = os.Getenv("category")
		if os.Getenv("category") == "__NONE__" {
			link.Category = nil
		} else {
			link.Category = stringPtr(os.Getenv("category"))
		}
	}
	if os.Getenv("done") == "true" {
		variables["Done"] = "true"
		link.Done = true
	} else if os.Getenv("done") == "false" {
		variables["Done"] = "false"
		link.Done = false
	}
	if os.Getenv("listIDs") != "" {
		variables["ListIDs"] = os.Getenv("listIDs")
		if os.Getenv("listIDs") == "__NONE__" {
			link.ListIDs = []string{}
		} else {
			link.ListIDs = strings.Split(os.Getenv("listIDs"), ",")
		}
	}
	if link.URL == nil {
		if input == "" {
			input = os.Getenv("input")
		}
		re := regexp.MustCompile(`^(- )?\[(.+)\]\((.+?)\)$`)
		matches := re.FindStringSubmatch(input)
		if len(matches) == 4 {
			variables["Title"] = matches[2]
			variables["URL"] = matches[3]
			link.Name = stringPtr(matches[2])
			link.URL = stringPtr(matches[3])
		} else if testURL(input) {
			variables["URL"] = input
			link.URL = stringPtr(input)
		} else {
			wf.addItem(Item{
				Title:    "Save a Link to Airtable",
				Subtitle: input,
				Valid:    boolPtr(false),
			})
			wf.output()
			return
		}
	}

	// Edit URL
	if testURL(input) {
		vars := map[string]string{}
		for k, v := range variables {
			vars[k] = v
		}
		vars["URL"] = input
		wf.addItem(Item{
			Title:        "Edit URL: " + input,
			Subtitle:     "Current: " + *link.URL,
			AutoComplete: link.URL,
			QuickLookURL: link.URL,
			Valid:        boolPtr(input != *link.URL),
			Icon:         &Icon{Path: stringPtr("media/link.png")},
			Variables:    &vars,
		})
	} else {
		wf.addItem(Item{
			Title:        *link.URL,
			AutoComplete: link.URL,
			QuickLookURL: link.URL,
			Icon:         &Icon{Path: stringPtr("media/link.png")},
			Valid:        boolPtr(false),
		})
	}

	// Edit Title
	if input != "" || link.Name != nil {
		title := input
		if title == "" {
			title = *link.Name
		}
		currentTitle := ""
		if link.Name != nil {
			currentTitle = *link.Name
		}
		vars := map[string]string{}
		for k, v := range variables {
			vars[k] = v
		}
		vars["title"] = input
		wf.addItem(Item{
			Title:        fmt.Sprintf("Edit Title: '%s'", title),
			Subtitle:     fmt.Sprintf("Current: '%s'", currentTitle),
			AutoComplete: link.Name,
			Valid:        boolPtr(input != currentTitle),
			Icon:         &Icon{Path: stringPtr("media/title.png")},
			Variables:    &vars,
		})
	}

	// Edit Tags
	// check if input starts with '#'

	// Edit Category

	// Edit Lists

	// Edit Done

	// Edit Note

	// Unshift an item to save the link

	wf.setVar("mode", "edit-link")
	wf.output()
}
