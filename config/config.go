// config/config.go
package config

type Config struct {
	Database Database
	Sites    SitesConfig
	Ollama   Ollama
}

type Database struct {
	Postgres string `json:"postgres"`
}

type Ollama struct {
	Host  string
	Port  string
	Model string // "mistral:7b", "qwen2.5-coder:14b"
}

type SitesConfig struct {
	Title string
	Sites []Site
}

type Site struct {
	Title string
	Url   string
}
