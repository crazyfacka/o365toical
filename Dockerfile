FROM golang:1.16 AS builder
WORKDIR /go/src/github.com/crazyfacka/o365toical/
COPY go.mod .
COPY go.sum .
COPY *.go .
RUN go build -a -o app

FROM debian:buster
RUN apt update && apt install ca-certificates -y
WORKDIR /app/
COPY --from=builder /go/src/github.com/crazyfacka/o365toical/app .
# You need to provide a valid config.json when building
COPY config.json .
CMD ["./app"]
