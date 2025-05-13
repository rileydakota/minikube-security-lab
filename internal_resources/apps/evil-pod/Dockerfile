# Build stage
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod .
COPY go.sum .
COPY main.go .
RUN CGO_ENABLED=0 GOOS=linux go build -o evilpod

# Final stage
FROM scratch
COPY --from=builder /app/evilpod /evilpod
ENTRYPOINT ["/evilpod"]