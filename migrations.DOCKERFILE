FROM golang:1.26.3-alpine AS builder
RUN apk add --no-cache git
RUN go install github.com/pressly/goose/v3/cmd/goose@latest

FROM alpine:latest
RUN apk add --no-cache ca-certificates
COPY --from=builder /go/bin/goose /usr/local/bin/goose
WORKDIR /app
ENTRYPOINT ["goose"]
CMD ["up"]
