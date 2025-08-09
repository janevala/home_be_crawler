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

type NewsItem struct {
	Source          string     `json:"source,omitempty"`
	Title           string     `json:"title,omitempty"`
	Description     string     `json:"description,omitempty"`
	Content         string     `json:"content,omitempty"`
	Link            string     `json:"link,omitempty"`
	Published       string     `json:"published,omitempty"`
	PublishedParsed *time.Time `json:"publishedParsed,omitempty"`
	LinkImage       string     `json:"linkImage,omitempty"`
	Uuid            string     `json:"uuid,omitempty"`
}

func crawlSites(sites Sites, database Database) {
	feedParser := gofeed.NewParser()

	var combinedItems []*NewsItem = []*NewsItem{}
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
						URL:   "https://github.com/janevala/home_be_crawler",
						Title: "N/A",
					}
				}
			}

			var items []*NewsItem = []*NewsItem{}
			for j := 0; j < len(feed.Items); j++ {
				items[j].Source = sites.Sites[i].Title
				items[j].Title = feed.Items[j].Title
				items[j].Description = feed.Items[j].Description
				items[j].Content = feed.Items[j].Content
				items[j].Link = feed.Items[j].Link
				items[j].Published = feed.Items[j].Published
				items[j].PublishedParsed = feed.Items[j].PublishedParsed
				items[j].LinkImage = feed.Items[j].Image.URL
				items[j].Uuid = feed.Items[j].GUID
				// items[j].Uuid = uuid.NewString()

			}

			combinedItems = append(combinedItems, items...)
		}
	}

	if len(combinedItems) == 0 {
		Log.Out("No items found, exiting...")
		return
	}

	for i := 0; i < len(combinedItems); i++ {
		combinedItems[i].Description = EllipticalTruncate(combinedItems[i].Description, 500)

		// Hashing title to create unique ID, that serves as mechanism to prevent duplicates in DB
		guidString := base64.StdEncoding.EncodeToString([]byte(EllipticalTruncate(combinedItems[i].Title, 40)))
		combinedItems[i].Uuid = guidString
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
	for i := 0; i < len(combinedItems); i++ {
		var pk = insertItem(db, combinedItems[i])
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

	sort.Slice(combinedItems, func(i, j int) bool {
		return combinedItems[i].PublishedParsed.After(*combinedItems[j].PublishedParsed)
	})
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

func insertItem(db *sql.DB, item *NewsItem) int {
	query := "INSERT INTO feed_items (title, description, link, published, published_parsed, source, thumbnail, guid) VALUES ($1, $2, $3, $4, $5, $6, $7, $8) ON CONFLICT DO NOTHING RETURNING id"

	var pk int
	err := db.QueryRow(query, item.Title, item.Description, item.Link, item.Published, item.PublishedParsed, item.Source, item.LinkImage, item.Uuid).Scan(&pk)

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
