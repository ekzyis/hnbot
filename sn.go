package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type GraphQLPayload struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables,omitempty"`
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
	req.Header.Set("Cookie", fmt.Sprintf("__Host-next-auth.csrf-token=%s", SnApiToken))

	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	log.Printf("POST %s %d\n", url, resp.StatusCode)
}
