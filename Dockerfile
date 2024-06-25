FROM alpine:3.20.1 AS certs

RUN apk add ca-certificates

FROM golang:1.22 AS builder

ARG VERSION

WORKDIR /build

COPY . /build/
RUN go mod download
RUN CGO_ENABLED=0 go build -ldflags="-X main.version=${VERSION}" -a -o crunchy-users main.go

FROM alpine:3.20.1

COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /build/crunchy-users /crunchy-users

ENTRYPOINT [ "/crunchy-users" ]
