package main

import (
	"fmt"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/dustin/go-humanize"
	"github.com/ekzyis/sn-goapi"
)

func CurateContentForStackerNews(stories *[]Story) *[]Story {
	// TODO: filter by relevance

	slice := (*stories)[0:1]
	return &slice
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
	SendStackerNewsEmbedToDiscord(story.Title, parentId)

	comment := fmt.Sprintf(
		"This link was posted by [%s](%s) %s on [HN](%s). It received %d points and %d comments.",
		story.By,
		HackerNewsUserLink(story.By),
		humanize.Time(time.Unix(int64(story.Time), 0)),
		HackerNewsItemLink(story.ID),
		story.Score, story.Descendants,
	)
	sn.CreateComment(parentId, comment)
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
