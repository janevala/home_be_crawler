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

# Docker notes
```
sudo docker run --name postgres-container -e POSTGRES_PASSWORD=1234 -p 5432:5432 -d postgres
sudo docker exec -ti postgres-container createdb -U postgres homebedb
sudo docker exec -ti postgres-container psql -U postgres
postgres=# \c homebedb
homebedb=# \q

homebedb=# \dt feed_items

sudo docker run -d home-web-crawler
sudo docker build --no-cache -f Dockerfile -t home-web-crawler .
```
