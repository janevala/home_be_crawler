package main

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/google/uuid"
	"github.com/mmcdole/gofeed"
	"github.com/rifaideen/talkative"

	B "github.com/janevala/home_be_crawler/build"
	Conf "github.com/janevala/home_be_crawler/config"
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

type QuestionItem struct {
	Question string `json:"question,omitempty"`
}

type AnswerItem struct {
	Answer string `json:"answer,omitempty"`
}

type Lang struct {
	Code string
	Name string
}

func queryAI(q QuestionItem, ollama Conf.Ollama, language Lang) AnswerItem {
	client, err := talkative.New("http://" + ollama.Host + ":" + ollama.Port)

	if err != nil {
		panic(err)
	}

	var response string
	callback := func(cr string, err error) {
		if err != nil {
			return
		}
		var r talkative.CompletionResponse

		if err = json.Unmarshal([]byte(cr), &r); err != nil {
			return
		}
		response += r.Response
	}

	// prompt := "You are language specialist. You generate text in " + language.Name + ". Just text itself, dont describe what you are doing in answer."
	prompt := "You are a professional " + language.Name + " (" + language.Code + ") translator. Your goal is to accurately convey the meaning and nuances of the original text while adhering to " + language.Name + " grammar, vocabulary, and cultural sensitivities. Produce only the " + language.Name + " translation, without any additional explanations or commentary. Please translate the following text into " + language.Name + ":"

	preparsedQuestion := parseQuestion(q)
	if preparsedQuestion.Question == "" {
		// bypass AI
		return AnswerItem{Answer: q.Question}
	}

	message := &talkative.CompletionMessage{
		Prompt: preparsedQuestion.Question,
		CompletionParams: &talkative.CompletionParams{
			System: prompt,
		},
	}

	model := ollama.Model
	done, err := client.PlainCompletion(model, callback, message)

	if err != nil {
		panic(err)
	}

	<-done

	answerItem := AnswerItem{Answer: response}

	return answerItem
}

func parseQuestion(q QuestionItem) QuestionItem {
	lines := strings.Split(q.Question, "\n")
	var filteredLines []string
	for _, line := range lines {
		if !strings.Contains(line, "http") && !strings.Contains(line, "Comments") {
			filteredLines = append(filteredLines, strings.TrimSpace(line))
		}
	}

	question := strings.Join(filteredLines, "\n")
	return QuestionItem{Question: question}
}

func translate(ollama Conf.Ollama, database Conf.Database, language Lang, limit int) {
	connStr := database.Postgres
	db, err := sql.Open("postgres", connStr)

	if err != nil {
		B.LogErr(err)
		return
	}

	if err = db.Ping(); err != nil {
		B.LogErr(err)
		return
	}

	rows, err := db.Query(`SELECT id, title, description, published, published_parsed FROM feed_items ORDER BY published_parsed DESC LIMIT $1`, limit)

	if err != nil {
		B.LogErr(err)
		return
	}

	defer rows.Close()

	for rows.Next() {
		var id int
		var title string
		var description string
		var published string
		var publishedParsed time.Time

		if err := rows.Scan(&id, &title, &description, &published, &publishedParsed); err != nil {
			B.LogErr(err)
			continue
		}

		exists := false
		err = db.QueryRow(
			`SELECT EXISTS (
				SELECT 1
				FROM feed_translations
				WHERE item_id = $1 AND language = $2
				)`, id, language.Code).Scan(&exists)

		if err != nil {
			B.LogErr(err)
			continue
		}

		if !exists {
			questionTitle := QuestionItem{
				Question: title,
			}

			answerTitle := queryAI(questionTitle, ollama, language)

			answerDescription := AnswerItem{}

			if description == "" {
				answerDescription = AnswerItem{Answer: answerTitle.Answer}
			} else {
				questionDescription := QuestionItem{
					Question: description,
				}

				answerDescription = queryAI(questionDescription, ollama, language)
			}

			insertTranslation(db, id, publishedParsed, language.Code, ollama.Model, ellipticalTruncate(answerTitle.Answer, 450), ellipticalTruncate(answerDescription.Answer, 950))
		}
	}

	if err = rows.Err(); err != nil {
		B.LogErr(err)
	}

	defer db.Close()
}

