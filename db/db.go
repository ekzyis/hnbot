package db

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3"
	"gitlab.com/ekzyis/hnbot/hn"
)

var (
	_db *sql.DB
)

func init() {
	var err error
	_db, err = sql.Open("sqlite3", "hnbot.sqlite3")
	if err != nil {
		log.Fatal(err)
	}
	migrate(_db)
}

func Query(query string, args ...interface{}) (*sql.Rows, error) {
	return _db.Query(query, args...)
}

func migrate(db *sql.DB) {
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS hn_items (
			id INTEGER NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
			time TIMESTAMP WITH TIMEZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
			title TEXT NOT NULL,
			url TEXT,
			author TEXT NOT NULL,
			ndescendants INTEGER NOT NULL,
			score INTEGER NOT NULL,
			rank INTEGER NOT NULL,
			PRIMARY KEY (id, created_at)
		);
	`); err != nil {
		err = fmt.Errorf("error during migration: %w", err)
		log.Fatal(err)
	}
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS sn_items (
			id INTEGER NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
			hn_id INTEGER NOT NULL REFERENCES hn_items(id),
			PRIMARY KEY (id, hn_id)
		);
	`); err != nil {
		err = fmt.Errorf("error during migration: %w", err)
		log.Fatal(err)
	}
}

func ItemHasComment(parentId int) bool {
	var count int
	err := _db.QueryRow(`SELECT COUNT(1) FROM comments WHERE parent_id = ?`, parentId).Scan(&count)
	if err != nil {
		err = fmt.Errorf("error during item check: %w", err)
		log.Fatal(err)
	}
	return count > 0
}

func SaveHnItems(story *[]hn.Item) error {
	for i, s := range *story {
		if err := SaveHnItem(&s, i+1); err != nil {
			return err
		}
	}
	return nil
}

func SaveHnItem(s *hn.Item, rank int) error {
	if _, err := _db.Exec(`
			INSERT INTO hn_items(id, time, title, url, author, ndescendants, score, rank)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		s.ID, s.Time, s.Title, s.Url, s.By, s.Descendants, s.Score, rank); err != nil {
		err = fmt.Errorf("error during item insert: %w", err)
		return err
	}
	return nil
}

func SaveSnItem(id int, hnId int) error {
	if _, err := _db.Exec(`INSERT INTO sn_items(id, hn_id) VALUES (?, ?)`, id, hnId); err != nil {
		err = fmt.Errorf("error during sn item insert: %w", err)
		return err
	}
	return nil
}
