# Build stage
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod .
COPY go.sum .
COPY main.go .
ARG GOOS=linux
ARG GOARCH=amd64
RUN CGO_ENABLED=0 GOOS=${GOOS} GOARCH=${GOARCH} go build -o evilpod -v

# Final stage
FROM scratch
COPY --from=builder /app/evilpod /evilpod
ENTRYPOINT ["/evilpod"]