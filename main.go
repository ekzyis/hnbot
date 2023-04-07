package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

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

type ItemID = int

type Story struct {
	ID          ItemID
	By          string // username of author
	Time        int    // UNIX timestamp
	Descendants int    // number of comments
	Kids        []ItemID
	Score       int
	Title       string
	Url         string
}

type GraphQLPayload struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables,omitempty"`
}

func fetchTopStoriesFromHN() []Story {
	// API docs: https://github.com/HackerNews/API

	url := "https://hacker-news.firebaseio.com/v0/topstories.json"
	resp, err := http.Get(url)
	if err != nil {
		log.Fatal("Error fetching top stories:", err)
	}
	defer resp.Body.Close()
	log.Printf("GET %s %d\n", url, resp.StatusCode)

	var ids []int
	err = json.NewDecoder(resp.Body).Decode(&ids)
	if err != nil {
		log.Fatal("Error decoding top stories JSON:", err)
	}

	// we are only interested in the first page of top stories
	const limit = 30
	ids = ids[:limit]

	var stories [limit]Story
	for i, id := range ids {
		story := fetchStoryByID(id)
		stories[i] = story
	}

	// Can't return [30]Story as []Story so we copy the array
	return stories[:]
}

func fetchStoryByID(id ItemID) Story {
	url := fmt.Sprintf("https://hacker-news.firebaseio.com/v0/item/%d.json", id)
	resp, err := http.Get(url)
	if err != nil {
		log.Fatal("Error fetching story:", err)
	}
	defer resp.Body.Close()
	log.Printf("GET %s %d\n", url, resp.StatusCode)

	var story Story
	err = json.NewDecoder(resp.Body).Decode(&story)
	if err != nil {
		log.Fatal("Error decoding story JSON:", err)
	}

	return story
}

func filterByRelevanceForSN(stories *[]Story) *[]Story {
	// TODO: filter by relevance

	slice := (*stories)[0:1]
	return &slice
}

func postToSN(story *Story) {
	// TODO: check for dupes first

	body := GraphQLPayload{
		Query: `
            mutation upsertLink($url: String!, $title: String!) {
                upsertLink(url: $url, title: $title) {
                    id
                }
            }
        `,
		Variables: map[string]interface{}{
			"url":   story.Url,
			"title": story.Title,
		},
	}

	bodyJSON, err := json.Marshal(body)
	if err != nil {
		log.Fatal("Error during json.Marshal:", err)
	}

	url := "https://stacker.news/api/graphql"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(bodyJSON))
	if err != nil {
		panic(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", AuthCookie)

	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	log.Printf("POST %s %d\n", url, resp.StatusCode)
}

func main() {
	stories := fetchTopStoriesFromHN()
	filtered := filterByRelevanceForSN(&stories)
	for _, story := range *filtered {
		postToSN(&story)
	}
}
