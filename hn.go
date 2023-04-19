package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

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

var (
	HackerNewsUrl         string
	HackerNewsFirebaseUrl string
)

func init() {
	HackerNewsUrl = "https://news.ycombinator.com"
	HackerNewsFirebaseUrl = "https://hacker-news.firebaseio.com/v0"
}

func FetchHackerNewsTopStories() []Story {
	// API docs: https://github.com/HackerNews/API

	url := fmt.Sprintf("%s/topstories.json", HackerNewsFirebaseUrl)
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
		story := FetchStoryById(id)
		stories[i] = story
	}

	// Can't return [30]Story as []Story so we copy the array
	return stories[:]
}

func FetchStoryById(id ItemID) Story {
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

func HackerNewsUserLink(user string) string {
	return fmt.Sprintf("%s/user?id=%s", HackerNewsUrl, user)
}

func HackerNewsItemLink(id int) string {
	return fmt.Sprintf("%s/item?id=%d", HackerNewsUrl, id)
}
