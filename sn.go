package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/dustin/go-humanize"
	"github.com/joho/godotenv"
	"github.com/namsral/flag"
)

type GraphQLPayload struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables,omitempty"`
}

type GraphQLError struct {
	Message string `json:"message"`
}

type User struct {
	Name string `json:"name"`
}

type Dupe struct {
	Id        int       `json:"id,string"`
	Url       string    `json:"url"`
	Title     string    `json:"title"`
	User      User      `json:"user"`
	CreatedAt time.Time `json:"createdAt"`
	Sats      int       `json:"sats"`
	NComments int       `json:"ncomments"`
}

type DupesResponse struct {
	Errors []GraphQLError `json:"errors"`
	Data   struct {
		Dupes []Dupe `json:"dupes"`
	} `json:"data"`
}

type DupesError struct {
	Url   string
	Dupes []Dupe
}

func (e *DupesError) Error() string {
	return fmt.Sprintf("found %d dupes for %s", len(e.Dupes), e.Url)
}

type Comment struct {
	Id       int       `json:"id,string"`
	Text     string    `json:"text"`
	User     User      `json:"user"`
	Comments []Comment `json:"comments"`
}

type CreateCommentsResponse struct {
	Errors []GraphQLError `json:"errors"`
	Data   struct {
		CreateComment Comment `json:"createComment"`
	} `json:"data"`
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
	Errors []GraphQLError `json:"errors"`
	Data   struct {
		UpsertLink Item `json:"upsertLink"`
	} `json:"data"`
}

type ItemsResponse struct {
	Errors []GraphQLError `json:"errors"`
	Data   struct {
		Items struct {
			Items  []Item `json:"items"`
			Cursor string `json:"cursor"`
		} `json:"items"`
	} `json:"data"`
}

type HasNewNotesResponse struct {
	Errors []GraphQLError `json:"errors"`
	Data   struct {
		HasNewNotes bool `json:"hasNewNotes"`
	} `json:"data"`
}

var (
	StackerNewsUrl = "https://stacker.news"
	SnApiUrl       = "https://stacker.news/api/graphql"
	SnAuthCookie   string
)

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("error loading .env file")
	}
	flag.StringVar(&SnAuthCookie, "SN_AUTH_COOKIE", "", "Cookie required for authorizing requests to stacker.news/api/graphql")
	flag.Parse()
	if SnAuthCookie == "" {
		log.Fatal("SN_AUTH_COOKIE not set")
	}
}

func MakeStackerNewsRequest(body GraphQLPayload) (*http.Response, error) {
	bodyJSON, err := json.Marshal(body)
	if err != nil {
		err = fmt.Errorf("error encoding SN payload: %w", err)
		return nil, err
	}

	req, err := http.NewRequest("POST", SnApiUrl, bytes.NewBuffer(bodyJSON))
	if err != nil {
		err = fmt.Errorf("error preparing SN request: %w", err)
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", SnAuthCookie)

	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		err = fmt.Errorf("error posting SN payload: %w", err)
		return nil, err
	}

	return resp, nil
}

func CurateContentForStackerNews(stories *[]Story) *[]Story {
	// TODO: filter by relevance

	slice := (*stories)[0:1]
	return &slice
}

func CheckForErrors(graphqlErrors []GraphQLError) error {
	if len(graphqlErrors) > 0 {
		errorMsg, marshalErr := json.Marshal(graphqlErrors)
		if marshalErr != nil {
			return marshalErr
		}
		return errors.New(fmt.Sprintf("error fetching SN dupes: %s", string(errorMsg)))
	}
	return nil
}

