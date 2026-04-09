FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /w2w-verification .

FROM alpine:3.19
COPY --from=builder /w2w-verification /usr/local/bin/
VOLUME /data
EXPOSE 8080
ENTRYPOINT ["w2w-verification", "-addr", ":8080", "-db", "/data/w2w.db"]
