package main

func main() {
	stories := FetchHackerNewsTopStories()
	filtered := CurateContentForStackerNews(&stories)
	for _, story := range *filtered {
		PostStoryToStackerNews(&story)
	}
}
