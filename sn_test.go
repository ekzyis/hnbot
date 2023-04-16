package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFetchDupes(t *testing.T) {
	// TODO: mock HTTP request
	url := "https://en.wikipedia.org/wiki/Dishwasher_salmon"
	dupes := FetchStackerNewsDupes(url)
	assert.NotEmpty(t, *dupes, "Expected at least one duplicate")
}
