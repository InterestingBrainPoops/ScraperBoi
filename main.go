package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/dgraph-io/badger"
)

func ExampleScrape(url string) []string {
	// Request the HTML page.
	res, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	if url[0:5] == "https" {
		url = url[8:]
	} else {
		url = url[7:]
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		log.Fatalf("status code error: %d %s", res.StatusCode, res.Status)
	}

	// Load the HTML document
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		log.Fatal(err)
	}
	ret := make([]string, 0)
	// Find the review items
	doc.Find("[href]").Each(func(i int, s *goquery.Selection) {
		// For each item found, get the band and title
		attr, _ := s.Attr("href")
		if len(attr) > 1 {
			if attr[0:2] == "//" {
				ret = append(ret, "http://"+attr[2:])
			} else if attr[0] == '/' && (attr != "/") && strings.ContainsAny(attr, "/") {
				if strings.IndexByte(url, '/') != -1 {
					ret = append(ret, "http://"+url[0:strings.IndexByte(url, '/')]+attr)
				}
			}
		}
	})
	return ret
}
func webboi() {}
func scrape(url string, comms chan map[string]struct{}) {
	out := make(map[string]struct{})
	urls := ExampleScrape(url)
	for _, y := range urls {
		out[y] = struct{}{}
	}
	comms <- out
}

func main() {
	db, err := badger.Open(badger.DefaultOptions("./"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	// entry:
	entry := "http://en.wikipedia.org/wiki/List_of_lists_of_lists"
	wgsize := 300
	// storage := make(map[string]struct{})
	queue := ExampleScrape(entry)
	comms := make(chan map[string]struct{})
	for true {
		temp := make(map[string]struct{})
		for x := 0; x < wgsize; x++ {
			go scrape(queue[x], comms)
		}
		for x := range queue {
			thing := <-comms
			for point := range thing {
				temp[point] = struct{}{}
			}
			if x+wgsize < len(queue) {
				go scrape(queue[x+wgsize], comms)
			}
		}
		for _, x := range queue {
			txn := db.NewTransaction(true)
			if err := txn.Set([]byte(x), []byte(fmt.Sprintf("%v", struct{}{}))); err == badger.ErrTxnTooBig {
				_ = txn.Commit()
				txn = db.NewTransaction(true)
				_ = txn.Set([]byte(x), []byte(fmt.Sprintf("%v", struct{}{})))
			}
			_ = txn.Commit()
			if err != nil {
				log.Fatal(err)
			}
		}
		queue = make([]string, 0)
		for x := range temp {
			queue = append(queue, x)
		}
	}
}
