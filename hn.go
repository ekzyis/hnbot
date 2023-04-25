package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
)

type Story struct {
	ID          int
	By          string // username of author
	Time        int    // UNIX timestamp
	Descendants int    // number of comments
	Kids        []int
	Score       int
	Title       string
	Url         string
}

var (
	HackerNewsUrl         = "https://news.ycombinator.com"
	HackerNewsFirebaseUrl = "https://hacker-news.firebaseio.com/v0"
	HackerNewsLinkRegexp  = regexp.MustCompile(`(?:https?:\/\/)?news\.ycombinator\.com\/item\?id=([0-9]+)`)
)

func FetchHackerNewsTopStories() ([]Story, error) {
	log.Println("Fetching HN top stories ...")

	// API docs: https://github.com/HackerNews/API
	url := fmt.Sprintf("%s/topstories.json", HackerNewsFirebaseUrl)
	resp, err := http.Get(url)
	if err != nil {
		err = fmt.Errorf("error fetching HN top stories %w:", err)
		return nil, err
	}
	defer resp.Body.Close()

	var ids []int
	err = json.NewDecoder(resp.Body).Decode(&ids)
	if err != nil {
		err = fmt.Errorf("error decoding HN top stories JSON: %w", err)
		return nil, err
	}

	// we are only interested in the first page of top stories
	const limit = 30
	ids = ids[:limit]

	var stories [limit]Story
	for i, id := range ids {
		story, err := FetchStoryById(id)
		if err != nil {
			return nil, err
		}
		stories[i] = story
	}

	log.Println("Fetching HN top stories ... OK")
	// Can't return [30]Story as []Story so we copy the array
	return stories[:], nil
}

func FetchStoryById(id int) (Story, error) {
	log.Printf("Fetching HN story (id=%d) ...\n", id)

	url := fmt.Sprintf("https://hacker-news.firebaseio.com/v0/item/%d.json", id)
	resp, err := http.Get(url)
	if err != nil {
		err = fmt.Errorf("error fetching HN story (id=%d): %w", id, err)
		return Story{}, err
	}
	defer resp.Body.Close()

	var story Story
	err = json.NewDecoder(resp.Body).Decode(&story)
	if err != nil {
		err := fmt.Errorf("error decoding HN story JSON (id=%d): %w", id, err)
		return Story{}, err
	}

	log.Printf("Fetching HN story (id=%d) ... OK\n", id)
	return story, nil
}

func ParseHackerNewsLink(link string) (int, error) {
	match := HackerNewsLinkRegexp.FindStringSubmatch(link)
	if len(match) == 0 {
		return -1, errors.New("input is not a hacker news link")
	}
	id, err := strconv.Atoi(match[1])
	if err != nil {
		return -1, errors.New("integer conversion to string failed")
	}
	return id, nil
}

func HackerNewsUserLink(user string) string {
	return fmt.Sprintf("%s/user?id=%s", HackerNewsUrl, user)
}

func HackerNewsItemLink(id int) string {
	return fmt.Sprintf("%s/item?id=%d", HackerNewsUrl, id)
}
