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

type GraphQLPayload struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables,omitempty"`
}

type Dupe struct {
	Id    int    `json:"id,string"`
	Url   string `json:"url"`
	Title string `json:"title"`
}

type DupesResponse struct {
	Data struct {
		Dupes []Dupe `json:"dupes"`
	} `json:"data"`
}

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

func makeGraphQLRequest(body GraphQLPayload) *http.Response {
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
	req.Header.Set("Cookie", fmt.Sprintf("__Host-next-auth.csrf-token=%s", SnApiToken))

	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}

	log.Printf("POST %s %d\n", url, resp.StatusCode)

	return resp
}

func filterByRelevanceForSN(stories *[]Story) *[]Story {
	// TODO: filter by relevance

	slice := (*stories)[0:1]
	return &slice
}

func fetchDupes(url string) *[]Dupe {
	body := GraphQLPayload{
		Query: `
			query Dupes($url: String!) {
				dupes(url: $url) {
					id
					url
					title
				}
			}`,
		Variables: map[string]interface{}{
			"url": url,
		},
	}
	resp := makeGraphQLRequest(body)
	defer resp.Body.Close()

	var dupesResp DupesResponse
	err := json.NewDecoder(resp.Body).Decode(&dupesResp)
	if err != nil {
		log.Fatal("Error decoding dupes JSON:", err)
	}

	return &dupesResp.Data.Dupes
}

func postToSN(story *Story) {
	dupes := fetchDupes(story.Url)
	if len(*dupes) > 0 {
		return
	}

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
	resp := makeGraphQLRequest(body)
	defer resp.Body.Close()
}
