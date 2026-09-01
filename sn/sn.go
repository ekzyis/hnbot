package sn

import (
	"database/sql"
	"fmt"
	"html"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/ekzyis/hnbot/db"
	"github.com/ekzyis/hnbot/hn"
	sn "github.com/ekzyis/snappy"
)

type DupesError = sn.DupesError

func CurateContent() (*[]hn.Item, error) {
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

	var items []hn.Item
	for rows.Next() {
		var item hn.Item
		if err = rows.Scan(&item.ID, &item.Time, &item.Title, &item.Url, &item.By, &item.Score, &item.Descendants); err != nil {
			err = fmt.Errorf("error scanning hn_items: %w", err)
			return nil, err
		}
		items = append(items, item)
	}
	if err = rows.Err(); err != nil {
		err = fmt.Errorf("error iterating hn_items: %w", err)
		return nil, err
	}
	return &items, nil
}

type PostOptions struct {
	SkipDupes bool
}

func Post(item *hn.Item, options PostOptions) (int, error) {
	c := sn.NewClient()
	url := item.Url
	if url == "" {
		url = hn.ItemLink(item.ID)
	}
	log.Printf("post to SN: %s ...\n", url)

	if !options.SkipDupes {
		dupes, err := c.Dupes(url)
		if err != nil {
			return -1, err
		}
		if len(*dupes) > 0 {
			return -1, &sn.DupesError{Url: url, Dupes: *dupes}
		}
	}

	title := item.Title
	if len(title) > 80 {
		title = title[0:80]
	}

	comment := fmt.Sprintf(
		"This link was posted by [%s](%s) %s on [HN](%s). It received %d points and %d comments.",
		item.By,
		hn.UserLink(item.By),
		humanize.Time(time.Unix(int64(item.Time), 0)),
		hn.ItemLink(item.ID),
		item.Score, item.Descendants,
	)

	if topComment, err := hn.FetchTopComment(item.ID); err != nil {
		log.Printf("error fetching top comment for HN item %d: %v\n", item.ID, err)
	} else if topComment != nil && topComment.Text != "" {
		comment += fmt.Sprintf(
			"\n\n[Top comment](%s) by [%s](%s):\n\n%s",
			hn.ItemLink(topComment.ID),
			topComment.By,
			hn.UserLink(topComment.By),
			blockquote(truncate(htmlToMarkdown(topComment.Text), 500)),
		)
	}

	parentId, err := c.PostLink(url, title, comment, []string{"tech"})
	if err != nil {
		return -1, fmt.Errorf("error posting link: %w", err)
	}

	log.Printf("post to SN: %s ... OK \n", url)
	if err := db.SaveSnItem(parentId, item.ID); err != nil {
		return -1, err
	}

	return parentId, nil
}

var (
	htmlParagraph = regexp.MustCompile(`(?i)<p>`)
	htmlLink      = regexp.MustCompile(`(?i)<a\b[^>]*\bhref="([^"]*)"[^>]*>(.*?)</a>`)
	htmlItalic    = regexp.MustCompile(`(?i)</?i>`)
	htmlTag       = regexp.MustCompile(`<[^>]+>`)
)

// htmlToMarkdown converts an HN comment's HTML body into Markdown.
func htmlToMarkdown(s string) string {
	s = htmlParagraph.ReplaceAllString(s, "\n\n")
	s = htmlLink.ReplaceAllString(s, "[$2]($1)")
	s = htmlItalic.ReplaceAllString(s, "*")
	s = htmlTag.ReplaceAllString(s, "")
	s = html.UnescapeString(s)
	return strings.TrimSpace(s)
}

func truncate(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return strings.TrimSpace(string(r[:max])) + "…"
}

func blockquote(s string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = "> " + line
	}
	return strings.Join(lines, "\n")
}
