FROM golang:1.21-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o ws-gateway ./cmd/ws-gateway

FROM alpine:latest

WORKDIR /app

RUN apk --no-cache add ca-certificates tzdata

COPY --from=builder /app/ws-gateway .
COPY --from=builder /app/config.yaml .

ENV TZ=UTC

EXPOSE 8080

CMD ["./ws-gateway"] 