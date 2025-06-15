FROM golang:1.24.0-alpine AS builder

WORKDIR /app

RUN apk add --no-cache gcc musl-dev git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o goods-service ./cmd/main.go

FROM alpine:3.19

WORKDIR /app

COPY --from=builder /app/goods-service .
COPY --from=builder /app/migrations/postgres/*.sql ./migrations/postgres/
COPY --from=builder /app/docs ./docs

RUN apk --no-cache add tzdata

EXPOSE 8080

CMD ["./goods-service"]
