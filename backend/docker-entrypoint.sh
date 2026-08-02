#!/bin/sh
set -eu

# docker-entrypoint.sh — envitoo API/mailer/worker image entrypoint.
#
# The container accepts configuration two ways:
#   1. Runtime environment variables (docker run -e, compose environment:/env_file:)
#   2. An optional env file mounted at /app/.env (e.g. -v "$PWD/.env:/app/.env")
#
# Runtime env vars always win: the file is only consulted for keys that
# are not already set. Values may be quoted ('...' or "..."); quotes and
# surrounding whitespace are stripped.

ENV_FILE="${ENVTOO_ENV_FILE:-/app/.env}"

if [ -f "$ENV_FILE" ]; then
  while IFS='=' read -r key value || [ -n "$key" ]; do
    case "$key" in
      '' | \#*) continue ;;
    esac
    key=$(printf '%s' "$key" | tr -d '[:space:]')
    [ -z "$key" ] && continue
    if ! env | grep -q "^${key}="; then
      value=$(printf '%s' "$value" | sed -e "s/^['\"]//" -e "s/['\"]\$//")
      export "$key=$value"
    fi
  done < "$ENV_FILE"
fi

exec "$@"
