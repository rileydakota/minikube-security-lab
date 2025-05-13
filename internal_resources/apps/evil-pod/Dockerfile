# Build stage
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod .
COPY go.sum .
COPY main.go .

RUN CGO_ENABLED=0 go build -o evilpod -v

# Download LetsEncrypt root certs
RUN wget https://letsencrypt.org/certs/isrgrootx1.pem -O /tmp/isrgrootx1.pem && \
    wget https://letsencrypt.org/certs/lets-encrypt-r3.pem -O /tmp/lets-encrypt-r3.pem

# Final stage
FROM scratch
COPY --from=builder /app/evilpod /evilpod
COPY --from=builder /tmp/isrgrootx1.pem /etc/ssl/certs/
COPY --from=builder /tmp/lets-encrypt-r3.pem /etc/ssl/certs/
ENTRYPOINT ["/evilpod"]