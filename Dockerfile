FROM golang:1.27-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o bot .

FROM alpine:latest
WORKDIR /root/
RUN apk --no-cache add ca-certificates tzdata
COPY --from=builder /app/bot .

CMD ["./bot"]