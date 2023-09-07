FROM golang:1.20 AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . ./
RUN GOARCH=amd64 GOOS=linux go build -o boomo *.go

# Create lightweight container
FROM alpine:latest

RUN adduser -D -H boomo
WORKDIR /home/boomo
RUN chown -R boomo:boomo /home/boomo
USER boomo

COPY --from=builder /build/boomo /usr/local/bin/boomo

CMD boomo