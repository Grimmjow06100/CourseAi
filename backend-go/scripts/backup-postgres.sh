#!/bin/sh
set -eu

: "${DATABASE_URL:?DATABASE_URL is required}"
output="${1:-course-ai-$(date -u +%Y%m%dT%H%M%SZ).dump}"

pg_dump --dbname="$DATABASE_URL" --format=custom --no-owner --no-privileges --file="$output"
echo "Backup written to $output"
