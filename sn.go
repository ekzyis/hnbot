package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/dustin/go-humanize"
	sn "github.com/ekzyis/snappy"
)

func CurateContentForStackerNews() (*[]Story, error) {
	var (
		rows *sql.Rows
		err  error
	)
	if rows, err = db.Query(`
		SELECT t.id, time, title, url, author, score, ndescendants
		FROM (
			SELECT id, MIN(created_at) AS start, MAX(created_at) AS end
			FROM hn_items
			WHERE rank = 1 AND id NOT IN (SELECT hn_id FROM sn_items) AND length(title) >= 5
			GROUP BY id
			HAVING unixepoch(end) - unixepoch(start) >= 3600
			ORDER BY time ASC
			LIMIT 1
		) t JOIN hn_items ON t.id = hn_items.id AND t.end = hn_items.created_at;
	`); err != nil {
		err = fmt.Errorf("error querying hn_items: %w", err)
		return nil, err
	}
	defer rows.Close()

	var stories []Story
	for rows.Next() {
		var story Story
		if err = rows.Scan(&story.ID, &story.Time, &story.Title, &story.Url, &story.By, &story.Score, &story.Descendants); err != nil {
			err = fmt.Errorf("error scanning hn_items: %w", err)
			return nil, err
		}
		stories = append(stories, story)
	}
	if err = rows.Err(); err != nil {
		err = fmt.Errorf("error iterating hn_items: %w", err)
		return nil, err
	}
	return &stories, nil
}

type PostStoryOptions struct {
	SkipDupes bool
}

func PostStoryToStackerNews(story *Story, options PostStoryOptions) (int, error) {
	c := sn.NewClient()
	url := story.Url
	if url == "" {
		url = HackerNewsItemLink(story.ID)
	}
	log.Printf("Posting to SN (url=%s) ...\n", url)

	if !options.SkipDupes {
		dupes, err := c.Dupes(url)
		if err != nil {
			return -1, err
		}
		if len(*dupes) > 0 {
			return -1, &sn.DupesError{Url: url, Dupes: *dupes}
		}
	}

	title := story.Title
	if len(title) > 80 {
		title = title[0:80]
	}

	comment := fmt.Sprintf(
		"This link was posted by [%s](%s) %s on [HN](%s). It received %d points and %d comments.",
		story.By,
		HackerNewsUserLink(story.By),
		humanize.Time(time.Unix(int64(story.Time), 0)),
		HackerNewsItemLink(story.ID),
		story.Score, story.Descendants,
	)

	parentId, err := c.PostLink(url, title, comment, "tech")
	if err != nil {
		return -1, fmt.Errorf("error posting link: %w", err)
	}

	log.Printf("Posting to SN (url=%s) ... OK \n", url)
	if err := SaveSnItem(parentId, story.ID); err != nil {
		return -1, err
	}

	return parentId, nil
}
