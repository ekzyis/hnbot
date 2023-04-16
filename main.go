package main

import (
	"log"

	"github.com/joho/godotenv"
	"github.com/namsral/flag"
)

var (
	SnApiToken string
)

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	flag.StringVar(&SnApiToken, "NEXT_AUTH_CSRF_TOKEN", "", "Token required for authorizing requests to stacker.news/api/graphql")
	flag.Parse()
	if SnApiToken == "" {
		log.Fatal("NEXT_AUTH_CSRF_TOKEN not set")
	}
}

func main() {
	stories := fetchTopStoriesFromHN()
	filtered := filterByRelevanceForSN(&stories)
	for _, story := range *filtered {
		postToSN(&story)
	}
}
