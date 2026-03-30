FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /bin/token-service ./services/token-service

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /bin/token-service /bin/token-service
EXPOSE 50052
CMD ["/bin/token-service"]
