package main

import (
	"errors"
	"log"
	"time"
)

func WaitUntilNextHour() {
	now := time.Now()
	dur := now.Truncate(time.Hour).Add(time.Hour).Sub(now)
	log.Println("sleeping for", dur.Round(time.Second))
	time.Sleep(dur)
}

func main() {
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
				var dupesErr *DupesError
				if errors.As(err, &dupesErr) {
					SendDupesErrorToDiscord(story.ID, dupesErr)
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
