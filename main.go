package main

import (
	"database/sql"
	"encoding/json"
	"encoding/xml"
	"io/ioutil"
	"net/http"
	"net/url"

	_ "github.com/mattn/go-sqlite3"
	"github.com/urfave/negroni"
	"github.com/yosssi/ace"
)

var db *sql.DB

const CLASSIFY_API = "http://classify.oclc.org/classify2/Classify"

// Page of the web application
type Page struct {
	Name     string
	DBStatus bool
}

// SearchResult of a book search
type SearchResult struct {
	Title  string `xml:"title,attr"`
	Author string `xml:"author,attr"`
	Year   string `xml:"hyr,attr"`
	ID     string `xml:"owi,attr"`
}

// ClassifyResponse from classify2 API
type ClassifyResponse struct {
	Results []SearchResult `xml:"works>work"`
}

// BookResponse from classify2 API
type BookResponse struct {
	BookData struct {
		Title  string `xml:"title,attr"`
		Author string `xml:"author,attr"`
		ID     string `xml:"owi,attr"`
	} `xml:"work"`
	Classification struct {
		MostPopular string `xml:"sfa,attr"`
	} `xml:"recommendations>ddc>mostPopular"`
}

func main() {
	db, _ = sql.Open("sqlite3", "dev.db")

	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		p := Page{Name: "Gopher", DBStatus: false}
		if name := r.FormValue("name"); name != "" {
			p.Name = name
		}

		p.DBStatus = db.Ping() == nil

		template, err := ace.Load("templates/index", "", nil)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		if err := template.Execute(w, p); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	mux.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
		var results []SearchResult
		var err error

		if results, err = search(r.FormValue("search")); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

		encoder := json.NewEncoder(w)
		if err := encoder.Encode(results); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	mux.HandleFunc("/books/add", func(w http.ResponseWriter, r *http.Request) {
		var book BookResponse
		var err error

		if book, err = fetch(r.FormValue("id")); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

		_, err = db.Exec("insert into books  (pk, title, author, id, classification) values (?, ?, ?, ?, ?)",
			nil, book.BookData.Title, book.BookData.Author, book.BookData.ID, book.Classification.MostPopular)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	n := negroni.Classic()
	n.Use(negroni.HandlerFunc(verifyDatabase))
	n.UseHandler(mux)

	n.Run(":8080")
}

func verifyDatabase(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	if err := db.Ping(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	next(w, r)
}

func fetch(id string) (BookResponse, error) {
	var b BookResponse
	body, err := queryClassifyAPI(CLASSIFY_API + "?owi=" + url.QueryEscape(id) + "&summary=true")
	if err != nil {
		return b, err
	}
	err = xml.Unmarshal(body, &b)
	return b, err
}

type SearchResponse struct {
	Results []SearchResult `xml:"works>work"`
}

func search(query string) ([]SearchResult, error) {
	var s SearchResponse
	body, err := queryClassifyAPI(CLASSIFY_API + "?title=" + url.QueryEscape(query) + "&summary=true")
	if err != nil {
		return s.Results, nil
	}
	err = xml.Unmarshal(body, &s)
	return s.Results, nil
}

func queryClassifyAPI(url string) ([]byte, error) {
	var resp *http.Response
	var err error

	if resp, err = http.Get(url); err != nil {
		return []byte{}, err
	}

	defer resp.Body.Close()

	var body []byte
	body, err = ioutil.ReadAll(resp.Body)

	return body, err
}
