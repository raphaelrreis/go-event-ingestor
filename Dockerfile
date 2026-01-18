FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o ingestor cmd/ingestor/main.go

FROM alpine:latest

WORKDIR /root/

COPY --from=builder /app/ingestor .

EXPOSE 8080

CMD ["./ingestor"]
