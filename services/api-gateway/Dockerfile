FROM golang:alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go mod tidy
RUN CGO_ENABLED=0 go build -o /bin/api-gateway ./services/api-gateway

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /bin/api-gateway /bin/api-gateway
EXPOSE 8080
CMD ["/bin/api-gateway"]
