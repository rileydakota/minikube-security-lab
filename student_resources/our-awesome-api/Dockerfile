# Build stage
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod .
COPY main.go .
RUN CGO_ENABLED=0 GOOS=${GOOS} GOARCH=${GOARCH} go build -o api -v


EXPOSE 8080
ENTRYPOINT ["./api"]