func crawl(sites Conf.SitesConfig, database Conf.Database) {
	feedParser := gofeed.NewParser()

	var combinedItems []*NewsItem = []*NewsItem{}
	for i := 0; i < len(sites.Sites); i++ {
		feed, err := feedParser.ParseURL(sites.Sites[i].Url)
		if err != nil {
			B.LogErr(err)
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
					Title:           strings.TrimSpace(feed.Items[j].Title),
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

	if len(combinedItems) > 0 {
		for i := 0; i < len(combinedItems); i++ {
			combinedItems[i].Description = ellipticalTruncate(combinedItems[i].Description, 950)

			// Hashing title to create unique ID, that serves as mechanism to prevent duplicates in DB
			uuidString := base64.StdEncoding.EncodeToString([]byte(ellipticalTruncate(combinedItems[i].Title, 35)))
			combinedItems[i].Uuid = uuidString
		}

		sort.Slice(combinedItems, func(i, j int) bool {
			return combinedItems[i].PublishedParsed.After(*combinedItems[j].PublishedParsed)
		})

		connStr := database.Postgres
		db, err := sql.Open("postgres", connStr)

		if err != nil {
			B.LogErr(err)
		}

		if err = db.Ping(); err != nil {
			B.LogErr(err)
		} else {
			B.LogOut("Connected to database successfully")
		}

		var pkAccumulated int
		for i := 0; i < len(combinedItems); i++ {
			var pk = insertItem(db, combinedItems[i])
			if pk == 0 {
				continue
			}

			if pk <= pkAccumulated {
				B.LogOut("PK ERROR")
			} else {
				pkAccumulated = pk
			}
		}

		defer db.Close()
	}
}

func createTablesIfNeeded(database Conf.Database) {
	connStr := database.Postgres
	db, err := sql.Open("postgres", connStr)

	if err != nil {
		B.LogErr(err)
	}

	if err = db.Ping(); err != nil {
		B.LogErr(err)
	} else {
		B.LogOut("Connected to database successfully")
	}

	feedItems := `CREATE TABLE IF NOT EXISTS feed_items (
		id SERIAL PRIMARY KEY,
		title VARCHAR(500) NOT NULL,
		description VARCHAR(1000) NOT NULL,
		link VARCHAR(500) NOT NULL,
		published timestamp NOT NULL,
		published_parsed timestamp NOT NULL,
		source VARCHAR(300) NOT NULL,
		thumbnail VARCHAR(500),
		uuid VARCHAR(300) NOT NULL,
		created timestamp DEFAULT NOW(),
		UNIQUE (uuid)
	)`

	_, err = db.Exec(feedItems)
	if err != nil {
		B.LogErr(err)
	}

	feedTranslations := `CREATE TABLE IF NOT EXISTS feed_translations (
		id SERIAL PRIMARY KEY,
		item_id INT NOT NULL,
		published_parsed timestamp NOT NULL,
		language VARCHAR(10) NOT NULL,
		title VARCHAR(500) NOT NULL,
		description VARCHAR(1000) NOT NULL,
		llm VARCHAR(100) NOT NULL,
		created timestamp DEFAULT NOW(),
		UNIQUE (item_id, language),
		FOREIGN KEY (item_id) REFERENCES feed_items(id) ON DELETE CASCADE
	)`

	_, err = db.Exec(feedTranslations)
	if err != nil {
		B.LogErr(err)
	}

	defer db.Close()
}

func insertItem(db *sql.DB, item *NewsItem) int {
	query := "INSERT INTO feed_items (title, description, llm, link, published, published_parsed, source, thumbnail, uuid) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) ON CONFLICT DO NOTHING RETURNING id"

	var pk int
	err := db.QueryRow(query, item.Title, item.Description, item.Link, item.Published, item.PublishedParsed, item.Source, item.LinkImage, item.Uuid).Scan(&pk)

	if err != nil {
		B.LogOut(err.Error() + " - duplicate uuid: " + item.Uuid)
	} else {
		B.LogOut("Item (pk: " + strconv.Itoa(pk) + "): " + ellipticalTruncate(item.Title, 35))
	}

	return pk
}

func insertTranslation(db *sql.DB, itemID int, publishedParsed time.Time, code string, llm string, title string, description string) {
	query := "INSERT INTO feed_translations (item_id, published_parsed, language, llm, title, description) VALUES ($1, $2, $3, $4, $5, $6) ON CONFLICT DO NOTHING"

	_, err := db.Exec(query, itemID, publishedParsed, code, llm, title, description)

	if err != nil {
		B.LogErr(err)
	} else {
		B.LogOut("Translation for item_id: " + strconv.Itoa(itemID) + " code: " + code + " - " + ellipticalTruncate(title, 95))
	}
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

var cfg *Conf.Config

func init() {
	var err error
	cfg, err = Conf.LoadConfig("config.json")
	if err != nil {
		B.LogErr(err)
	}
}

func main() {
	logger := log.New(log.Writer(), "[LOG] ", log.LstdFlags)
	B.SetLogger(logger)

	language := Lang{}
	language.Code = "en"
	language.Name = "English"
	limit := 10

	fmt.Println("All arguments:", os.Args)
	arguments := os.Args[1:]

	if len(arguments) == 2 {
		language.Code = arguments[0]
		switch language.Code {
		case "fi":
			language.Name = "Finnish"
		case "th":
			language.Name = "Thai"
		case "de":
			language.Name = "German"
		case "pt-BR":
			language.Name = "Brazilian Portuguese"
		case "en":
		default:
			language.Code = "en"
			language.Name = "English"
		}
		limitArg := arguments[1]
		parsedLimit, err := strconv.Atoi(limitArg)
		if err != nil {
			B.LogErr(fmt.Errorf("invalid limit argument: %v", err))

			limit = 10
			B.LogOut("Using default limit: " + strconv.Itoa(limit))
		} else {
			limit = parsedLimit
		}
	} else {
		language.Code = "en"
		language.Name = "English"
		limit = 10
		B.LogOut("Using default language: " + language.Name)
		B.LogOut("Using default limit: " + strconv.Itoa(limit))
	}

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		defer B.LogOut("Crawling completed")

		createTablesIfNeeded(cfg.Database)
		// crawl(cfg.Sites, cfg.Database)
		if language.Code != "en" && limit > 0 && limit <= 1000 {
			B.LogOut("Starting translation to " + language.Name)
			translate(cfg.Ollama, cfg.Database, language, limit)
		}
	}()

	wg.Wait()
	B.LogOut("All goroutines completed")
}
