# Home BE Crawler

RSS Crawler to be used in conjunction with Home BE API provider. This crawler is meant to be run from Cron, using Docker run. NOTE: Docker should run only once, then quit, then started again after defined period.

# Go notes
```
sudo apt install -y golang
go mod init github.com/janevala/home_be_crawler
go mod tidy
go get github.com/mmcdole/gofeed
go get github.com/google/uuid
go get github.com/lib/pq

```
