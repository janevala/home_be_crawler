FROM debian:latest
RUN apt update && apt install -y git golang
ENV PATH="/usr/bin:${PATH}"
WORKDIR /homebe_crawler
COPY . .
CMD ["make", "run_production"]
