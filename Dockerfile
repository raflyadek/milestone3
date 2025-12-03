# Builder stage
FROM golang:1.25-alpine AS builder
RUN apk add --no-cache git ca-certificates
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64
RUN go build -ldflags="-s -w" -o /app/server ./be/app

# Final stage
FROM alpine:3.18
RUN apk add --no-cache ca-certificates tzdata
RUN addgroup -S app && adduser -S -G app app
WORKDIR /app
COPY --from=builder /app/server /app/server
RUN chown app:app /app/server
USER app

ENV PORT=8000
EXPOSE 8000
CMD ["/app/server"]