func FetchStackerNewsDupes(url string) (*[]Dupe, error) {
	log.Printf("Fetching SN dupes (url=%s) ...\n", url)

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
	resp, err := MakeStackerNewsRequest(body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var dupesResp DupesResponse
	err = json.NewDecoder(resp.Body).Decode(&dupesResp)
	if err != nil {
		err = fmt.Errorf("error decoding SN dupes: %w", err)
		return nil, err
	}
	err = CheckForErrors(dupesResp.Errors)
	if err != nil {
		return nil, err
	}

	log.Printf("Fetching SN dupes (url=%s) ... OK\n", url)
	return &dupesResp.Data.Dupes, nil
}

type PostStoryOptions struct {
	SkipDupes bool
}

func PostStoryToStackerNews(story *Story, options PostStoryOptions) (int, error) {
	log.Printf("Posting to SN (url=%s) ...\n", story.Url)

	if !options.SkipDupes {
		dupes, err := FetchStackerNewsDupes(story.Url)
		if err != nil {
			return -1, err
		}
		if len(*dupes) > 0 {
			return -1, &DupesError{story.Url, *dupes}
		}
	}

	body := GraphQLPayload{
		Query: `
			mutation upsertLink($url: String!, $title: String!, $sub: String!) {
				upsertLink(url: $url, title: $title, sub: $sub) {
					id
				}
			}`,
		Variables: map[string]interface{}{
			"url":   story.Url,
			"title": story.Title,
			"sub":   "bitcoin",
		},
	}
	resp, err := MakeStackerNewsRequest(body)
	if err != nil {
		return -1, err
	}
	defer resp.Body.Close()

	var upsertLinkResp UpsertLinkResponse
	err = json.NewDecoder(resp.Body).Decode(&upsertLinkResp)
	if err != nil {
		err = fmt.Errorf("error decoding SN upsertLink: %w", err)
		return -1, err
	}
	err = CheckForErrors(upsertLinkResp.Errors)
	if err != nil {
		return -1, err
	}
	parentId := upsertLinkResp.Data.UpsertLink.Id

	log.Printf("Posting to SN (url=%s) ... OK \n", story.Url)
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

func CommentStackerNewsPost(text string, parentId int) (*http.Response, error) {
	log.Printf("Commenting SN post (parentId=%d) ...\n", parentId)

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
	resp, err := MakeStackerNewsRequest(body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var createCommentsResp CreateCommentsResponse
	err = json.NewDecoder(resp.Body).Decode(&createCommentsResp)
	if err != nil {
		err = fmt.Errorf("error decoding SN upsertLink: %w", err)
		return nil, err
	}
	err = CheckForErrors(createCommentsResp.Errors)
	if err != nil {
		return nil, err
	}

	log.Printf("Commenting SN post (parentId=%d) ... OK\n", parentId)
	return resp, nil
}

func SendStackerNewsEmbedToDiscord(title string, id int) {
	Timestamp := time.Now().Format(time.RFC3339)
	url := StackerNewsItemLink(id)
	color := 0xffc107
	embed := discordgo.MessageEmbed{
		Title: title,
		URL:   url,
		Color: color,
		Footer: &discordgo.MessageEmbedFooter{
			Text:    "Stacker News",
			IconURL: "https://stacker.news/favicon.png",
		},
		Timestamp: Timestamp,
	}
	SendEmbedToDiscord(&embed)
}

func SendNotificationsEmbedToDiscord() {
	Timestamp := time.Now().Format(time.RFC3339)
	color := 0xffc107
	embed := discordgo.MessageEmbed{
		Title: "new notifications",
		URL:   "https://stacker.news/hn/posts",
		Color: color,
		Footer: &discordgo.MessageEmbedFooter{
			Text:    "Stacker News",
			IconURL: "https://stacker.news/favicon-notify.png",
		},
		Timestamp: Timestamp,
	}
	SendEmbedToDiscord(&embed)
}

func FetchHasNewNotes() (bool, error) {
	log.Println("Checking notifications ...")

	body := GraphQLPayload{
		Query: `
			{
				hasNewNotes
			}`,
	}
	resp, err := MakeStackerNewsRequest(body)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	var hasNewNotesResp HasNewNotesResponse
	err = json.NewDecoder(resp.Body).Decode(&hasNewNotesResp)
	if err != nil {
		err = fmt.Errorf("error decoding SN hasNewNotes: %w", err)
		return false, err
	}
	err = CheckForErrors(hasNewNotesResp.Errors)
	if err != nil {
		return false, err
	}

	hasNewNotes := hasNewNotesResp.Data.HasNewNotes

	msg := "Checking notifications ... OK - "
	if hasNewNotes {
		msg += "NEW"
	} else {
		msg += "NONE"
	}
	log.Println(msg)

	return hasNewNotes, nil
}
