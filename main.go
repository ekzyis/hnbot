package main

import (
	"log"

	"github.com/joho/godotenv"
	"github.com/namsral/flag"
)

var (
	AuthCookie string
)

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	flag.StringVar(&AuthCookie, "auth_cookie", "", "Cookie required for authorization")
	flag.Parse()
	if AuthCookie == "" {
		log.Fatal("auth cookie not set")
	}
}

func main() {
	stories := fetchTopStoriesFromHN()
	filtered := filterByRelevanceForSN(&stories)
	log.Println(filtered)
	for _, story := range *filtered {
		postToSN(&story)
	}
}
