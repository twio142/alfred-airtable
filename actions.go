package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

func handleUserActions() {
	http.HandleFunc("/action", actionHandler)
	http.ListenAndServe(":8080", nil)
}

func actionHandler(w http.ResponseWriter, r *http.Request) {
	action := r.URL.Query().Get("action")
	switch action {
	case "fetch":
		handleFetch(w, r)
	case "create":
		handleCreate(w, r)
	case "update":
		handleUpdate(w, r)
	case "delete":
		handleDelete(w, r)
	default:
		http.Error(w, "Invalid action", http.StatusBadRequest)
	}
}

func handleFetch(w http.ResponseWriter, r *http.Request) {
	// Placeholder for fetch action
	fmt.Fprintln(w, "Fetch action")
}

func handleCreate(w http.ResponseWriter, r *http.Request) {
	// Placeholder for create action
	fmt.Fprintln(w, "Create action")
}

func handleUpdate(w http.ResponseWriter, r *http.Request) {
	// Placeholder for update action
	fmt.Fprintln(w, "Update action")
}

func handleDelete(w http.ResponseWriter, r *http.Request) {
	// Placeholder for delete action
	fmt.Fprintln(w, "Delete action")
}
