package main

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"sort"
	"strconv"
	"sync"
	"time"
	"unicode"

	"github.com/google/uuid"
	"github.com/mmcdole/gofeed"

	Log "github.com/janevala/home_be/llog"
	_ "github.com/lib/pq"
)

type Database struct {
	Postgres string `json:"postgres"`
}

type Sites struct {
	Time  int    `json:"time"`
	Title string `json:"title"`
	Sites []Site `json:"sites"`
}

type Site struct {
	Uuid  string `json:"uuid"`
	Title string `json:"title"`
	Url   string `json:"url"`
}

type ExtendedItem struct {
	Title           string     `json:"title,omitempty"`
	Description     string     `json:"description,omitempty"`
	Content         string     `json:"content,omitempty"`
	Link            string     `json:"link,omitempty"`
	Updated         string     `json:"updated,omitempty"`
	Published       string     `json:"published,omitempty"`
	PublishedParsed *time.Time `json:"publishedParsed,omitempty"`
	LinkImage       string     `json:"linkImage,omitempty"`
	GUID            string     `json:"guid,omitempty"`
}

func crawlSites(sites Sites, database Database) {
	feedParser := gofeed.NewParser()

	var items []*gofeed.Item = []*gofeed.Item{}
	for i := 0; i < len(sites.Sites); i++ {
		feed, err := feedParser.ParseURL(sites.Sites[i].Url)
		if err != nil {
			Log.Err(err)
			panic(err)
		} else {
			if feed.Image != nil {
				for j := 0; j < len(feed.Items); j++ {
					feed.Items[j].Image = feed.Image
				}
			} else {
				for j := 0; j < len(feed.Items); j++ {
					feed.Items[j].Image = &gofeed.Image{
						URL: "https://www.google.com",
					}
				}
			}

			for j := 0; j < len(feed.Items); j++ {
				feed.Items[j].Updated = sites.Sites[i].Title // reusing for another purpose because lazyness TODO
			}

			items = append(items, feed.Items...)
		}
	}

	for i := 0; i < len(items); i++ {
		items[i].Description = EllipticalTruncate(items[i].Description, 990)
		guidString := base64.StdEncoding.EncodeToString([]byte(EllipticalTruncate(items[i].Title, 50)))
		items[i].GUID = guidString
	}

	connStr := database.Postgres
	db, err := sql.Open("postgres", connStr)

	if err != nil {
		Log.Err(err)
		panic(err)
	}

	if err = db.Ping(); err != nil {
		Log.Err(err)
		panic(err)
	} else {
		Log.Out("Connected to database successfully")
	}

	createTableIfNeeded(db)

	var pkAccumulated int
	for i := 0; i < len(items); i++ {
		var pk = insertItem(db, items[i])
		if pk == 0 {
			continue
		}

		if pk <= pkAccumulated {
			Log.Fatal(fmt.Errorf("PK ERROR"))
		} else {
			pkAccumulated = pk
		}
	}

	defer db.Close()

	var isSorted bool = sort.SliceIsSorted(items, func(i, j int) bool {
		return items[i].PublishedParsed.After(*items[j].PublishedParsed)
	})

	if !isSorted {
		sort.Slice(items, func(i, j int) bool {
			return items[i].PublishedParsed.After(*items[j].PublishedParsed)
		})
	}
}

func createTableIfNeeded(db *sql.DB) {
	query := `CREATE TABLE IF NOT EXISTS feed_items (
		id SERIAL PRIMARY KEY,
		title VARCHAR(200) NOT NULL,
		description VARCHAR(1000) NOT NULL,
		link VARCHAR(500) NOT NULL,
		published timestamp NOT NULL,
		published_parsed timestamp NOT NULL,
		source VARCHAR(200) NOT NULL,
		thumbnail VARCHAR(500),
		guid VARCHAR(250) NOT NULL,
		created timestamp DEFAULT NOW(),
		UNIQUE (guid)
	)`

	_, err := db.Exec(query)
	if err != nil {
		Log.Fatal(err)
	}
}

func insertItem(db *sql.DB, item *gofeed.Item) int {
	query := "INSERT INTO feed_items (title, description, link, published, published_parsed, source, thumbnail, guid) VALUES ($1, $2, $3, $4, $5, $6, $7, $8) ON CONFLICT DO NOTHING RETURNING id"

	var pk int
	err := db.QueryRow(query, item.Title, item.Description, item.Link, item.Published, item.PublishedParsed, item.Updated, item.Image.URL, item.GUID).Scan(&pk)

	if err != nil {
		Log.Err(err)
	}

	return pk
}

// https://stackoverflow.com/a/73939904 find better way with AI if needed
func EllipticalTruncate(text string, maxLen int) string {
	lastSpaceIx := maxLen
	len := 0
	for i, r := range text {
		if unicode.IsSpace(r) {
			lastSpaceIx = i
		}
		len++
		if len > maxLen {
			return text[:lastSpaceIx] + "..."
		}
	}
	return text
}

func main() {
	Log.Out("Number of CPUs: " + strconv.Itoa(runtime.NumCPU()))
	Log.Out("Number of Goroutines: " + strconv.Itoa(runtime.NumGoroutine()))
}

func init() {
	sitesFile, err := os.ReadFile("sites.json")
	if err != nil {
		Log.Err(err)
		panic(err)
	}

	sites := Sites{}
	json.Unmarshal(sitesFile, &sites)
	sitesString, err := json.MarshalIndent(sites, "", "\t")
	if err != nil {
		Log.Err(err)
		panic(err)
	} else {
		sites.Time = int(time.Now().UTC().UnixMilli())
		for i := 0; i < len(sites.Sites); i++ {
			sites.Sites[i].Uuid = uuid.NewString()
		}

		Log.Out(string(sitesString))
	}

	databaseFile, err := os.ReadFile("database.json")
	if err != nil {
		Log.Err(err)
		panic(err)
	}

	database := Database{}
	json.Unmarshal(databaseFile, &database)
	databaseString, err := json.MarshalIndent(database, "", "\t")
	if err != nil {
		Log.Err(err)
		panic(err)
	} else {
		Log.Out(string(databaseString))
	}

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		defer Log.Out("Crawling completed")

		crawlSites(sites, database)
	}()
}
