#!/bin/sh
set -eu

export PORT="${PORT:-10000}"

: "${DATABASE_URL:?DATABASE_URL is required}"

if [ -z "${APP_ORIGIN:-}" ] && [ -n "${RENDER_EXTERNAL_URL:-}" ]; then
	export APP_ORIGIN="$RENDER_EXTERNAL_URL"
fi

: "${APP_ORIGIN:?APP_ORIGIN is required}"

MIGRATION_URL="${MIGRATION_DATABASE_URL:-$DATABASE_URL}"

echo "running database migrations"
/app/bin/migrate -path /app/migrations -database "$MIGRATION_URL" up

exec /usr/bin/supervisord -c /etc/supervisord.conf
