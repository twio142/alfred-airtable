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
	cmd.Env = append(os.Environ(), "mode=sync")
	if len(force) > 0 && force[0] {
		cmd.Env = append(cmd.Env, "force=true")
	} else {
		cmd.Env = append(cmd.Env, "force=false")
	}
	cmd.Start()
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

	switch os.Getenv("mode") {
	case "sync":
		airtable.syncData(os.Getenv("force") == "true")
	case "list-links":
		syncInBackground()
		var list List
		if os.Getenv("listID") != "" {
			listID := os.Getenv("listID")
			list = List{ID: &listID}
		}
		airtable.listLinks(&list)
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
		err := airtable.saveLink()
		if err != nil {
			notify(err.Error())
		} else {
			notify("Link saved!")
			syncInBackground(true)
		}
	case "delete-link":
		var link Link
		if os.Getenv("ID") != "" {
			linkID := os.Getenv("ID")
			link = Link{ID: &linkID}
		} else {
			fmt.Fprintln(os.Stderr, "Error: ID is required")
			return
		}
		err := airtable.deleteLink(&link)
		if err != nil {
			notify(err.Error())
		} else {
			notify("Link deleted!")
			syncInBackground(true)
		}
	case "delete-list":
		var list List
		if os.Getenv("listID") != "" {
			listID := os.Getenv("listID")
			list = List{ID: &listID}
		} else {
			fmt.Fprintln(os.Stderr, "Error: listID is required")
			return
		}
		err := airtable.deleteList(&list, false)
		if err != nil {
			notify(err.Error())
		} else {
			notify("List deleted!")
			syncInBackground(true)
		}
	case "delete-list-links":
		var list List
		if os.Getenv("listID") != "" {
			listID := os.Getenv("listID")
			list = List{ID: &listID}
		} else {
			fmt.Fprintln(os.Stderr, "Error: listID is required")
			return
		}
		err := airtable.deleteList(&list, true)
		if err != nil {
			notify(err.Error())
		} else {
			notify("List deleted!")
			syncInBackground(true)
		}
	case "complete-link":
		var link Link
		if os.Getenv("ID") != "" {
			linkID := os.Getenv("ID")
			link = Link{ID: &linkID}
		} else {
			fmt.Fprintln(os.Stderr, "Error: ID is required")
			return
		}
		link.Done = true
		err := airtable.updateLink(&link)
		if err != nil {
			notify(err.Error())
		} else {
			notify("List maked as done!")
			syncInBackground(true)
		}
	}
}
