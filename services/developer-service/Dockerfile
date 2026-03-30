FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /bin/developer-service ./services/developer-service

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /bin/developer-service /bin/developer-service
EXPOSE 50051
CMD ["/bin/developer-service"]
