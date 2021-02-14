package gunnhacks

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/dgraph-io/badger"
)

type req struct {
	Qry string
}
type resp struct {
	Query string   "json:query"
	Out   []string "json:out"
}

func serveFiles(w http.ResponseWriter, r *http.Request) {
	// if r.URL.Path != "/" {
	// 	http.Error(w, "404 not found.", http.StatusNotFound)
	// 	return
	// }

	switch r.Method {
	case "POST":
		fmt.Println("reached")
		jsn, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Fatal("Error reading the body", err)
		}
		request := req{}
		err = json.Unmarshal(jsn, &request)
		if err != nil {
			log.Fatal("Decoding err", err)
		}
		log.Printf("recieved")
		res := resp{
			Query: request.Qry,
			Out:   getRelevantURLs(request.Qry),
		}
		fmt.Println(res)
		out, err := json.Marshal(res)
		if err != nil {
			fmt.Fprintf(w, "Error: %s", err)
		}
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")
		w.Write(out)

	default:
		fmt.Fprintf(w, "Sorry, only GET and POST methods are supported.")
	}
}
func getRelevantURLs(qry string) []string {
	db, err := badger.Open(badger.DefaultOptions("./db/"))
	if err != nil {
		log.Fatal(err)
	}
	ret := make([]string, 0)
	err = db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 10
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			k := item.Key()
			if strings.Contains(string(k), qry) {
				// fmt.Println(string(k))

				ret = append(ret, string(k))
			}
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
	db.Close()
	return ret
}
func api() {
	fmt.Println("Now Listening on 8080")
	http.HandleFunc("/", serveFiles)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
func main() {
	go api()
	for true {

	}
}
