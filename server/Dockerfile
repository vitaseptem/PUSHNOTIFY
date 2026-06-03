# ---- build stage ----
FROM golang:1.22-alpine AS build
WORKDIR /src

# Cache dependencies first.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
# Build a static binary; migrations are embedded via go:embed.
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/pushnotify ./cmd/pushnotify

# ---- runtime stage ----
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata && adduser -D -u 10001 app
WORKDIR /app
COPY --from=build /out/pushnotify /app/pushnotify
USER app
EXPOSE 8080
ENTRYPOINT ["/app/pushnotify"]
