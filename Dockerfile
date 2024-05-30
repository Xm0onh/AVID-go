# Dockerfile
FROM golang:1.22.1

WORKDIR /app

COPY . .

COPY eth_transactions.json /

RUN apt-get update && apt-get install -y netcat-openbsd

RUN go build -o main ./cmd/main.go

ENTRYPOINT ["./main"]
