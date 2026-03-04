#!/bin/bash

ARC=$(uname -m)
ARGUMENTS=("$@")

if [ ${#ARGUMENTS[@]} -eq 0 ]; then
    echo
    echo "Supported languages: en, fi, th, de, pt-BR"
    echo "E.g. to run in Brazilian Portuguese:"
    echo 
    echo "./start.sh pt-BR"
    exit 1
fi

if [ "$ARC" = "aarch64" ]; then
    echo "ARM currently not supported..."
    exit 1

    # if [ ! -f "home_be_crawler_arm64" ]; then
    #     /usr/bin/make clean
    #     /usr/bin/make debug
    #     /usr/bin/make dep
    #     ./home_be_crawler_arm64 "${ARGUMENTS[@]}"
    # else
    #     ./home_be_crawler_arm64 "${ARGUMENTS[@]}"
    # fi
fi

if [ "$ARC" = "x86_64" ]; then
    if [ ! -f "home_be_crawler_amd64" ]; then
        /usr/bin/make clean
        /usr/bin/make debug
        /usr/bin/make dep
        ./home_be_crawler_amd64 "${ARGUMENTS[@]}"
    else
        ./home_be_crawler_amd64 "${ARGUMENTS[@]}"
    fi
fi
