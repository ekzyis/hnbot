package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/dustin/go-humanize"
	"github.com/ekzyis/sn-goapi"
)

func CurateContentForStackerNews() (*[]Story, error) {
	var (
		rows *sql.Rows
		err  error
	)
	if rows, err = db.Query(`
		SELECT t.id, time, title, url, author, score, ndescendants
		FROM (
			SELECT id, MAX(created_at) AS created_at FROM hn_items
			WHERE rank = 1 AND id NOT IN (SELECT hn_id FROM sn_items) AND length(title) >= 5
			GROUP BY id
			ORDER BY time ASC
			LIMIT 1
		) t JOIN hn_items ON t.id = hn_items.id AND t.created_at = hn_items.created_at;
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
	url := story.Url
	if url == "" {
		url = HackerNewsItemLink(story.ID)
	}
	log.Printf("Posting to SN (url=%s) ...\n", url)

	if !options.SkipDupes {
		dupes, err := sn.Dupes(url)
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

	parentId, err := sn.PostLink(url, title, "tech")
	if err != nil {
		return -1, fmt.Errorf("error posting link: %w", err)
	}

	log.Printf("Posting to SN (url=%s) ... OK \n", url)
	if err := SaveSnItem(parentId, story.ID); err != nil {
		return -1, err
	}

	SendStackerNewsEmbedToDiscord(story.Title, parentId)

	comment := fmt.Sprintf(
		"This link was posted by [%s](%s) %s on [HN](%s). It received %d points and %d comments.",
		story.By,
		HackerNewsUserLink(story.By),
		humanize.Time(time.Unix(int64(story.Time), 0)),
		HackerNewsItemLink(story.ID),
		story.Score, story.Descendants,
	) + fmt.Sprintf("\n\nhttps://files.ekzyis.com/public/hn/hn_%d.png", story.ID)
	if _, err := sn.CreateComment(parentId, comment); err != nil {
		return -1, fmt.Errorf("error posting comment :%w", err)
	}
	return parentId, nil
}

func SendStackerNewsEmbedToDiscord(title string, id int) {
	Timestamp := time.Now().Format(time.RFC3339)
	url := sn.FormatLink(id)
	color := 0xffc107
	embed := discordgo.MessageEmbed{
		Title: title,
		URL:   url,
		Color: color,
		Footer: &discordgo.MessageEmbedFooter{
			Text:    "Stacker News",
			IconURL: "https://stacker.news/favicon.png",
		},
		Timestamp: Timestamp,
	}
	SendEmbedToDiscord(&embed)
}

func SendNotificationsEmbedToDiscord() {
	Timestamp := time.Now().Format(time.RFC3339)
	color := 0xffc107
	embed := discordgo.MessageEmbed{
		Title: "new notifications",
		URL:   "https://stacker.news/hn/posts",
		Color: color,
		Footer: &discordgo.MessageEmbedFooter{
			Text:    "Stacker News",
			IconURL: "https://stacker.news/favicon-notify.png",
		},
		Timestamp: Timestamp,
	}
	SendEmbedToDiscord(&embed)
}
