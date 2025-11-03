FROM golang:1.24.5-alpine AS base
RUN apk add --no-cache git ca-certificates build-base

WORKDIR /app

# copy mod và lấy dependency
COPY go.mod go.sum ./
RUN go mod download

# copy toàn bộ code vào
COPY . .

# Build main API binary (Linux)
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o server ./cmd/server

# Runtime image có GO để build exe sau này
FROM golang:1.24.5-alpine

RUN apk add --no-cache git ca-certificates build-base

WORKDIR /app

# copy binary server đã build
COPY --from=base /app/server .


EXPOSE 8081
CMD ["./server"]
