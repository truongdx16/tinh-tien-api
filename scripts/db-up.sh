#!/usr/bin/env bash
# Makes sure the local database is up before `make run` / `migrate` / `seed`.
#
#  - Docker not running      → opens Docker Desktop (macOS) and waits
#  - DB_CONTAINER is set     → starts that container if stopped, turns on
#                              auto-restart, checks DB_DSN's port matches the
#                              container's published port, waits until MariaDB
#                              accepts connections
#  - DB_CONTAINER not set    → docker compose up --wait (deployments/)
#
# Reads DB_CONTAINER, DB_PASSWORD, DB_DSN from the environment (the Makefile
# exports .env). Skip with SKIP_DB_UP=1 (e.g. a database outside Docker).
set -euo pipefail

if [[ "${SKIP_DB_UP:-}" == "1" ]]; then
  echo "SKIP_DB_UP=1: bỏ qua kiểm tra database"
  exit 0
fi

fail() { echo "Lỗi: $*" >&2; exit 1; }

# 1. Docker daemon
if ! docker info >/dev/null 2>&1; then
  if [[ "$(uname)" == "Darwin" ]]; then
    echo "Docker chưa chạy → mở Docker Desktop..."
    open -a Docker || true
    for _ in $(seq 1 60); do
      docker info >/dev/null 2>&1 && break
      sleep 2
    done
  fi
  docker info >/dev/null 2>&1 || fail "Docker không chạy. Hãy mở Docker rồi chạy lại."
fi

# 2a. No named container: use docker compose.
if [[ -z "${DB_CONTAINER:-}" ]]; then
  docker compose -f deployments/docker-compose.yml up -d --wait
  exit 0
fi

# 2b. Named container (local data, e.g. imported from Firebase).
docker inspect "$DB_CONTAINER" >/dev/null 2>&1 ||
  fail "không có container '$DB_CONTAINER' (DB_CONTAINER trong .env).
      Xoá DB_CONTAINER khỏi .env để dùng docker compose, hoặc tạo lại container đó."

if [[ "$(docker inspect -f '{{.State.Running}}' "$DB_CONTAINER")" != "true" ]]; then
  echo "khởi động container $DB_CONTAINER..."
  docker start "$DB_CONTAINER" >/dev/null
fi

# Come back by itself after Docker / the Mac restarts (unless stopped by hand).
docker update --restart unless-stopped "$DB_CONTAINER" >/dev/null

# DB_DSN must point at the port the container publishes.
host_port="$(docker port "$DB_CONTAINER" 3306/tcp 2>/dev/null | head -1 | sed 's/.*://')"
dsn_port="$(printf '%s' "${DB_DSN:-}" | sed -n 's/.*tcp([^:]*:\([0-9]*\)).*/\1/p')"
if [[ -n "$host_port" && -n "$dsn_port" && "$host_port" != "$dsn_port" ]]; then
  fail "DB_DSN dùng cổng $dsn_port nhưng container $DB_CONTAINER mở cổng $host_port.
      Sửa DB_DSN trong .env thành ...tcp(127.0.0.1:$host_port)/..."
fi

printf "chờ database %s" "$DB_CONTAINER"
for _ in $(seq 1 60); do
  # `ping` succeeds as soon as the server answers (even if auth would fail).
  if docker exec "$DB_CONTAINER" mariadb-admin ping -h127.0.0.1 -uroot \
      -p"${DB_PASSWORD:-}" --silent >/dev/null 2>&1; then
    echo " → sẵn sàng"
    exit 0
  fi
  printf "."
  sleep 1
done
echo
fail "database không sẵn sàng sau 60s. Xem log: docker logs $DB_CONTAINER"
