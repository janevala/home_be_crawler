FROM golang:1.24
RUN apt update
RUN apt install -y make
ENV PATH="/usr/bin:${PATH}"
WORKDIR /homebe_crawler
COPY . .
RUN rm -f go.mod
RUN rm -f go.sum
CMD ["./start.sh"]
