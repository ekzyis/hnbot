package main

import (
	"log"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFetchDupes(t *testing.T) {
	// TODO: mock HTTP request
	url := "https://en.wikipedia.org/wiki/Dishwasher_salmon"
	dupes, err := FetchStackerNewsDupes(url)
	if err != nil {
		log.Fatal(err)
	}
	assert.NotEmpty(t, *dupes, "Expected at least one duplicate")
}
