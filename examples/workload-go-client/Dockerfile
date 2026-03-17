FROM golang:1.26-alpine
WORKDIR /app
COPY . .
RUN go mod tidy
CMD ["go", "run", "main.go"]
