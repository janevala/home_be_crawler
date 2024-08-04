FROM debian:latest
RUN apt update && apt install -y git golang
RUN apt install -y make
ENV PATH="/usr/bin:${PATH}"
WORKDIR /homebe_crawler
COPY . .
CMD ["./start.sh"]
