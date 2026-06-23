FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY main.go .
RUN go mod init cs-mock && CGO_ENABLED=0 GOOS=linux go build -o /cs-mock .

FROM alpine:3.20
RUN apk --no-cache add ca-certificates
COPY --from=builder /cs-mock /cs-mock
EXPOSE 9090
CMD ["/cs-mock"]
