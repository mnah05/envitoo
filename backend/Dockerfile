FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -o /server ./cmd/server \
    && CGO_ENABLED=0 go build -o /mailer ./cmd/mailer \
    && CGO_ENABLED=0 go build -o /worker ./cmd/worker

FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /server /server
COPY --from=builder /mailer /mailer
COPY --from=builder /worker /worker

EXPOSE 8080

CMD ["/server"]
