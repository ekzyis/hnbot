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

func WaitUntilNextRun() {
	now := time.Now()
	dur := now.Truncate(time.Minute).Add(15 * time.Minute).Sub(now)
	log.Println("sleeping for", dur.Round(time.Second))
	time.Sleep(dur)
}

func CheckNotifications() {
	var prevHasNewNotes bool
	for {
		log.Println("Checking notifications ...")
		hasNewNotes, err := sn.CheckNotifications()
		if err != nil {
			SendErrorToDiscord(err)
		} else {
			if !prevHasNewNotes && hasNewNotes {
				// only send embed on "rising edge"
				SendNotificationsEmbedToDiscord()
				log.Println("Forwarded notifications to monitoring")
			} else if hasNewNotes {
				log.Println("Notifications already forwarded")
			}
		}
		prevHasNewNotes = hasNewNotes
		WaitUntilNextRun()
	}
}

func SessionKeepAlive() {
	for {
		log.Println("Refresh session using GET /api/auth/session ...")
		sn.RefreshSession()
		WaitUntilNextHour()
	}
}

func SyncStories() {
	for {
		stories, err := FetchHackerNewsTopStories()
		if err != nil {
			SendErrorToDiscord(err)
			WaitUntilNextMinute()
			continue
		}

		if err := SaveStories(&stories); err != nil {
			SendErrorToDiscord(err)
			WaitUntilNextMinute()
			continue
		}

		WaitUntilNextMinute()
	}
}

func main() {
	go CheckNotifications()
	go SessionKeepAlive()
	go SyncStories()
	for {
		var (
			filtered *[]Story
			err      error
		)

		if filtered, err = CurateContentForStackerNews(); err != nil {
			SendErrorToDiscord(err)
			WaitUntilNextRun()
			continue
		}

		for _, story := range *filtered {
			_, err := PostStoryToStackerNews(&story, PostStoryOptions{SkipDupes: false})
			if err != nil {
				var dupesErr *sn.DupesError
				if errors.As(err, &dupesErr) {
					// SendDupesErrorToDiscord(story.ID, dupesErr)
					log.Println(dupesErr)
					// save dupe in db to prevent retries
					parentId := dupesErr.Dupes[0].Id
					if err := SaveSnItem(parentId, story.ID); err != nil {
						SendErrorToDiscord(err)
					}
					continue
				}
				SendErrorToDiscord(err)
				continue
			}
		}
		WaitUntilNextRun()
	}
}
