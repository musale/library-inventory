package main

import (
	"database/sql"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"net/url"

	_ "github.com/mattn/go-sqlite3"
)

const CLASSIFY_API = "http://classify.oclc.org/classify2/Classify?title="

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

func main() {
	templates := template.Must(template.ParseFiles("templates/index.html"))

	db, _ := sql.Open("sqlite3", "dev.db")

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		p := Page{Name: "Gopher", DBStatus: false}
		if name := r.FormValue("name"); name != "" {
			p.Name = name
		}

		p.DBStatus = db.Ping() == nil

		if err := templates.ExecuteTemplate(w, "index.html", p); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	http.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
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

	fmt.Println(http.ListenAndServe(":8080", nil))
}

func search(query string) ([]SearchResult, error) {
	var resp *http.Response
	var err error
	var c ClassifyResponse

	if resp, err = http.Get(CLASSIFY_API + url.QueryEscape(query) + "&summary=true"); err != nil {
		return c.Results, err
	}

	defer resp.Body.Close()

	var body []byte
	if body, err = ioutil.ReadAll(resp.Body); err != nil {
		return c.Results, err
	}

	if err = xml.Unmarshal(body, &c); err != nil {
		return c.Results, err
	}

	return c.Results, nil
}
