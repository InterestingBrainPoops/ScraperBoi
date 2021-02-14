package gunnhacks

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/dgraph-io/badger"
)

var mutex = &sync.Mutex{}

func ExampleScrape(url string) ([]string, bool) {
	time.Sleep(100 * time.Millisecond)
	// Request the HTML page.
	res, err := http.Get(url)
	if err != nil {
		return nil, true
		// log.Fatal(err)
	}
	if url[0:5] == "https" {
		url = url[8:]
	} else {
		url = url[7:]
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		if res.StatusCode == 404 {
			return nil, false
		}
		return nil, true
		// log.Fatalf("status code error: %d %s", res.StatusCode, res.Status)
	}

	// Load the HTML document
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return nil, true
		// log.Fatal(err)
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
	return ret, false
}
func webboi() {}
func scrape(url string, comms chan map[string]struct{}) {
	out := make(map[string]struct{})
	urls, retry := ExampleScrape(url)
	maxRetries := 5
	numRetries := 0
	for true {
		if retry && numRetries < maxRetries {
			numRetries++
		} else {
			break
		}
	}
	for _, y := range urls {
		out[y] = struct{}{}
	}
	comms <- out
}

func main() {
	f, err := os.OpenFile("testlogfile", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	defer f.Close()
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	log.SetOutput(f)
	// entry:
	entry := "http://en.wikipedia.org/wiki/List_of_lists_of_lists"
	wgsize := 50

	// storage := make(map[string]struct{})
	queue, retry := ExampleScrape(entry)
	for true {
		if retry {
			queue, retry = ExampleScrape(entry)
		} else {
			break
		}
	}
	comms := make(chan map[string]struct{})
	for true {
		f, err := os.OpenFile("testlogfile", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		t0 := time.Now()
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
			if x%1000 == 0 {
				log.Println(time.Now().Sub(t0))
			}
			// fmt.Println(x)
		}

		log.Println("Starting write to db")
		db, err := badger.Open(badger.DefaultOptions("./db/"))
		if err != nil {
			log.Fatal(err)
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
		db.Close()
		queue = make([]string, 0)
		for x := range temp {
			queue = append(queue, x)
		}
		log.Println(len(queue))
		log.Println("Finsihed 1 iteration")
		if err != nil {
			log.Fatal(err)
		}
		f.Close()
	}
}
