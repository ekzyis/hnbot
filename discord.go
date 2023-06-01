package main

import (
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/dustin/go-humanize"
	"github.com/ekzyis/sn-goapi"
	"github.com/joho/godotenv"
	"github.com/namsral/flag"
)

var (
	DiscordToken     string
	dg               *discordgo.Session
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
	dg, err = discordgo.New("Bot " + DiscordToken)
	if err != nil {
		log.Fatal("error creating discord session:", err)
	}
	dg.AddHandler(func(s *discordgo.Session, event *discordgo.Ready) {
		log.Println("Logged in as", event.User.Username)
	})
	dg.AddHandler(onMessage)
	dg.AddHandler(onMessageReact)
	dg.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsMessageContent | discordgo.IntentGuildMessageReactions
	err = dg.Open()
	if err != nil {
		log.Fatal("error opening connection to discord: ", err, " -- Is your token correct?")
	}
}

func onMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}
	hackerNewsId, err := ParseHackerNewsLink(m.Content)
	if err != nil {
		return
	}
	story, err := FetchStoryById(hackerNewsId)
	_, err = PostStoryToStackerNews(&story, PostStoryOptions{SkipDupes: false})
	if err != nil {
		var dupesErr *sn.DupesError
		if errors.As(err, &dupesErr) {
			SendDupesErrorToDiscord(hackerNewsId, dupesErr)
			return
		}
		SendErrorToDiscord(err)
	}
}

func onMessageReact(s *discordgo.Session, reaction *discordgo.MessageReactionAdd) {
	if reaction.UserID == s.State.User.ID {
		return
	}
	if reaction.Emoji.Name != "⏭️" {
		return
	}
	m, err := s.ChannelMessage(reaction.ChannelID, reaction.MessageID)
	if err != nil {
		SendErrorToDiscord(err)
		return
	}
	if len(m.Embeds) == 0 {
		return
	}
	embed := m.Embeds[0]
	if !strings.Contains(embed.Title, "dupe(s) found for") {
		return
	}
	id, err := ParseHackerNewsLink(embed.Footer.Text)
	if err != nil {
		return
	}
	story, err := FetchStoryById(id)
	if err != nil {
		SendErrorToDiscord(err)
		return
	}
	id, err = PostStoryToStackerNews(&story, PostStoryOptions{SkipDupes: true})
	if err != nil {
		SendErrorToDiscord(err)
	}
}

func SendDupesErrorToDiscord(hackerNewsId int, dupesErr *sn.DupesError) {
	msg := fmt.Sprint(dupesErr)
	log.Println(msg)

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
				Value:  sn.FormatLink(dupe.Id),
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
		Footer: &discordgo.MessageEmbedFooter{
			Text:    HackerNewsItemLink(hackerNewsId),
			IconURL: "https://news.ycombinator.com/y18.gif",
		},
	}
	SendEmbedToDiscord(&embed)
}

func SendEmbedToDiscord(embed *discordgo.MessageEmbed) {
	_, err := dg.ChannelMessageSendEmbed(DiscordChannelId, embed)
	if err != nil {
		err = fmt.Errorf("error during sending embed: %w", err)
		log.Println(err)
	}
}

func SendErrorToDiscord(err error) {
	msg := fmt.Sprint(err)
	log.Println(msg)

	embed := discordgo.MessageEmbed{
		Title:       "Error",
		Color:       0xff0000,
		Description: msg,
	}
	SendEmbedToDiscord(&embed)
}
