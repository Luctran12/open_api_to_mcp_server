FROM golang:1.24.5-alpine AS builder
RUN apk add --no-cache git ca-certificates build-base

WORKDIR /app

# copy mod và lấy dependency
COPY go.mod go.sum ./
RUN go mod download

# copy toàn bộ code vào
COPY . .

# Build main API binary (Linux)
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o open-api-to-mcp-server

# Runtime image có GO để build exe sau này
FROM golang:1.24.5-alpine

RUN apk add --no-cache git ca-certificates build-base

WORKDIR /app

COPY . .

# copy binary server đã build
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /app/open-api-to-mcp-server /open-api-to-mcp-server


EXPOSE 8081
CMD ["/open-api-to-mcp-server"]
