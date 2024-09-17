#!/bin/bash

ARC=$(uname -m)

if [ "$ARC" = "aarch64" ]; then
    if [ ! -f "home_be_crawler_arm64" ]; then
        /usr/bin/make clean
        /usr/bin/make build
        /usr/bin/make dep
        /usr/bin/make run
    else
        ./home_be_crawler_arm64
    fi
fi

if [ "$ARC" = "x86_64" ]; then
    if [ ! -f "home_be_crawler_amd64" ]; then
        /usr/bin/make clean
        /usr/bin/make build
        /usr/bin/make dep
        /usr/bin/make run
    else
        ./home_be_crawler_amd64
    fi
fi
