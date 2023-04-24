package main

import (
	"errors"
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/dustin/go-humanize"
	"github.com/joho/godotenv"
	"github.com/namsral/flag"
)

var (
	DiscordToken     string
	DiscordClient    *discordgo.Session
	DiscordChannelId string
)

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	flag.StringVar(&DiscordToken, "DISCORD_TOKEN", "", "Discord bot token")
	flag.StringVar(&DiscordChannelId, "DISCORD_CHANNEL_ID", "", "Discord channel id")
	flag.Parse()
	if DiscordToken == "" {
		log.Fatal("DISCORD_TOKEN not set")
	}
	if DiscordChannelId == "" {
		log.Fatal("DISCORD_CHANNEL_ID not set")
	}
	initBot()
}

func initBot() {
	var err error
	DiscordClient, err = discordgo.New("Bot " + DiscordToken)
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
	var fields []*discordgo.MessageEmbedField
	for _, dupe := range dupesErr.Dupes {
		fields = append(fields,
			&discordgo.MessageEmbedField{
				Name:   "Title",
				Value:  dupe.Title,
				Inline: false,
			},
			&discordgo.MessageEmbedField{
				Name:   "Id",
				Value:  StackerNewsItemLink(dupe.Id),
				Inline: true,
			},
			&discordgo.MessageEmbedField{
				Name:   "Url",
				Value:  dupe.Url,
				Inline: true,
			},
			&discordgo.MessageEmbedField{
				Name:   "User",
				Value:  dupe.User.Name,
				Inline: true,
			},
			&discordgo.MessageEmbedField{
				Name:   "Created",
				Value:  humanize.Time(dupe.CreatedAt),
				Inline: true,
			},
			&discordgo.MessageEmbedField{
				Name:   "Sats",
				Value:  fmt.Sprint(dupe.Sats),
				Inline: true,
			},
			&discordgo.MessageEmbedField{
				Name:   "Comments",
				Value:  fmt.Sprint(dupe.NComments),
				Inline: true,
			},
		)
	}
	embed := discordgo.MessageEmbed{
		Title:  title,
		Color:  color,
		Fields: fields,
	}
	SendEmbedToDiscord(&embed)
}

func SendEmbedToDiscord(embed *discordgo.MessageEmbed) {
	_, err := DiscordClient.ChannelMessageSendEmbed(DiscordChannelId, embed)
	if err != nil {
		log.Fatal("Error during json.Marshal:", err)
	}
}
