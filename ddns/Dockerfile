FROM golang:1.21-alpine AS builder

WORKDIR /source
COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o app .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /source/app .
CMD ["./app"]
