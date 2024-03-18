package main

import (
	"errors"
	"log"
	"time"

	"github.com/ekzyis/sn-goapi"
)

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
