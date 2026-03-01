#!/usr/bin/env bash
set -e

GREEN='\033[0;32m'
RESET='\033[0m'

changes=()

prompt() {
  printf "${GREEN}%s${RESET}" "$1"
}

prompt "Use PostgreSQL? [y/N] "
read -r use_pg

if [[ "$use_pg" =~ ^[Yy]$ ]]; then
  prompt "Docker or external? [docker/external] "
  read -r pg_mode

  if [[ "$pg_mode" == "external" ]]; then
    # Remove postgres service from compose.yaml (keep app service)
    if grep -q 'postgres:' compose.yaml 2>/dev/null; then
      sed -i '' '/^  postgres:/,/^  [^ ]/{ /^  postgres:/d; /^  [^ ]/!d; }' compose.yaml
      # More robust: use awk to remove the postgres service block
      awk '
        /^  postgres:/ { skip=1; next }
        skip && /^  [^ ]/ { skip=0 }
        skip { next }
        { print }
      ' compose.yaml > compose.yaml.tmp && mv compose.yaml.tmp compose.yaml
      changes+=("✓ Removed postgres service from compose.yaml (kept app)")
    fi

    # Remove db and db-down targets from Makefile
    if grep -q '^db:' Makefile 2>/dev/null; then
      # Remove db and db-down from .PHONY line
      sed -i '' 's/ db / /g; s/ db-down / /g' Makefile
      # Remove db: and db-down: target blocks
      awk '
        /^db:/ { skip=1; next }
        /^db-down:/ { skip=1; next }
        skip && /^[^ \t]/ { skip=0 }
        skip { next }
        { print }
      ' Makefile > Makefile.tmp && mv Makefile.tmp Makefile
      changes+=("✓ Removed make db / db-down targets")
    fi
  else
    changes+=("✓ Kept PostgreSQL with Docker (no changes)")
  fi
else
  # Remove pkg/db/
  if [ -d "pkg/db" ]; then
    rm -rf pkg/db
    changes+=("✓ Removed pkg/db/")
  fi

  # Remove postgres service from compose.yaml
  if grep -q 'postgres:' compose.yaml 2>/dev/null; then
    awk '
      /^  postgres:/ { skip=1; next }
      skip && /^  [^ ]/ { skip=0 }
      skip { next }
      { print }
    ' compose.yaml > compose.yaml.tmp && mv compose.yaml.tmp compose.yaml
    changes+=("✓ Removed postgres service from compose.yaml")
  fi

  # Remove DATABASE_URL from .env.example
  if [ -f ".env.example" ]; then
    sed -i '' '/DATABASE_URL/d' .env.example
    # Remove the comment line above it too
    sed -i '' '/Postgres Database URL/d' .env.example
    sed -i '' '/Point this at any postgres/d' .env.example
    sed -i '' '/To start the bundled postgres/d' .env.example
    changes+=("✓ Removed DATABASE_URL from .env.example")
  fi

  # Remove db/migration targets from Makefile
  if grep -q '^db:' Makefile 2>/dev/null; then
    sed -i '' 's/ db / /g; s/ db-down / /g; s/ migrate / /g; s/ migrate-down / /g; s/ migrate-new / /g' Makefile
    awk '
      /^# Database/ { skip=1; next }
      /^# Migrations/ { skip=1; next }
      /^db:/ { skip=1; next }
      /^db-down:/ { skip=1; next }
      /^migrate:/ { skip=1; next }
      /^migrate-down:/ { skip=1; next }
      /^migrate-new:/ { skip=1; next }
      skip && /^[^ \t#]/ && !/^\t/ { skip=0 }
      skip && /^$/ { skip=0; next }
      skip { next }
      { print }
    ' Makefile > Makefile.tmp && mv Makefile.tmp Makefile
    changes+=("✓ Removed db/migration Make targets")
  fi

  # Remove pgx/goose/sqlc deps
  if grep -qE 'pgx|goose|sqlc' go.mod 2>/dev/null; then
    go mod tidy 2>/dev/null || true
    changes+=("✓ Cleaned Go dependencies (go mod tidy)")
  fi
fi

echo ""
for c in "${changes[@]}"; do
  echo "$c"
done
