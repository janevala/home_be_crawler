FROM golang:1.24
RUN apt update
RUN go version
RUN apt install -y make
ENV PATH="/usr/bin:${PATH}"
WORKDIR /homebe_crawler
COPY . .
RUN rm -f go.mod
RUN rm -f go.sum
CMD ["./start.sh"]

# FROM debian:stable-slim AS builder
# RUN apt update && apt install -y git golang
