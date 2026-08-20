# 1. Етап збірки
FROM golang:1.22-alpine AS builder
WORKDIR /app

COPY go.mod ./
# Якщо у вас є go.sum, додайте його копіювання
COPY go.sum* ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /web-scanner main.go

# 2. Фінальний образ
FROM alpine:3.20
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app

COPY --from=builder /web-scanner .
COPY signatures.json .
COPY ui ./ui

EXPOSE 8080
ENTRYPOINT ["./web-scanner"]
