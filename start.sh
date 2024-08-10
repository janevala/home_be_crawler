#!/bin/bash

ARC=$(uname -m)

if [ "$ARC" = "aarch64" ]; then
    if [ ! -f "home_be_crawler_arm64" ]; then
        /usr/bin/make clean 
        /usr/bin/go mod init github.com/janevala/home_be_crawler
        /usr/bin/go mod tidy
        /usr/bin/go get github.com/mmcdole/gofeed
        /usr/bin/go get github.com/google/uuid
        /usr/bin/go get github.com/lib/pq

        /usr/bin/make build_and_run
    else
        ./home_be_crawler_arm64
    fi
fi

if [ "$ARC" = "x86_64" ]; then
    if [ ! -f "home_be_crawler_amd64" ]; then
        /usr/bin/make clean 
        /usr/bin/go mod init github.com/janevala/home_be_crawler
        /usr/bin/go mod tidy
        /usr/bin/go get github.com/mmcdole/gofeed
        /usr/bin/go get github.com/google/uuid
        /usr/bin/go get github.com/lib/pq

        /usr/bin/make build_and_run
    else
        ./home_be_crawler_amd64
    fi
fi
