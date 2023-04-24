package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/joho/godotenv"
	"github.com/namsral/flag"
)

type GraphQLPayload struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables,omitempty"`
}

type SnUser struct {
	Name string `json:"name"`
}
type Dupe struct {
	Id        int       `json:"id,string"`
	Url       string    `json:"url"`
	Title     string    `json:"title"`
	User      SnUser    `json:"user"`
	CreatedAt time.Time `json:"createdAt"`
	Sats      int       `json:"sats"`
	NComments int       `json:"ncomments"`
}

type DupesResponse struct {
	Data struct {
		Dupes []Dupe `json:"dupes"`
	} `json:"data"`
}

type DupesError struct {
	Url   string
	Dupes []Dupe
}

func (e *DupesError) Error() string {
	return fmt.Sprintf("%s has %d dupes", e.Url, len(e.Dupes))
}

type User struct {
	Name string `json:"name"`
}
type Comment struct {
	Id       int       `json:"id,string"`
	Text     string    `json:"text"`
	User     User      `json:"user"`
	Comments []Comment `json:"comments"`
}
type Item struct {
	Id        int       `json:"id,string"`
	Title     string    `json:"title"`
	Url       string    `json:"url"`
	Sats      int       `json:"sats"`
	CreatedAt time.Time `json:"createdAt"`
	Comments  []Comment `json:"comments"`
	NComments int       `json:"ncomments"`
}

type UpsertLinkResponse struct {
	Data struct {
		UpsertLink Item `json:"upsertLink"`
	} `json:"data"`
}

type ItemsResponse struct {
	Data struct {
		Items struct {
			Items  []Item `json:"items"`
			Cursor string `json:"cursor"`
		} `json:"items"`
	} `json:"data"`
}

var (
	StackerNewsUrl string
	SnApiUrl       string
	SnAuthCookie   string
)

func init() {
	StackerNewsUrl = "https://stacker.news"
	SnApiUrl = "https://stacker.news/api/graphql"
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	flag.StringVar(&SnAuthCookie, "SN_AUTH_COOKIE", "", "Cookie required for authorizing requests to stacker.news/api/graphql")
	flag.Parse()
	if SnAuthCookie == "" {
		log.Fatal("SN_AUTH_COOKIE not set")
	}
}

func MakeStackerNewsRequest(body GraphQLPayload) *http.Response {
	bodyJSON, err := json.Marshal(body)
	if err != nil {
		log.Fatal("Error during json.Marshal:", err)
	}

	req, err := http.NewRequest("POST", SnApiUrl, bytes.NewBuffer(bodyJSON))
	if err != nil {
		panic(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", SnAuthCookie)

	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}

	log.Printf("POST %s %d\n", SnApiUrl, resp.StatusCode)

	return resp
}

func CurateContentForStackerNews(stories *[]Story) *[]Story {
	// TODO: filter by relevance

	slice := (*stories)[0:1]
	return &slice
}

func FetchStackerNewsDupes(url string) *[]Dupe {
	body := GraphQLPayload{
		Query: `
			query Dupes($url: String!) {
				dupes(url: $url) {
					id
					url
					title
					user {
						name
					}
					createdAt
					sats
					ncomments
				}
			}`,
		Variables: map[string]interface{}{
			"url": url,
		},
	}
	resp := MakeStackerNewsRequest(body)
	defer resp.Body.Close()

	var dupesResp DupesResponse
	err := json.NewDecoder(resp.Body).Decode(&dupesResp)
	if err != nil {
		log.Fatal("Error decoding dupes JSON:", err)
	}

	return &dupesResp.Data.Dupes
}

func PostStoryToStackerNews(story *Story) (int, error) {
	dupes := FetchStackerNewsDupes(story.Url)
	if len(*dupes) > 0 {
		log.Printf("%s was already posted. Skipping.\n", story.Url)
		return -1, &DupesError{story.Url, *dupes}
	}

	body := GraphQLPayload{
		Query: `
			mutation upsertLink($url: String!, $title: String!) {
				upsertLink(url: $url, title: $title) {
					id
				}
			}`,
		Variables: map[string]interface{}{
			"url":   story.Url,
			"title": story.Title,
		},
	}
	resp := MakeStackerNewsRequest(body)
	defer resp.Body.Close()

	var upsertLinkResp UpsertLinkResponse
	err := json.NewDecoder(resp.Body).Decode(&upsertLinkResp)
	if err != nil {
		log.Fatal("Error decoding dupes JSON:", err)
	}
	parentId := upsertLinkResp.Data.UpsertLink.Id

	log.Println("Created new post on SN")
	log.Printf("id=%d title='%s' url=%s\n", parentId, story.Title, story.Url)
	SendStackerNewsEmbedToDiscord(story.Title, parentId)

	comment := fmt.Sprintf(
		"This link was posted by [%s](%s) %s on [HN](%s). It received %d points and %d comments.",
		story.By,
		HackerNewsUserLink(story.By),
		humanize.Time(time.Unix(int64(story.Time), 0)),
		HackerNewsItemLink(story.ID),
		story.Score, story.Descendants,
	)
	CommentStackerNewsPost(comment, parentId)
	return parentId, nil
}

func StackerNewsItemLink(id int) string {
	return fmt.Sprintf("https://stacker.news/items/%d", id)
}

func CommentStackerNewsPost(text string, parentId int) {
	body := GraphQLPayload{
		Query: `
			mutation createComment($text: String!, $parentId: ID!) {
        createComment(text: $text, parentId: $parentId) {
          id
        }
			}`,
		Variables: map[string]interface{}{
			"text":     text,
			"parentId": parentId,
		},
	}
	resp := MakeStackerNewsRequest(body)
	defer resp.Body.Close()

	log.Println("Commented post on SN")
	log.Printf("text='%s' parentId=%d\n", text, parentId)
}

func SendStackerNewsEmbedToDiscord(title string, id int) {
	Timestamp := time.Now().Format(time.RFC3339)
	url := StackerNewsItemLink(id)
	color := 0xffc107
	embed := DiscordEmbed{
		Title: title,
		Url:   url,
		Color: color,
		Footer: DiscordEmbedFooter{
			Text:    "Stacker News",
			IconUrl: "https://stacker.news/favicon.png",
		},
		Timestamp: Timestamp,
	}
	SendEmbedToDiscord(embed)
}
