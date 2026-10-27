# syntax=docker/dockerfile:1

FROM node:24-alpine AS web-deps
WORKDIR /app/web
COPY web/package.json web/package-lock.json ./
RUN npm ci

FROM node:24-alpine AS web-build
WORKDIR /app/web
COPY --from=web-deps /app/web/node_modules ./node_modules
COPY web/ ./
RUN npm run build

FROM node:24-alpine AS web-prod-deps
WORKDIR /app/web
COPY web/package.json web/package-lock.json ./
RUN npm ci --omit=dev

FROM golang:1.26-alpine AS go-build
WORKDIR /app/server
COPY server/go.mod server/go.sum ./
RUN go mod download
COPY server/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api \
 && CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/seed ./cmd/seed \
 && CGO_ENABLED=0 GOBIN=/out go install -tags postgres github.com/golang-migrate/migrate/v4/cmd/migrate@v4.20.1

FROM node:24-alpine AS runtime
RUN apk add --no-cache ca-certificates supervisor tzdata
COPY --from=caddy:2-alpine /usr/bin/caddy /usr/bin/caddy
WORKDIR /app

COPY --from=web-prod-deps /app/web/node_modules ./web/node_modules
COPY web/package.json ./web/package.json
COPY --from=web-build /app/web/build ./web/build
COPY --from=go-build /out/api /app/bin/api
COPY --from=go-build /out/seed /app/bin/seed
COPY --from=go-build /out/migrate /app/bin/migrate
COPY server/migrations /app/migrations

COPY Caddyfile /etc/caddy/Caddyfile
COPY supervisord.conf /etc/supervisord.conf
COPY docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh

RUN chmod +x /usr/local/bin/docker-entrypoint.sh /app/bin/*

ENV NODE_ENV=production
EXPOSE 10000
ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]
