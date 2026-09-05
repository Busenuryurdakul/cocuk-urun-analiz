# Miyuna API — Render Docker (root)
FROM golang:1.23-bookworm AS builder
WORKDIR /src
COPY apps/api/go.mod apps/api/go.sum ./apps/api/
WORKDIR /src/apps/api
RUN go mod download
COPY apps/api/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /server ./cmd/server

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=builder /server ./server
ENV API_HOST=0.0.0.0
EXPOSE 8080
CMD ["./server"]
