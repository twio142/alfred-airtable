package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"
	"unicode"

	"github.com/mozillazg/go-pinyin"
)

func notify(subtitle string, m ...string) {
	message := ""
	if len(m) > 0 {
		message = m[0]
	}
	_ = exec.Command(
		"terminal-notifier",
		"-title",
		"Airtable",
		"-message",
		message,
		"-subtitle",
		subtitle,
		"-sender",
		"com.runningwithcrayons.Alfred",
		"-contentImage",
		"media/airtable.png",
	).Start()
}

func logMessage(level string, format string, a ...interface{}) {
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	fmt.Fprintf(os.Stderr, "%s [%s] %s\n", timestamp, level, fmt.Sprintf(format, a...))
}

var pinyinConverter = pinyin.NewArgs()

func toPinyin(s *string) string {
	out := ""
	for _, r := range *s {
		if unicode.Is(unicode.Han, r) {
			c := pinyin.Pinyin(string(r), pinyinConverter)
			if len(c) > 0 && len(c[0]) > 0 {
				for _, p := range c[0] {
					out += p + " "
				}
			}
		} else if unicode.IsLetter(r) || unicode.IsDigit(r) {
			out += string(r)
		} else {
			out += " "
		}
	}
	return out
}

func (l *Link) match() *string {
	parts := []string{*l.Name, toPinyin(l.Name)}
	u, err := url.Parse(*l.URL)
	if err == nil {
		host := strings.TrimPrefix(u.Host, "www.")
		parts = append(parts, host)
	}
	if l.Note != nil {
		parts = append(parts, *l.Note)
	}
	if l.Category != nil {
		parts = append(parts, "/"+*l.Category)
	}
	for _, tag := range l.Tags {
		parts = append(parts, "#"+tag)
	}
	parts = append(parts, l.ListNames...)
	return stringPtr(strings.Join(parts, " "))
}

func (l *List) match() *string {
	parts := []string{*l.Name, toPinyin(l.Name)}
	if l.Note != nil {
		parts = append(parts, *l.Note)
	}
	return stringPtr(strings.Join(parts, " "))
}

func testURL(URL string) bool {
	u, err := url.Parse(URL)
	return err == nil && u.Scheme != "" && u.Host != ""
}

// func getURLTitle(URL string) string {
// }

func (l *Link) toRecord() Record {
	fields := map[string]interface{}{
		"Done": l.Done,
	}
	if l.Name != nil {
		fields["Name"] = *l.Name
	}
	if l.Note != nil {
		fields["Note"] = *l.Note
	}
	if l.URL != nil {
		fields["URL"] = *l.URL
	}
	if l.Category != nil {
		fields["Category"] = *l.Category
	}
	if len(l.Tags) > 0 {
		fields["Tags"] = l.Tags
	}
	if len(l.ListIDs) > 0 {
		fields["Lists"] = l.ListIDs
	}

	return Record{
		Fields: &fields,
		ID:     l.ID,
	}
}

func (l *List) toRecord() Record {
	fields := map[string]interface{}{}
	if l.Name != nil {
		fields["Name"] = *l.Name
	}
	if l.Note != nil {
		fields["Note"] = *l.Note
	}
	if len(l.LinkIDs) > 0 {
		fields["Links"] = l.LinkIDs
	}

	return Record{
		Fields: &fields,
		ID:     l.ID,
	}
}

func (r *Record) toLink() *Link {
	fields := *r.Fields
	link := Link{
		ID:           r.ID,
		Done:         getBoolField(fields, "Done"),
		Name:         getStringField(fields, "Name"),
		Note:         getStringField(fields, "Note"),
		URL:          getStringField(fields, "URL"),
		Category:     getStringField(fields, "Category"),
		Tags:         getStringSliceField(fields, "Tags"),
		ListIDs:      getStringSliceField(fields, "Lists"),
		Created:      r.CreatedTime,
		LastModified: getTimeField(fields, "Last Modified"),
		RecordURL:    getStringField(fields, "Record URL"),
	}
	return &link
}

func (r *Record) toList() *List {
	fields := *r.Fields
	list := List{
		ID:           r.ID,
		Name:         getStringField(fields, "Name"),
		Note:         getStringField(fields, "Note"),
		LinkIDs:      getStringSliceField(fields, "Links"),
		Created:      r.CreatedTime,
		LastModified: getTimeField(fields, "Last Modified"),
		RecordURL:    getStringField(fields, "Record URL"),
	}
	return &list
}

func stringPtr(s string) *string {
	return &s
}

func boolPtr(b bool) *bool {
	return &b
}

func getStringField(fields map[string]interface{}, key string) *string {
	if value, ok := fields[key]; ok {
		if str, ok := value.(string); ok {
			return &str
		}
	}
	return nil
}

func getStringSliceField(fields map[string]interface{}, key string) []string {
	if value, ok := fields[key]; ok {
		if slice, ok := value.([]interface{}); ok && len(slice) > 0 {
			strSlice := make([]string, len(slice))
			for i, v := range slice {
				strSlice[i] = fmt.Sprintf("%v", v)
			}
			return strSlice
		}
	}
	return []string{}
}

func getTimeField(fields map[string]interface{}, key string) *time.Time {
	if value, ok := fields[key]; ok {
		if str, ok := value.(string); ok {
			t, err := time.Parse(time.RFC3339, str)
			if err == nil {
				return &t
			}
		}
	}
	return nil
}

func getBoolField(fields map[string]interface{}, key string) bool {
	if value, ok := fields[key]; ok {
		if b, ok := value.(bool); ok {
			return b
		}
	}
	return false
}

func randomString(length int) (string, error) {
	b := make([]byte, length)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func createCodeChallenge(codeVerifier string) string {
	hash := sha256.Sum256([]byte(codeVerifier))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}
