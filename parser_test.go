package main

import "testing"

func TestParseFile(t *testing.T) {
	demos := make(chan demo)
	go crawlBookmarkFile("test.html", demos)
	for d := range demos {
		t.Log(d.name)
		t.Log(d.code)
	}
}
