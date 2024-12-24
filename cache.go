package main

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Link struct {
	Name         *string    `json:"Name,omitempty"`
	Note         *string    `json:"Note,omitempty"`
	URL          *string    `json:"URL,omitempty"`
	Category     *string    `json:"Category,omitempty"`
	Tags         []string   `json:"Tags,omitempty"`
	Created      *time.Time `json:"Created,omitempty"`
	LastModified *time.Time `json:"Last Modified,omitempty"`
	RecordURL    *string    `json:"Record URL,omitempty"`
	ID           *string    `json:"ID,omitempty"`
	Done         bool       `json:"Done"`
	ListIDs      []string   `json:"Lists,omitempty"`
	ListNames    []string   `json:"List-Names,omitempty"`
}

type List struct {
	Name         *string    `json:"Name,omitempty"`
	Note         *string    `json:"Note,omitempty"`
	LinkIDs      []string   `json:"Links,omitempty"`
	LinkNames    []string   `json:"Link Names,omitempty"`
	Created      *time.Time `json:"Created,omitempty"`
	LastModified *time.Time `json:"Last Modified,omitempty"`
	RecordURL    *string    `json:"Record URL,omitempty"`
	LinksDone    *int       `json:"Links Done,omitempty"`
	Status       *string    `json:"Status,omitempty"`
	ID           *string    `json:"ID,omitempty"`
}

type Cache struct {
	file         string
	db           *sql.DB
	lastSyncedAt time.Time
	maxAge       time.Duration
}

func (c *Cache) init() error {
	c.maxAge = 5 * time.Minute
	if os.Getenv("MAX_AGE") != "" {
		interval, err := time.ParseDuration(os.Getenv("MAX_AGE") + "m")
		if err == nil && interval >= 0 {
			c.maxAge = interval
		}
	}

	if c.db == nil {
		db, err := sql.Open("sqlite3", c.file)
		if err != nil {
			return err
		}

		createTableQuery := `
		CREATE TABLE IF NOT EXISTS Metadata (
			Key TEXT PRIMARY KEY,
			Value TEXT
		);

		CREATE TABLE IF NOT EXISTS Links (
			Name TEXT,
			Note TEXT,
			URL TEXT,
			Category TEXT,
			Tags TEXT,
			Created DATETIME,
			LastModified DATETIME,
			RecordURL TEXT,
			ID TEXT PRIMARY KEY,
			Done BOOLEAN,
			ListIDs TEXT
		);

		CREATE TABLE IF NOT EXISTS Lists (
			Name TEXT,
			Note TEXT,
			Created DATETIME,
			LastModified DATETIME,
			RecordURL TEXT,
			ID TEXT PRIMARY KEY
		);
  `
		_, err = db.Exec(createTableQuery)
		if err != nil {
			return err
		}

		c.db = db
	}

	if str, _ := c.getData("LastSyncedAt"); str != nil {
		c.lastSyncedAt, _ = time.Parse(time.RFC3339, *str)
	}
	return nil
}

