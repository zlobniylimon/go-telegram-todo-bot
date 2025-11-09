FROM golang:1.24.2 as builder
ARG CGO_ENABLED=0
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN go build -o go-telegram-todo

FROM scratch
COPY --from=builder /app/go-telegram-todo /go-telegram-todo
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group
ENTRYPOINT ["/go-telegram-todo"]
