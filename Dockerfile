FROM golang:1.22-alpine AS build
WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o /out/mbti-site ./cmd/server

FROM alpine:3.20
WORKDIR /app

RUN adduser -D -H -u 10001 appuser \
    && mkdir -p /app/data \
    && chown -R appuser:appuser /app

COPY --from=build /out/mbti-site /usr/local/bin/mbti-site
COPY --from=build /app/web/static ./web/static

ENV HOST=0.0.0.0
ENV PORT=8080
EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD port="${PORT#:}"; wget -qO- "http://127.0.0.1:${port}/healthz" || exit 1

USER appuser
CMD ["mbti-site"]
