package main

func main() {
	stories := fetchTopStoriesFromHN()
	filtered := filterByRelevanceForSN(&stories)
	for _, story := range *filtered {
		postToSN(&story)
	}
}