func (c *Cache) getLinks(list *List, linkID *string) ([]Link, error) {
	err := c.init()
	if err != nil {
		return nil, err
	}

	selectQuery := `
  SELECT Links.Name, Links.Note, URL, Category, Tags, Links.Created, Links.LastModified, Links.RecordURL, Links.ID, Done, ListIDs,
      GROUP_CONCAT(Lists.Name, '\n') AS ListNames
  FROM Links
  LEFT JOIN Lists ON Links.ListIDs LIKE '%' || Lists.ID || '%'
  `

	if list != nil {
		if list.ID != nil {
			selectQuery += `WHERE Links.ListIDs LIKE '%' || ? || '%' `
		} else if list.Name != nil {
			selectQuery += `WHERE Lists.Name = ? `
		} else {
			list = nil
		}
	} else if linkID != nil {
		selectQuery += `WHERE Links.ID = ? `
	}

	selectQuery += `
	GROUP BY Links.ID
	ORDER BY Done, Links.LastModified DESC;
	`

	var rows *sql.Rows
	if list != nil {
		if list.ID != nil {
			rows, err = c.db.Query(selectQuery, *list.ID)
		} else if list.Name != nil {
			rows, err = c.db.Query(selectQuery, *list.Name)
		}
	} else if linkID != nil {
		rows, err = c.db.Query(selectQuery, *linkID)
	} else {
		rows, err = c.db.Query(selectQuery)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []Link
	for rows.Next() {
		var link Link
		var tags, listIDs, listNames sql.NullString
		err = rows.Scan(&link.Name, &link.Note, &link.URL, &link.Category, &tags, &link.Created, &link.LastModified, &link.RecordURL, &link.ID, &link.Done, &listIDs, &listNames)
		if err != nil {
			return nil, err
		}
		if tags.Valid && tags.String != "" {
			link.Tags = strings.Split(tags.String, ",")
		}
		if listIDs.Valid && listIDs.String != "" {
			link.ListIDs = strings.Split(listIDs.String, ",")
		}
		if listNames.Valid && listNames.String != "" {
			link.ListNames = strings.Split(listNames.String, "\n")
		}
		links = append(links, link)
	}
	return links, nil
}

func (c *Cache) getLists(list *List) ([]List, error) {
	err := c.init()
	if err != nil {
		return nil, err
	}

	selectQuery := `
  SELECT Lists.Name, Lists.Note, Lists.Created, Lists.LastModified, Lists.RecordURL, Lists.ID,
      SUM(CASE WHEN Links.Done THEN 1 ELSE 0 END) AS done_links,
      GROUP_CONCAT(Links.ID, ',') AS link_ids,
      GROUP_CONCAT(Links.Name, '\n') AS link_names
  FROM
      Lists
  LEFT JOIN
      Links ON Links.ListIDs LIKE '%' || Lists.ID || '%'
  `
	if list != nil {
		if list.ID != nil {
			selectQuery += `WHERE Lists.ID = ? `
		} else if list.Name != nil {
			selectQuery += `WHERE Lists.Name = ? `
		} else {
			list = nil
		}
	}
	selectQuery += `
	GROUP BY Lists.ID
	ORDER BY Lists.LastModified DESC;
	`
	rows, err := c.db.Query(selectQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lists []List
	for rows.Next() {
		var list List
		var linkIDs sql.NullString
		var linkNames sql.NullString
		err = rows.Scan(&list.Name, &list.Note, &list.Created, &list.LastModified, &list.RecordURL, &list.ID, &list.LinksDone, &linkIDs, &linkNames)
		if err != nil {
			return nil, err
		}
		if linkIDs.Valid && linkIDs.String != "" {
			list.LinkIDs = strings.Split(linkIDs.String, ",")
		}
		if linkNames.Valid {
			list.LinkNames = strings.Split(linkNames.String, "\n")
		}
		status := "In progress"
		switch *list.LinksDone {
		case len(list.LinkIDs):
			status = "Done"
		case 0:
			status = "To do"
		}
		list.Status = &status
		lists = append(lists, list)
	}
	return lists, nil
}

// Save links to the database
// If a link with the ID already exists, update it
func (c *Cache) saveLinks(links []Link) error {
	err := c.init()
	if err != nil {
		return err
	}

	insertQuery := `
  INSERT OR REPLACE INTO Links (
    Name, Note, URL, Category, Tags, Created, LastModified, RecordURL, ID, Done, ListIDs
  ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
  `
	for _, link := range links {
		var tags, listIDs string
		if link.Tags != nil {
			tags = strings.Join(link.Tags, ",")
		}
		if link.ListIDs != nil {
			listIDs = strings.Join(link.ListIDs, ",")
		}
		_, err = c.db.Exec(insertQuery, link.Name, link.Note, link.URL, link.Category, tags, link.Created, link.LastModified, link.RecordURL, link.ID, link.Done, listIDs)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Cache) saveLists(lists []List) error {
	err := c.init()
	if err != nil {
		return err
	}

	insertQuery := `
	INSERT OR REPLACE INTO Lists (
		Name, Note, Created, LastModified, RecordURL, ID
	) VALUES (?, ?, ?, ?, ?, ?)
	`
	for _, list := range lists {
		_, err = c.db.Exec(insertQuery, list.Name, list.Note, list.Created, list.LastModified, list.RecordURL, list.ID)
		if err != nil {
			return err
		}
	}
	return nil
}

// Delete records from the database whose IDs are not in the list of IDs
func (c *Cache) clearDeletedRecords(table string, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	err := c.init()
	if err != nil {
		return err
	}

	existingIDsQuery := `SELECT ID FROM ` + table
	rows, err := c.db.Query(existingIDsQuery)
	if err != nil {
		return err
	}
	defer rows.Close()
	var existingIDs []string
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			return err
		}
		existingIDs = append(existingIDs, id)
	}
	idMap := make(map[string]bool)
	for _, id := range ids {
		idMap[id] = true
	}
	var idsToDelete []string
	for _, id := range existingIDs {
		if !idMap[id] {
			idsToDelete = append(idsToDelete, id)
		}
	}
	if len(idsToDelete) == 0 {
		return nil
	}
	placeholders := strings.Repeat("?,", len(idsToDelete))
	placeholders = placeholders[:len(placeholders)-1]
	deleteQuery := fmt.Sprintf(`DELETE FROM %s WHERE ID IN (%s)`, table, placeholders)
	args := make([]interface{}, len(idsToDelete))
	for i, id := range idsToDelete {
		args[i] = id
	}
	_, err = c.db.Exec(deleteQuery, args...)
	if err != nil {
		return err
	}
	return nil
}

func (c *Cache) setData(key string, value string) error {
	insertQuery := `
  INSERT OR REPLACE INTO Metadata (Key, Value) VALUES (?, ?)
  `
	_, err := c.db.Exec(insertQuery, key, value)
	if err != nil {
		return err
	}

	return nil
}

func (c *Cache) getData(key string) (*string, error) {
	selectQuery := `
  SELECT Value FROM Metadata WHERE Key = ?
  `
	var value string
	err := c.db.QueryRow(selectQuery, key).Scan(&value)
	if err != nil {
		return nil, err
	}

	return &value, nil
}

func (c *Cache) clearCache() error {
	deleteQuery := `
  DELETE FROM Metadata;
  DELETE FROM Links;
  DELETE FROM Lists;
  `
	_, err := c.db.Exec(deleteQuery)
	if err != nil {
		return err
	}
	return nil
}
