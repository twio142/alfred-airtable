package main

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"
)

func syncInBackground(force ...bool) {
	cmd := exec.Command(os.Args[0])
	if len(force) > 0 && force[0] {
		cmd.Env = append(os.Environ(), "mode=force-sync")
	} else {
		cmd.Env = append(os.Environ(), "mode=sync")
	}
	_ = cmd.Start()
}

func main() {
	cacheDir := os.Getenv("alfred_workflow_data")
	if cacheDir == "" {
		fmt.Fprintln(os.Stderr, "Error: alfred_workflow_data is not set")
		return
	}
	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		_ = os.Mkdir(cacheDir, 0755)
	}

	airtable := &Airtable{
		baseURL: "https://api.airtable.com/v0",
		baseID:  os.Getenv("BASE_ID"),
		dbPath:  path.Join(cacheDir, "airtable.db"),
	}
	if err := airtable.init(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		return
	}

	mode := os.Getenv("mode")
	if mode == "" {
		mode = os.Getenv("exec")
	}
	switch mode {
	case "sync":
		_ = airtable.syncData()
	case "force-sync":
		_ = airtable.syncData(true)
	case "list-links":
		syncInBackground()
		var list *List
		if listID := os.Getenv("listID"); listID != "" {
			list = &List{ID: &listID}
		}
		airtable.listLinks(list)
	case "list-lists":
		syncInBackground()
		airtable.listLists()
	case "edit-link":
		syncInBackground()
		input := ""
		if len(os.Args) > 1 {
			input = strings.Trim(os.Args[1], " ")
		}
		airtable.editLink(input)
	case "save-link":
		if err := airtable.saveLink(); err != nil {
			notify(err.Error())
		} else {
			notify("Link saved!", os.Getenv("title"))
			syncInBackground(true)
		}
	case "delete-link":
		var link *Link
		if linkID := os.Getenv("ID"); linkID != "" {
			link = &Link{ID: &linkID}
		} else {
			fmt.Fprintln(os.Stderr, "Error: ID is required")
			return
		}
		if err := airtable.deleteLink(link); err != nil {
			notify(err.Error())
		} else {
			notify("Link deleted!")
			syncInBackground(true)
		}
	case "delete-list":
		var list *List
		if listID := os.Getenv("listID"); listID != "" {
			list = &List{ID: &listID}
		} else {
			fmt.Fprintln(os.Stderr, "Error: listID is required")
			return
		}
		if err := airtable.deleteList(list, false); err != nil {
			notify(err.Error())
		} else {
			notify("List deleted!")
			syncInBackground(true)
		}
	case "delete-list-links":
		var list *List
		if listID := os.Getenv("listID"); listID != "" {
			list = &List{ID: &listID}
		} else {
			fmt.Fprintln(os.Stderr, "Error: listID is required")
			return
		}
		if err := airtable.deleteList(list, true); err != nil {
			notify(err.Error())
		} else {
			notify("List deleted!")
			syncInBackground(true)
		}
	case "complete-link":
		var link *Link
		if linkID := os.Getenv("ID"); os.Getenv("ID") != "" {
			link = &Link{ID: &linkID}
		} else {
			fmt.Fprintln(os.Stderr, "Error: ID is required")
			return
		}
		link.Done = true
		if err := airtable.updateLink(link); err != nil {
			notify(err.Error())
		} else {
			notify("Link marked as done!")
			syncInBackground(true)
		}
	case "list-to-lc":
		var list *List
		if listID := os.Getenv("listID"); listID != "" {
			list = &List{ID: &listID}
		} else {
			fmt.Fprintln(os.Stderr, "Error: listID is required")
			return
		}
		file, err := airtable.listToLinkCopier(list)
		if err != nil {
			notify(err.Error())
			return
		}
		_ = exec.Command("alfred", *file).Start()
	}
	airtable.cache.db.Close()
}
