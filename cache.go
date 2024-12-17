package main

import (
	"database/sql"
	"time"
	"strings"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

type Metadata struct {
	CachedAt  time.Time
	Tags       []string
	Categories []string
}

type Link struct {
	Name          string
	Note          string
	Url           string
	Category      string
	Tags          []string
	Created       time.Time
	LastModified  time.Time
	RecordUrl     string
	ID            string
	Done          bool
	ListIDs       []string
	ListNames     []string
}

type List struct {
	Name          string
	Note          string
	LinkIDs       []string
	Created       time.Time
	LastModified  time.Time
	RecordUrl     string
	Status        string
	ID            string
}

type Cache struct {
	file string
	db   *sql.DB
}

func (c *Cache) init() error {
	if c.db != nil {
		return nil
	}
	db, err := sql.Open("sqlite3", c.file)
	if err != nil {
		return err
	}

	createTableQuery := `
	CREATE TABLE IF NOT EXISTS Metadata (
			Key TEXT PRIMARY KEY,
			Value TEXT,
	);

	CREATE TABLE IF NOT EXISTS Links (
			Name TEXT,
			Note TEXT,
			Url TEXT,
			Category TEXT,
			Tags TEXT,
			Created DATETIME,
			LastModified DATETIME,
			RecordUrl TEXT,
			ID TEXT PRIMARY KEY,
			Done BOOLEAN,
			ListIDs TEXT
	);

	CREATE TABLE IF NOT EXISTS Lists (
			Name TEXT,
			Note TEXT,
			Created DATETIME,
			LastModified DATETIME,
			RecordUrl TEXT,
			ID TEXT PRIMARY KEY
	);
	`
	_, err = db.Exec(createTableQuery)
	if err != nil {
		return err
	}

	c.db = db
	return nil
}

func (c *Cache) getLinks(listID *string) ([]Link, error) {
    err := c.init()
    if err != nil {
        return nil, err
    }

    selectQuery := `
    SELECT Name, Note, Url, Category, Tags, Created, LastModified, RecordUrl, ID, Done, ListIDs,
            GROUP_CONCAT(Lists.Name, '\n') AS ListNames
    FROM Links
    LEFT JOIN Lists ON Links.ListIDs LIKE '%' || Lists.ID || '%'
    `

    if listID != nil {
        selectQuery += `WHERE Links.ListIDs LIKE '%' || ? || '%' `
    }

    selectQuery += `GROUP BY Links.ID`

    var rows *sql.Rows
    if listID != nil {
        rows, err = c.db.Query(selectQuery, *listID)
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
        var tags, listIDs, listNames string
        err = rows.Scan(&link.Name, &link.Note, &link.Url, &link.Category, &tags, &link.Created, &link.LastModified, &link.RecordUrl, &link.ID, &link.Done, &listIDs, &listNames)
        if err != nil {
            return nil, err
        }
        link.Tags = strings.Split(tags, ",")
        link.ListIDs = strings.Split(listIDs, ",")
        link.ListNames = strings.Split(listNames, "\n")
        links = append(links, link)
    }
    return links, nil
}

func (c *Cache) getLists() ([]List, error) {
	err := c.init()
	if err != nil {
		return nil, err
	}

	selectQuery := `
	SELECT Name, Note, Created, LastModified, RecordUrl, ID
			COUNT(Links.ID) AS total_links,
			SUM(CASE WHEN Links.Done THEN 1 ELSE 0 END) AS done_links
	FROM
			Lists
	LEFT JOIN
			Links ON Links.ListIDs LIKE '%' || Lists.ID || '%'
	GROUP BY
			Lists.ID;
	`
	rows, err := c.db.Query(selectQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lists []List
	for rows.Next() {
		var list List
		var totalLinks, doneLinks int
		err = rows.Scan(&list.Name, &list.Note, &list.Created, &list.LastModified, &list.RecordUrl, &list.ID, &totalLinks, &doneLinks)
		if err != nil {
			return nil, err
		}
		if doneLinks == totalLinks {
			list.Status = "Done"
		} else if doneLinks == 0 {
			list.Status = "To do"
		} else {
			list.Status = "In progress"
		}
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
		Name, Note, Url, Category, Tags, Created, LastModified, RecordUrl, ID, Done, ListIDs
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	for _, link := range links {
		tags := strings.Join(link.Tags, ",")
		listIDs := strings.Join(link.ListIDs, ",")
		_, err = c.db.Exec(insertQuery, link.Name, link.Note, link.Url, link.Category, tags, link.Created, link.LastModified, link.RecordUrl, link.ID, link.Done, listIDs)
		if err != nil {
			return err
		}
	}

	c.setData("CachedAt", time.Now().Format(time.RFC3339))

	return nil
}

// Delete records from the database whose IDs are not in the list of IDs
func (c *Cache) clearDeletedRecords(table string, ids []string) error {
		if table != "Links" && table != "Lists" {
				return fmt.Errorf("invalid table name: %s", table)
		}
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
        if err := rows.Scan(&id); err != nil {
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
	err := c.init()
	if err != nil {
		return err
	}

	insertQuery := `
	INSERT OR REPLACE INTO Metadata (Key, Value) VALUES (?, ?)
	`
	_, err = c.db.Exec(insertQuery, key, value)
	if err != nil {
		return err
	}

	return nil
}

func (c *Cache) getData(key string) (*string, error) {
	err := c.init()
	if err != nil {
		return nil, err
	}

	selectQuery := `
	SELECT Value FROM Metadata WHERE Key = ?
	`
	var value *string
	err = c.db.QueryRow(selectQuery, key).Scan(value)
	if err != nil {
		return nil, err
	}

	return value, nil
}

func (c *Cache) clearCache() error {
	err := c.init()
	if err != nil {
		return err
	}

	deleteQuery := `
	DELETE FROM Metadata
	DELETE FROM Links
	DELETE FROM Lists
	`
	_, err = c.db.Exec(deleteQuery)
	if err != nil {
		return err
	}
	return nil
}