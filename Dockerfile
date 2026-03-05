FROM golang:1.25-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /bin/crescendo ./cmd/crescendo

FROM alpine:3.21
RUN apk add --no-cache ffmpeg ca-certificates \
    && addgroup -S app && adduser -S app -G app
COPY --from=builder /bin/crescendo /bin/crescendo
USER app
EXPOSE 8888
ENTRYPOINT ["/bin/crescendo"]
