#!/bin/sh
set -eu

: "${DATABASE_URL:?DATABASE_URL is required}"
: "${COURSE_AI_CONFIRM_RESTORE:?Set COURSE_AI_CONFIRM_RESTORE=yes to confirm a destructive restore}"

if [ "$COURSE_AI_CONFIRM_RESTORE" != "yes" ]; then
  echo "Restore cancelled: COURSE_AI_CONFIRM_RESTORE must equal yes" >&2
  exit 1
fi

dump="${1:?Usage: restore-postgres.sh path/to/backup.dump}"
pg_restore --dbname="$DATABASE_URL" --clean --if-exists --no-owner --no-privileges "$dump"
echo "Restore completed from $dump"
