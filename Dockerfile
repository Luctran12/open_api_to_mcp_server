FROM golang:1.24.5-alpine AS builder
RUN apk add --no-cache git ca-certificates

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o open-api-to-mcp-server

FROM scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /app/open-api-to-mcp-server /open-api-to-mcp-server
EXPOSE 8081
CMD ["/open-api-to-mcp-server"]