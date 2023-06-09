package main

import (
	"errors"
	"log"
	"time"

	"github.com/ekzyis/sn-goapi"
)

func WaitUntilNextHour() {
	now := time.Now()
	dur := now.Truncate(time.Hour).Add(time.Hour).Sub(now)
	log.Println("sleeping for", dur.Round(time.Second))
	time.Sleep(dur)
}

func WaitUntilNextMinute() {
	now := time.Now()
	dur := now.Truncate(time.Minute).Add(time.Minute).Sub(now)
	log.Println("sleeping for", dur.Round(time.Second))
	time.Sleep(dur)
}

func CheckNotifications() {
	var prevHasNewNotes bool
	for {
		hasNewNotes, err := sn.CheckNotifications()
		if err != nil {
			SendErrorToDiscord(err)
		} else {
			if !prevHasNewNotes && hasNewNotes {
				// only send embed on "rising edge"
				SendNotificationsEmbedToDiscord()
				log.Println("Forwarded to monitoring")
			} else if hasNewNotes {
				log.Println("Already forwarded")
			}
		}
		prevHasNewNotes = hasNewNotes
		WaitUntilNextMinute()
	}
}

func main() {
	go CheckNotifications()
	for {

		stories, err := FetchHackerNewsTopStories()
		if err != nil {
			SendErrorToDiscord(err)
			WaitUntilNextHour()
			continue
		}

		filtered := CurateContentForStackerNews(&stories)

		for _, story := range *filtered {
			_, err := PostStoryToStackerNews(&story, PostStoryOptions{SkipDupes: false})
			if err != nil {
				var dupesErr *sn.DupesError
				if errors.As(err, &dupesErr) {
					// SendDupesErrorToDiscord(story.ID, dupesErr)
					continue
				}
				SendErrorToDiscord(err)
				continue
			}
			log.Println("Posting to SN ... OK")
		}
		WaitUntilNextHour()
	}
}
