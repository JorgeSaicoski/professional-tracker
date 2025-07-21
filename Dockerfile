FROM golang:latest

WORKDIR /app

COPY go.* ./
RUN go mod download

COPY . .

RUN go build -o main ./cmd/server/main.go

EXPOSE 8002

CMD ["./main"]