FROM golang:alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go mod tidy
RUN CGO_ENABLED=0 go build -o /bin/user-service ./services/user-service

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /bin/user-service /bin/user-service
EXPOSE 50053
CMD ["/bin/user-service"]
