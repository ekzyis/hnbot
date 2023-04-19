package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"

	"github.com/joho/godotenv"
	"github.com/namsral/flag"
)

var (
	DiscordWebhook string
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
	flag.Parse()
	if DiscordWebhook == "" {
		log.Fatal("DISCORD_WEBHOOK not set")
	}
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
