FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o wireframez ./cmd/proxy

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/wireframez .
EXPOSE 8080
CMD ["./wireframez"]