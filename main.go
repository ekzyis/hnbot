package main

import (
	"log"

	"github.com/joho/godotenv"
	"github.com/namsral/flag"
)

var (
	SnUserName string
)

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	flag.StringVar(&SnUserName, "SN_USERNAME", "", "Username of bot on SN")
	flag.Parse()
	if SnUserName == "" {
		log.Fatal("SN_USERNAME not set")
	}
}

func main() {
	stories := FetchHackerNewsTopStories()
	filtered := CurateContentForStackerNews(&stories)
	for _, story := range *filtered {
		PostStoryToStackerNews(&story)
	}
}
