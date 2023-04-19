package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"

	"github.com/bwmarrin/discordgo"
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

type DiscordEmbed struct {
	Title     string             `json:"title"`
	Url       string             `json:"url"`
	Color     int                `json:"color"`
	Footer    DiscordEmbedFooter `json:"footer"`
	Timestamp string             `json:"timestamp"`
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
	PostStoryToStackerNews(&story)
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
