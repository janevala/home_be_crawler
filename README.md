# Home BE Crawler

RSS Crawler to be used in conjunction with Home BE API provider. This crawler is meant to be run from Cron, using Docker run. NOTE: Docker should run only once, then quit, then started again after defined period.

# Go notes
```
sudo apt install -y golang
go mod init github.com/janevala/home_be_crawler
make build
```

# Docker notes
```
sudo docker network create home-network

sudo docker build --no-cache -f Dockerfile -t crawler .
sudo docker run --name crawler-host --network home-network -d crawler

sudo docker network connect home-network postgres-host
sudo docker network connect home-network crawler-host
```

# Postgres notes
```
sudo docker run --name postgres-host --network home-network -l com.centurylinklabs.watchtower.enable=false -e POSTGRES_PASSWORD=1234 -p 5432:5432 -d postgres
sudo docker exec -ti postgres-host createdb -U postgres homebedb
sudo docker exec -ti postgres-host psql -U postgres
postgres=# \c homebedb
homebedb=# \q
homebedb=# \dt feed_items
sudo docker exec -t postgres-host pg_dump -U postgres homebedb | gzip -9 > backup.sql.gz
gunzip backup.sql.gz
cat backup.sql | sudo docker exec -i <container_id> psql -U postgres -d homebedb
```
