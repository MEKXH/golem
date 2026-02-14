FROM golang:1.26-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/golem ./cmd/golem

FROM alpine:3.22

RUN apk add --no-cache ca-certificates tzdata && adduser -D -h /home/golem golem

USER golem
WORKDIR /home/golem

COPY --from=builder /out/golem /usr/local/bin/golem

EXPOSE 18790
VOLUME ["/home/golem/.golem"]

ENTRYPOINT ["golem", "run"]
