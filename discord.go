package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/bwmarrin/discordgo"
	"github.com/dustin/go-humanize"
	"github.com/joho/godotenv"
	"github.com/namsral/flag"
)

var (
	DiscordWebhook string
	DiscordToken   string
	DiscordClient  *discordgo.Session
)

type DiscordEmbedFooter struct {
	Text    string `json:"text"`
	IconUrl string `json:"icon_url"`
}

type DiscordEmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline"`
}

type DiscordEmbed struct {
	Title     string              `json:"title"`
	Url       string              `json:"url"`
	Color     int                 `json:"color"`
	Footer    DiscordEmbedFooter  `json:"footer"`
	Timestamp string              `json:"timestamp"`
	Fields    []DiscordEmbedField `json:"fields"`
}

type DiscordWebhookPayload struct {
	Embeds []DiscordEmbed `json:"embeds"`
}

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	flag.StringVar(&DiscordWebhook, "DISCORD_WEBHOOK", "", "Webhook to send logs to discord")
	flag.StringVar(&DiscordToken, "DISCORD_TOKEN", "", "Discord bot token")
	flag.Parse()
	if DiscordWebhook == "" {
		log.Fatal("DISCORD_WEBHOOK not set")
	}
	if DiscordToken == "" {
		log.Fatal("DISCORD_TOKEN not set")
	}
	initBot()
}

func initBot() {
	var err error
	DiscordClient, err = discordgo.New(DiscordToken)
	if err != nil {
		log.Fatal("error creating discord session:", err)
	}
	DiscordClient.AddHandler(func(s *discordgo.Session, event *discordgo.Ready) {
		log.Println("Logged in as", event.User.Username)
	})
	DiscordClient.AddHandler(onMessage)
	DiscordClient.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsMessageContent
	err = DiscordClient.Open()
	if err != nil {
		log.Fatal("error opening connection to discord: ", err, " -- Is your token correct?")
	}
}

func onMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore all messages created by the bot itself
	if m.Author.ID == s.State.User.ID {
		return
	}
	id, err := ParseHackerNewsLink(m.Content)
	if err != nil {
		return
	}
	story := FetchStoryById(id)
	id, err = PostStoryToStackerNews(&story)
	if err != nil {
		var dupesErr *DupesError
		if errors.As(err, &dupesErr) {
			SendDupesErrorToDiscord(dupesErr)
		} else {
			log.Fatal("unexpected error returned")
		}
	}
}

func SendDupesErrorToDiscord(dupesErr *DupesError) {
	title := fmt.Sprintf("%d dupe(s) found for %s:", len(dupesErr.Dupes), dupesErr.Url)
	color := 0xffc107
	var fields []DiscordEmbedField
	for _, dupe := range dupesErr.Dupes {
		fields = append(fields,
			DiscordEmbedField{
				Name:   "Title",
				Value:  dupe.Title,
				Inline: false,
			},
			DiscordEmbedField{
				Name:   "Id",
				Value:  StackerNewsItemLink(dupe.Id),
				Inline: true,
			},
			DiscordEmbedField{
				Name:   "Url",
				Value:  dupe.Url,
				Inline: true,
			},
			DiscordEmbedField{
				Name:   "User",
				Value:  dupe.User.Name,
				Inline: true,
			},
			DiscordEmbedField{
				Name:   "Created",
				Value:  humanize.Time(dupe.CreatedAt),
				Inline: true,
			},
			DiscordEmbedField{
				Name:   "Sats",
				Value:  fmt.Sprint(dupe.Sats),
				Inline: true,
			},
			DiscordEmbedField{
				Name:   "Comments",
				Value:  fmt.Sprint(dupe.NComments),
				Inline: true,
			},
		)
	}
	embed := DiscordEmbed{
		Title:  title,
		Color:  color,
		Fields: fields,
	}
	SendEmbedToDiscord(embed)
}

func SendEmbedToDiscord(embed DiscordEmbed) {
	bodyJSON, err := json.Marshal(
		DiscordWebhookPayload{
			Embeds: []DiscordEmbed{embed},
		},
	)
	if err != nil {
		log.Fatal("Error during json.Marshal:", err)
	}
	req, err := http.NewRequest("POST", DiscordWebhook, bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Discord webhook error:", err)
	}
	defer resp.Body.Close()
}
