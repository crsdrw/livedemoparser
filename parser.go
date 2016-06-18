package main

import (
	"bufio"
	"io"
	"os"
	"strings"
	"sync"

	"net/http"

	"bytes"
	"errors"
	"fmt"

	"encoding/json"

	"io/ioutil"

	"golang.org/x/net/html"
)

type demo struct {
	name string
	code string
}

func getHref(t html.Token) (ok bool, href string) {
	for _, a := range t.Attr {
		if a.Key == "href" {
			href = a.Val
			ok = true
		}
	}
	return
}

func findWandboxCodeInText(b []byte) (bool, string) {
	const codePrefix = `var JSON_CODE = {"code":`
	var code string
	start := bytes.Index(b, []byte(codePrefix))
	if start == -1 {
		return false, code
	}
	start += len(codePrefix)
	end := bytes.Index(b, []byte(`,"compiler":"`))
	if end == 01 {
		return false, code
	}
	err := json.Unmarshal(b[start:end], &code)
	if err != nil {
		return false, code
	}
	return true, code
}

const wandboxPrefix = `http://melpon.org/wandbox/permlink/`

func parseWandbox(url string) (demo, error) {
	name := url[len(wandboxPrefix):] + `.cpp`
	resp, err := http.Get(url)
	if err != nil {
		return demo{}, err
	}

	b := resp.Body
	defer b.Close()

	z := html.NewTokenizer(b)

	for {
		tt := z.Next()

		switch {
		case tt == html.ErrorToken:
			return demo{}, errors.New("No code found")
		case tt == html.StartTagToken:
			t := z.Token()

			isScript := t.Data == "script"
			if !isScript {
				continue
			}

			tt = z.Next()
			if tt != html.TextToken {
				continue
			}

			found, c := findWandboxCodeInText(z.Text())
			if found {
				return demo{name, c}, nil
			}

		}
	}
}

func parseURL(url string, demos chan demo, wg *sync.WaitGroup) {
	defer wg.Done()
	switch {
	case strings.Index(url, wandboxPrefix) == 0:
		d, err := parseWandbox(url)
		if err == nil {
			demos <- d
		}
	}
}

func crawlBookmarks(r io.Reader, demos chan demo) {
	z := html.NewTokenizer(r)

	var wg sync.WaitGroup
	defer wg.Wait()

	for {
		tt := z.Next()

		switch {
		case tt == html.ErrorToken:
			return
		case tt == html.StartTagToken:
			t := z.Token()

			isAnchor := t.Data == "a"
			if !isAnchor {
				continue
			}

			ok, url := getHref(t)
			if !ok {
				continue
			}
			wg.Add(1)
			parseURL(url, demos, &wg)
		}
	}
}

func crawlBookmarkFile(fn string, demos chan demo) {
	defer close(demos)

	fi, err := os.Open(fn)
	if err != nil {
		panic(err)
	}

	defer func() {
		if err := fi.Close(); err != nil {
			panic(err)
		}
	}()

	r := bufio.NewReader(fi)
	crawlBookmarks(r, demos)
}

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: parser <bookmarkfile>")
		return
	}

	file := os.Args[1]
	demos := make(chan demo)

	go crawlBookmarkFile(file, demos)

	for d := range demos {
		err := ioutil.WriteFile(d.name, []byte(d.code), 0644)
		if err != nil {
			panic(err)
		}
	}
}
