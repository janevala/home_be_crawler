package main

import (
	"database/sql"
	"encoding/base64"
	"fmt"
	"runtime"
	"sort"
	"strconv"
	"sync"
	"time"
	"unicode"

	"github.com/google/uuid"
	"github.com/mmcdole/gofeed"

	"github.com/janevala/home_be_crawler/config"
	"github.com/janevala/home_be_crawler/llog"
	_ "github.com/lib/pq"
)

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

func crawl(sites config.SitesConfig, database config.Database) {
	feedParser := gofeed.NewParser()

	var combinedItems []*NewsItem = []*NewsItem{}
	for i := 0; i < len(sites.Sites); i++ {
		feed, err := feedParser.ParseURL(sites.Sites[i].Url)
		if err != nil {
			llog.Err(err)
			panic(err)
		} else {
			if feed.Image != nil {
				for j := 0; j < len(feed.Items); j++ {
					feed.Items[j].Image = feed.Image
				}
			} else {
				for j := 0; j < len(feed.Items); j++ {
					feed.Items[j].Image = &gofeed.Image{
						URL:   "https://github.com/janevala/home_be_crawler.git",
						Title: "N/A",
					}
				}
			}

			var items []*NewsItem = []*NewsItem{}
			for j := 0; j < len(feed.Items); j++ {
				NewsItem := &NewsItem{
					Source:          sites.Sites[i].Title,
					Title:           feed.Items[j].Title,
					Description:     feed.Items[j].Description,
					Content:         feed.Items[j].Content,
					Link:            feed.Items[j].Link,
					Published:       feed.Items[j].Published,
					PublishedParsed: feed.Items[j].PublishedParsed,
					LinkImage:       feed.Items[j].Image.URL,
					Uuid:            uuid.NewString(),
				}

				items = append(items, NewsItem)
			}

			combinedItems = append(combinedItems, items...)
		}
	}

	if len(combinedItems) >= 0 {
		for i := 0; i < len(combinedItems); i++ {
			combinedItems[i].Description = ellipticalTruncate(combinedItems[i].Description, 500)

			// Hashing title to create unique ID, that serves as mechanism to prevent duplicates in DB
			// TODO: consider using getting uuid from Published or PublishedParsed, do more debugging
			uuidString := base64.StdEncoding.EncodeToString([]byte(ellipticalTruncate(combinedItems[i].Title, 40)))
			combinedItems[i].Uuid = uuidString
		}

		connStr := database.Postgres
		db, err := sql.Open("postgres", connStr)

		if err != nil {
			llog.Err(err)
			panic(err)
		}

		if err = db.Ping(); err != nil {
			llog.Err(err)
			panic(err)
		} else {
			llog.Out("Connected to database successfully")
		}

		createTableIfNeeded(db)

		var pkAccumulated int
		for i := 0; i < len(combinedItems); i++ {
			var pk = insertItem(db, combinedItems[i])
			if pk == 0 {
				continue
			}

			if pk <= pkAccumulated {
				llog.Fatal(fmt.Errorf("PK ERROR"))
			} else {
				pkAccumulated = pk
			}
		}

		defer db.Close()

		sort.Slice(combinedItems, func(i, j int) bool {
			return combinedItems[i].PublishedParsed.After(*combinedItems[j].PublishedParsed)
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
		uuid VARCHAR(300) NOT NULL,
		created timestamp DEFAULT NOW(),
		UNIQUE (uuid)
	)`

	_, err := db.Exec(query)
	if err != nil {
		llog.Fatal(err)
	}
}

func insertItem(db *sql.DB, item *NewsItem) int {
	query := "INSERT INTO feed_items (title, description, link, published, published_parsed, source, thumbnail, uuid) VALUES ($1, $2, $3, $4, $5, $6, $7, $8) ON CONFLICT DO NOTHING RETURNING id"

	var pk int
	err := db.QueryRow(query, item.Title, item.Description, item.Link, item.Published, item.PublishedParsed, item.Source, item.LinkImage, item.Uuid).Scan(&pk)

	if err != nil {
		llog.Err(err)
	}

	return pk
}

// https://stackoverflow.com/a/73939904 find better way with AI if needed
func ellipticalTruncate(text string, maxLen int) string {
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

var cfg *config.Config

func init() {
	var err error
	cfg, err = config.LoadConfig("config.json")
	if err != nil {
		llog.Err(err)
		panic(err)
	}
}

func main() {
	llog.Out("Number of CPUs: " + strconv.Itoa(runtime.NumCPU()))
	llog.Out("Number of Goroutines: " + strconv.Itoa(runtime.NumGoroutine()))

	llog.Out("Starting crawler with configuration:")
	llog.Out("Sites: " + fmt.Sprintf("%v", cfg.Sites))
	llog.Out("Database: " + fmt.Sprintf("%v", cfg.Database))

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		defer llog.Out("Crawling completed")

		crawl(cfg.Sites, cfg.Database)
	}()

	wg.Wait()
	llog.Out("All goroutines completed")
}
