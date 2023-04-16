package main

import (
	"fmt"
	"log"
	"time"

	"github.com/joho/godotenv"
	"github.com/namsral/flag"
)

var (
	SnUserName string
)

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	flag.StringVar(&SnUserName, "SN_USERNAME", "", "Username of bot on SN")
	flag.Parse()
	if SnUserName == "" {
		log.Fatal("SN_USERNAME not set")
	}
}

func GenerateHnComment(id int, sats int, nComments int) string {
	lnInvoiceDocs := "https://docs.lightning.engineering/the-lightning-network/payment-lifecycle/understanding-lightning-invoices"
	return fmt.Sprintf(
		""+
			"Your post received %d sats and %d comments on %s [0].\n\n"+
			"To claim your sats, reply to this comment with a LN address or invoice [1].\n\n"+
			"You can create a SN account to obtain a LN address.\n"+
			"\n\n"+
			"[0] %s/r/%s (referral link)\n\n"+
			"[1] %s",
		sats,
		nComments,
		StackerNewsUrl,
		StackerNewsItemLink(id),
		SnUserName,
		lnInvoiceDocs,
	)
}

func GenerateSnReply(sats int, nComments int) string {
	return fmt.Sprintf("Notified OP on HN that their post received %d sats and %d comments.", sats, nComments)
}

func main() {
	stories := FetchHackerNewsTopStories()
	filtered := CurateContentForStackerNews(&stories)
	for _, story := range *filtered {
		PostStoryToStackerNews(&story)
	}

	items := FetchStackerNewsUserItems(SnUserName)
	now := time.Now()
	for _, item := range *items {
		duration := now.Sub(item.CreatedAt)
		if duration >= 24*time.Hour && item.Sats > 0 {
			log.Printf("Found SN item (id=%d) older than 24 hours with %d sats and %d comments\n", item.Id, item.Sats, item.NComments)
			for _, comment := range item.Comments {
				if comment.User.Name == SnUserName {
					snReply := GenerateSnReply(item.Sats, item.NComments)
					// Check if OP on HN was already notified
					alreadyNotified := false
					for _, comment2 := range comment.Comments {
						if comment2.User.Name == SnUserName {
							alreadyNotified = true
						}
					}
					if alreadyNotified {
						log.Println("OP on HN was already notified")
						break
					}
					text := comment.Text
					hnItemId := FindHackerNewsItemId(text)
					hnComment := GenerateHnComment(item.Id, item.Sats, item.NComments)
					CommentHackerNewsStory(hnComment, hnItemId)
					CommentStackerNewsPost(snReply, comment.Id)
					break
				}
			}
		}
	}
}
