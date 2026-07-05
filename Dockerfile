# --- build stage ---
FROM golang:1.26-alpine AS build

WORKDIR /src

# Кэшируем зависимости отдельно от исходников.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
# Статичный бинарник без CGO; календарь встроен через go:embed.
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/bot ./cmd/bot

# --- runtime stage ---
FROM alpine:3.20

# ca-certificates — для HTTPS к api.telegram.org; tzdata — для Europe/Moscow.
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app
COPY --from=build /out/bot /app/bot

ENV STATE_DIR=/data
VOLUME /data

ENTRYPOINT ["/app/bot"]
