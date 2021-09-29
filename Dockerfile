FROM golang:1.17 AS builder
WORKDIR /go/src/github.com/crazyfacka/o365toical/
COPY go.mod .
COPY go.sum .
COPY *.go .
RUN NOW=$(date +"%Y-%m-%d_%H%M") && \
go build -ldflags "-X main.BuildDate=$NOW" -a -o app

FROM debian:bullseye
RUN apt update && apt install ca-certificates -y
WORKDIR /app/
COPY --from=builder /go/src/github.com/crazyfacka/o365toical/app .
# You need to provide a valid config.json to /app/config.json
CMD ["./app"]
