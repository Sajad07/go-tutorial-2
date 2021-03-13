package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	log "github.com/llimllib/loglevel"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

type Link struct {
	URL   string
	Text  string
	Depth int
}

type HttpError struct {
	Original string
}

func (self Link) String() string {
	spacer := strings.Repeat("\t", self.Depth)
	return fmt.Sprintf("%s%s (%d) - %s", spacer, self.Text, self.Depth, self.URL)
}
func (self Link) Valid() bool {
	if self.Depth >= MaxDepth {
		return false
	}
	if len(self.Text) == 0 {
		return false
	}
	if len(self.URL) == 0 || strings.Contains(strings.ToLower(self.URL), "javascript") {
		return false
	}
	return true
}
func (self HttpError) Error() string {
	return self.Original
}

var MaxDepth = 2

func LinkReader(r *http.Response, depth int) []Link {
	page := html.NewTokenizer(r.Body)
	links := []Link{}

	var start *html.Token
	var text string

	for {
		_ = page.Next()
		token := page.Token()
		if token.Type == html.ErrorToken {
			break
		}
		if token.DataAtom == atom.A {
			switch token.Type {
			case html.StartTagToken:
				if len(token.Attr) > 0 {
					start = &token
				}
			case html.EndTagToken:
				if start == nil {
					log.Warnf("Link End found without Start: %s", text)
					continue
				}
				link := NewLink(*start, text, depth)
				if link.Valid() {
					links = append(links, link)
					log.Debugf("Link found %v", link)
				}
				start = nil
				text = ""
			}
		}
	}
	log.Debug(links)
	return links
}
func NewLink(tag html.Token, text string, depth int) Link {
	link := Link{Text: strings.TrimSpace(text), Depth: depth}
	for i := range tag.Attr {
		if tag.Attr[i].Key == "href" {
			link.URL = strings.TrimSpace(tag.Attr[i].Val)
		}
	}
	return link
}
func recurDownloader(url string, depth int) {
	page, err := downloader(url)
	if err != nil {
		log.Error(err)
	}
	links := LinkReader(page, depth)
	for _, link := range links {
		fmt.Println(link)
		if depth+1 < MaxDepth {
			recurDownloader(link.URL, depth)
		}
	}
}
func downloader(url string) (r *http.Response, err error) {
	log.Debug("Downloading %s", url)
	r, err = http.Get(url)
	if err != nil {
		log.Debugf("Error: %s", err)
		return
	}
	if r.StatusCode > 299 {
		err = HttpError{fmt.Sprintf("Error (%d): %s", r.StatusCode, url)}
		log.Debug(err)
		return
	}
	return
}
func main() {
	log.SetPriorityString("info")
	log.SetPrefix("crawler")
	log.Debug(os.Args)
	if len(os.Args) < 2 {
		log.Fatalln("Missing Url arg")
	}
	recurDownloader(os.Args[1], 0)
}
