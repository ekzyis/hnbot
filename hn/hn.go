package hn

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
)

type Item struct {
	ID          int
	By          string // username of author
	Time        int    // UNIX timestamp
	Descendants int    // number of comments
	Kids        []int
	Score       int
	Title       string
	Url         string
	Text        string // HTML body, set for comments
}

var (
	hnUrl         = "https://news.ycombinator.com"
	hnFirebaseUrl = "https://hacker-news.firebaseio.com/v0"
	hnLinkRegexp  = regexp.MustCompile(`(?:https?:\/\/)?news\.ycombinator\.com\/item\?id=([0-9]+)`)
)

func FetchTopItems() ([]Item, error) {
	log.Println("[hn] fetch top items ...")

	// API docs: https://github.com/HackerNews/API
	url := fmt.Sprintf("%s/topstories.json", hnFirebaseUrl)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error fetching HN top stories %w:", err)
	}
	defer resp.Body.Close()

	var ids []int
	err = json.NewDecoder(resp.Body).Decode(&ids)
	if err != nil {
		return nil, fmt.Errorf("error decoding HN top stories JSON: %w", err)
	}

	// we are only interested in the first page of top stories
	const limit = 30
	ids = ids[:limit]

	var stories [limit]Item
	for i, id := range ids {
		var item Item
		err := FetchItemById(id, &item)
		if err != nil {
			return nil, err
		}
		stories[i] = item
	}

	log.Println("[hn] fetch top items ... OK")
	// Can't return [30]Item as []Item so we copy the array
	return stories[:], nil
}

func FetchItemById(id int, hnItem *Item) error {
	// log.Printf("[hn] fetch HN item %d ...\n", id)

	url := fmt.Sprintf("https://hacker-news.firebaseio.com/v0/item/%d.json", id)
	resp, err := http.Get(url)
	if err != nil {
		err = fmt.Errorf("error fetching HN item %d: %w", id, err)
		return err
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&hnItem)
	if err != nil {
		err := fmt.Errorf("error decoding JSON for HN item %d: %w", id, err)
		return err
	}

	// log.Printf("[hn] fetch HN item %d ... OK\n", id)
	return nil
}

// FetchTopComment returns the highest-ranked comment of the given item,
// or nil if the item has no comments.
func FetchTopComment(id int) (*Item, error) {
	var item Item
	if err := FetchItemById(id, &item); err != nil {
		return nil, err
	}
	if len(item.Kids) == 0 {
		return nil, nil
	}
	var comment Item
	if err := FetchItemById(item.Kids[0], &comment); err != nil {
		return nil, err
	}
	return &comment, nil
}

func ParseLink(link string) (int, error) {
	match := hnLinkRegexp.FindStringSubmatch(link)
	if len(match) == 0 {
		return -1, errors.New("not a hacker news link")
	}
	id, err := strconv.Atoi(match[1])
	if err != nil {
		return -1, errors.New("integer conversion to string failed")
	}
	return id, nil
}

func UserLink(user string) string {
	return fmt.Sprintf("%s/user?id=%s", hnUrl, user)
}

func ItemLink(id int) string {
	return fmt.Sprintf("%s/item?id=%d", hnUrl, id)
}
