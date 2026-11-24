FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /usr/local/bin/gh-repro .

FROM alpine:3.21
RUN apk add --no-cache github-cli git ca-certificates
COPY --from=builder /usr/local/bin/gh-repro /usr/local/bin/gh-repro
ENTRYPOINT ["gh-repro", "--mcp"]
