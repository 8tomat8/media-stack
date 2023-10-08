FROM golang:1.21-alpine AS builder

WORKDIR /go/src/github.com/monitor
COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o monitor .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /go/src/github.com/monitor/monitor .
CMD ["./monitor"]
