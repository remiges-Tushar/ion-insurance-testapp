FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY main.go .
RUN go mod init ion-service && go mod tidy && CGO_ENABLED=0 GOOS=linux go build -o /ion-server .

FROM alpine:3.20
RUN apk --no-cache add ca-certificates
COPY --from=builder /ion-server /ion-server
EXPOSE 8090
CMD ["/ion-server"]
