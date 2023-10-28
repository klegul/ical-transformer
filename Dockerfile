FROM golang:1.21-alpine
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /ical-transformer
EXPOSE 8080
CMD ["/ical-transformer", "--config=/transformer.toml", "--server"]