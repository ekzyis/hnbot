package main

import "time"

func main() {
	for {
		stories := FetchHackerNewsTopStories()
		filtered := CurateContentForStackerNews(&stories)
		for _, story := range *filtered {
			PostStoryToStackerNews(&story)
		}
		time.Sleep(time.Hour)
	}
}
