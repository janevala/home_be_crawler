// config/config.go
package config

type Config struct {
	Database Database
	Sites    SitesConfig
}

type Database struct {
	Postgres string `json:"postgres"`
}

type SitesConfig struct {
	Time  int
	Title string
	Sites []Site
}

type Site struct {
	Uuid  string
	Title string
	Url   string
}
