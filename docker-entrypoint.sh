#!/bin/sh
set -eu

run_migrations="${RUN_MIGRATIONS:-true}"

if [ "$run_migrations" = "true" ] || [ "$run_migrations" = "1" ]; then
  if [ -z "${DATABASE_URL:-}" ]; then
    echo "DATABASE_URL is required when RUN_MIGRATIONS is enabled."
    exit 1
  fi

  migrations_path="${MIGRATIONS_PATH:-/app/db/migrations}"
  echo "Running database migrations from ${migrations_path}..."

  migrate_output=""
  if ! migrate_output="$(migrate -path "${migrations_path}" -database "${DATABASE_URL}" up 2>&1)"; then
    case "$migrate_output" in
      *"no change"*)
        echo "$migrate_output"
        ;;
      *)
        echo "$migrate_output"
        exit 1
        ;;
    esac
  else
    if [ -n "$migrate_output" ]; then
      echo "$migrate_output"
    fi
  fi
fi

exec /app/ournezt-core
