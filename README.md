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
sudo docker network create home-network

sudo docker run --name postgres-host --network home-network -l com.centurylinklabs.watchtower.enable=false -e POSTGRES_PASSWORD=1234 -p 5432:5432 -d postgres
sudo docker exec -ti postgres-host createdb -U postgres homebedb
sudo docker exec -ti postgres-host psql -U postgres
postgres=# \c homebedb
homebedb=# \q
homebedb=# \dt feed_items

sudo docker build --no-cache -f Dockerfile -t crawler .
sudo docker run --name crawler-host --network home-network -d crawler

sudo docker network connect home-network postgres-host
sudo docker network connect home-network crawler-host
```

TODO
- FIX WITH CUSTOM OBJECT feed.Items[j].Updated = sites.Sites[i].Title // reusing for another purpose because lazyness TODO

