package main

import (
	"errors"
	"log"
	"time"
)

func main() {
	for {

		stories, err := FetchHackerNewsTopStories()
		if err != nil {
			SendErrorToDiscord(err)
			time.Sleep(time.Hour)
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
		time.Sleep(time.Hour)
	}
}
