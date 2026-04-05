FROM golang:1.24-alpine AS builder

RUN apk add --no-cache gcc musl-dev pkgconf librdkafka-dev

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG SERVICE
RUN go build -tags musl -o /bin/service ./cmd/${SERVICE}

FROM alpine:3.21

RUN apk add --no-cache librdkafka

COPY --from=builder /bin/service /bin/service

ENTRYPOINT ["/bin/service"]
