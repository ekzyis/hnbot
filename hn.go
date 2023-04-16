package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
	"github.com/namsral/flag"
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
	HnAuthCookie          string
)

func init() {
	HackerNewsUrl = "https://news.ycombinator.com"
	HackerNewsFirebaseUrl = "https://hacker-news.firebaseio.com/v0"
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	flag.StringVar(&HnAuthCookie, "HN_AUTH_COOKIE", "", "Cookie required for authorizing requests to news.ycombinator.com")
	flag.Parse()
	if HnAuthCookie == "" {
		log.Fatal("HN_AUTH_COOKIE not set")
	}
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

func FetchHackerNewsItemHMAC(id ItemID) string {
	hnUrl := fmt.Sprintf("%s/item?id=%d", HackerNewsUrl, id)
	req, err := http.NewRequest("GET", hnUrl, nil)
	if err != nil {
		panic(err)
	}
	// Cookie header must be set to fetch the correct HMAC for posting comments
	req.Header.Set("Cookie", HnAuthCookie)
	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	log.Printf("GET %s %d\n", hnUrl, resp.StatusCode)

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal("Failed to read response body:", err)
	}

	// Find HMAC in body
	re := regexp.MustCompile(`name="hmac" value="([a-z0-9]+)"`)
	match := re.FindStringSubmatch(string(body))
	if len(match) == 0 {
		log.Fatal("No HMAC found")
	}
	hmac := match[1]

	return hmac
}

func CommentHackerNewsStory(text string, id ItemID) {
	hmac := FetchHackerNewsItemHMAC(id)

	hnUrl := fmt.Sprintf("%s/comment", HackerNewsUrl)
	data := url.Values{}
	data.Set("parent", strconv.Itoa(id))
	data.Set("goto", fmt.Sprintf("item?id=%d", id))
	data.Set("text", text)
	data.Set("hmac", hmac)
	req, err := http.NewRequest("POST", hnUrl, strings.NewReader(data.Encode()))
	if err != nil {
		panic(err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Cookie", HnAuthCookie)
	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	log.Printf("POST %s %d\n", hnUrl, resp.StatusCode)
}

func HackerNewsUserLink(user string) string {
	return fmt.Sprintf("%s/user?id=%s", HackerNewsUrl, user)
}

func HackerNewsItemLink(id int) string {
	return fmt.Sprintf("%s/item?id=%d", HackerNewsUrl, id)
}

func FindHackerNewsItemId(text string) int {
	re := regexp.MustCompile(fmt.Sprintf(`\[HN\]\(%s/item\?id=([0-9]+)\)`, HackerNewsUrl))
	match := re.FindStringSubmatch(text)
	if len(match) == 0 {
		log.Fatal("No Hacker News item URL found")
	}
	id, err := strconv.Atoi(match[1])
	if err != nil {
		panic(err)
	}
	return id
}
