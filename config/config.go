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
	Title string
	Sites []Site
}

type Site struct {
	Title string
	Url   string
}
