package main

import (
	"encoding/json"
	"fmt"
	"os"
)

type Icon struct {
	Path *string `json:"path,omitempty"`
	Type *string `json:"type,omitempty"`
}

type Mod struct {
	Valid     *bool             `json:"valid,omitempty"`
	Subtitle  string            `json:"subtitle,omitempty"`
	Arg       string            `json:"arg,omitempty"`
	Icon      *Icon             `json:"icon,omitempty"`
	Variables map[string]string `json:"variables,omitempty"`
}

func (m *Mod) setVar(name string, value string) {
	if m.Variables == nil {
		m.Variables = make(map[string]string)
	}
	m.Variables[name] = value
}

func (m *Mod) setVars(vars map[string]string) {
	if m.Variables == nil {
		m.Variables = make(map[string]string)
	}
	for k, v := range vars {
		m.Variables[k] = v
	}
}

type Item struct {
	Title        string  `json:"title"`
	Subtitle     string  `json:"subtitle,omitempty"`
	Arg          string  `json:"arg,omitempty"`
	Valid        *bool   `json:"valid,omitempty"`
	AutoComplete *string `json:"autocomplete,omitempty"`
	Type         *string `json:"type,omitempty"`
	Match        *string `json:"match,omitempty"`
	Text         struct {
		Copy      *string `json:"copy,omitempty"`
		LargeType *string `json:"largetype,omitempty"`
	} `json:"text"`
	Action struct {
		Text *string `json:"text,omitempty"`
		File *string `json:"file,omitempty"`
		URL  *string `json:"url,omitempty"`
	} `json:"action,omitempty"`
	QuickLookURL *string           `json:"quicklookurl,omitempty"`
	Icon         *Icon             `json:"icon,omitempty"`
	Variables    map[string]string `json:"variables,omitempty"`
	Mods         *map[string]Mod   `json:"mods"`
}

func (i *Item) setVar(name string, value string) {
	if i.Variables == nil {
		i.Variables = make(map[string]string)
	}
	i.Variables[name] = value
}

func (i *Item) setVars(vars map[string]string) {
	if i.Variables == nil {
		i.Variables = make(map[string]string)
	}
	for k, v := range vars {
		i.Variables[k] = v
	}
}

type Workflow struct {
	Items     []Item            `json:"items,omitempty"`
	Variables map[string]string `json:"variables,omitempty"`
}

func (w *Workflow) addItem(item Item, prepend ...bool) {
	if len(prepend) > 0 && prepend[0] {
		w.Items = append([]Item{item}, w.Items...)
	} else {
		w.Items = append(w.Items, item)
	}
}

func (w *Workflow) warnEmpty(s ...string) {
	title := "No Result Found"
	if len(s) > 0 && s[0] != "" {
		title = s[0]
	}
	icon := os.Getenv("alfred_preferences") + "/resources/AlertCautionIcon.icns"
	if len(s) > 1 && s[1] != "" {
		icon = s[1]
	}
	valid := false
	w.Items = []Item{
		{
			Title: title,
			Valid: &valid,
			Icon:  &Icon{Path: &icon},
		},
	}
}

func (w *Workflow) setVar(name string, value string) {
	if w.Variables == nil {
		w.Variables = make(map[string]string)
	}
	w.Variables[name] = value
}

func (w *Workflow) setVars(vars map[string]string) {
	if w.Variables == nil {
		w.Variables = make(map[string]string)
	}
	for k, v := range vars {
		w.Variables[k] = v
	}
}

func (w *Workflow) output() {
	jsonItems, err := json.MarshalIndent(w, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err.Error())
		return
	}
	fmt.Println(string(jsonItems))
}
