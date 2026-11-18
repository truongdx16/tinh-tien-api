#!/usr/bin/env bash
# ============================================================
# Tinh Tien API — deploy to a VPS (no Docker)
#
# Builds a Linux binary locally, uploads a release to the VPS, switches the
# `current` symlink and restarts the systemd service. Nginx (TLS) sits in
# front; MariaDB is already installed on the VPS.
#
# Usage (see deploy/README.md):
#   ./scripts/deploy.sh setup      # once: user, folders, nginx, TLS, systemd
#   ./scripts/deploy.sh db-init    # once: create database + user in MariaDB
#   ./scripts/deploy.sh            # deploy (build, upload, restart, check)
#   ./scripts/deploy.sh seed       # once: create the owner account
#   ./scripts/deploy.sh rollback   # back to the previous release
#   ./scripts/deploy.sh status     # service, release and health
#   ./scripts/deploy.sh logs       # follow the API log
#
# Config: scripts/deploy.config.env (+ .env.production for the app).
# Any config value can be set inline: DRY_RUN=true ./scripts/deploy.sh
# ============================================================
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
CONFIG_FILE="${DEPLOY_CONFIG:-$ROOT/scripts/deploy.config.env}"
ENV_FILE="${DEPLOY_ENV_FILE:-$ROOT/.env.production}"
TIMESTAMP="$(date '+%Y%m%d-%H%M%S')"
LOG_DIR="$ROOT/deploy/logs"
mkdir -p "$LOG_DIR"
LOG_FILE="$LOG_DIR/deploy-$TIMESTAMP.log"
BUILD_DIR="$ROOT/deploy/.build"

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; CYAN='\033[0;36m'; NC='\033[0m'
log()     { echo -e "${CYAN}[$(date '+%H:%M:%S')]${NC} $*" | tee -a "$LOG_FILE"; }
success() { echo -e "${GREEN}[OK]${NC} $*" | tee -a "$LOG_FILE"; }
warn()    { echo -e "${YELLOW}[WARN]${NC} $*" | tee -a "$LOG_FILE"; }
die()     { echo -e "${RED}[ERROR]${NC} $*" | tee -a "$LOG_FILE"; exit 1; }
require_cmd() { command -v "$1" >/dev/null 2>&1 || die "Missing command: $1"; }
# Never stop silently: any failing step reports where it happened.
trap 'echo -e "${RED}[ERROR]${NC} Step failed (line $LINENO, exit $?) — see $LOG_FILE" | tee -a "$LOG_FILE"' ERR

# ─── Config ───────────────────────────────────────────────
CONFIG_VARS=(VPS_HOST VPS_USER VPS_PORT DOMAIN CERTBOT_EMAIL VPS_PATH SERVICE_NAME APP_USER
  API_PORT TARGET_ARCH RUN_TESTS KEEP_RELEASES UPLOAD_ENV DRY_RUN FORCE_JWT_CHANGE BACKUP_DB)

load_config() {
  # Inline values (VAR=x ./deploy.sh) win over the config file.
  local name saved=()
  for name in "${CONFIG_VARS[@]}"; do
    [[ -n "${!name+x}" ]] && saved+=("$name" "${!name}")
  done
  # shellcheck disable=SC1090
  [[ -f "$CONFIG_FILE" ]] && source "$CONFIG_FILE"
  local i
  for ((i = 0; i < ${#saved[@]}; i += 2)); do
    printf -v "${saved[i]}" '%s' "${saved[i + 1]}"
  done

  VPS_HOST="${VPS_HOST:-}"
  VPS_USER="${VPS_USER:-root}"
  VPS_PORT="${VPS_PORT:-22}"
  DOMAIN="${DOMAIN:-7b.truongdx.dev}"
  CERTBOT_EMAIL="${CERTBOT_EMAIL:-}"
  VPS_PATH="${VPS_PATH:-/var/www/tinh-tien-api}"
  SERVICE_NAME="${SERVICE_NAME:-tinh-tien-api}"
  APP_USER="${APP_USER:-tinhtien}"
  API_PORT="${API_PORT:-2170}"
  TARGET_ARCH="${TARGET_ARCH:-amd64}"
  RUN_TESTS="${RUN_TESTS:-true}"
  KEEP_RELEASES="${KEEP_RELEASES:-5}"
  UPLOAD_ENV="${UPLOAD_ENV:-true}"
  DRY_RUN="${DRY_RUN:-false}"
  FORCE_JWT_CHANGE="${FORCE_JWT_CHANGE:-false}"
  BACKUP_DB="${BACKUP_DB:-true}"

  [[ -n "$VPS_HOST" && "$VPS_HOST" != "your-vps-ip" ]] ||
    die "Set VPS_HOST in $CONFIG_FILE (cp scripts/deploy.config.example.env scripts/deploy.config.env)"
}

remote() {
  ssh -p "$VPS_PORT" -o ConnectTimeout=10 -o BatchMode=yes "$VPS_USER@$VPS_HOST" "$@"
}

# Runs a bash script (stdin) on the VPS as root (sudo when not root).
# Arguments are passed to the script as $1, $2...
remote_root() {
  local sudo="" args=""
  [[ "$VPS_USER" != "root" ]] && sudo="sudo"
  local a
  for a in "$@"; do args+=" $(printf '%q' "$a")"; done
  ssh -p "$VPS_PORT" -o ConnectTimeout=10 -o BatchMode=yes "$VPS_USER@$VPS_HOST" "$sudo bash -s --$args"
}

upload() { # upload <local> <remote-path>
  local flags=(-az)
  [[ "$DRY_RUN" == "true" ]] && flags+=(--dry-run -v)
  rsync "${flags[@]}" -e "ssh -p $VPS_PORT -o BatchMode=yes" "$1" "$VPS_USER@$VPS_HOST:$2"
}

env_value() { # env_value <KEY> <file>
  sed -n "s/^$1=//p" "$2" | tail -1
}

check_ssh() {
  log "Checking SSH to $VPS_USER@$VPS_HOST:$VPS_PORT"
  remote true || die "Cannot SSH (key login) to $VPS_USER@$VPS_HOST — add your key: ssh-copy-id -p $VPS_PORT $VPS_USER@$VPS_HOST"
}

# ─── setup: one-time VPS provisioning ─────────────────────
cmd_setup() {
  check_ssh
  [[ -n "$CERTBOT_EMAIL" ]] || die "Set CERTBOT_EMAIL in $CONFIG_FILE (Let's Encrypt notices)"
  log "Uploading nginx + systemd files"
  remote "mkdir -p /tmp/tinhtien-setup"
  upload "$ROOT/deploy/nginx/" "/tmp/tinhtien-setup/nginx/"
  upload "$ROOT/deploy/systemd/" "/tmp/tinhtien-setup/systemd/"

  log "Provisioning $DOMAIN on the VPS"
  remote_root "$DOMAIN" "$CERTBOT_EMAIL" "$VPS_PATH" "$SERVICE_NAME" "$APP_USER" <<'REMOTE'
set -euo pipefail
DOMAIN="$1"; EMAIL="$2"; APP_DIR="$3"; SERVICE="$4"; APP_USER="$5"
SRC=/tmp/tinhtien-setup

for c in nginx certbot systemctl; do
  command -v "$c" >/dev/null || { echo "Missing $c — install: apt update && apt install -y nginx certbot"; exit 1; }
done

# App user without login + folders (releases are replaced, .env and uploads stay).
id "$APP_USER" >/dev/null 2>&1 || useradd --system --no-create-home --shell /usr/sbin/nologin "$APP_USER"
mkdir -p "$APP_DIR/releases" "$APP_DIR/uploads" /var/www/certbot
chown -R "$APP_USER:$APP_USER" "$APP_DIR/uploads"
chmod 755 "$APP_DIR" "$APP_DIR/uploads"

# Nginx: shared proxy snippet.
install -m 644 "$SRC/nginx/tinhtien-proxy.conf" /etc/nginx/snippets/tinhtien-proxy.conf

SITE=/etc/nginx/sites-available/$DOMAIN.conf
if [[ ! -f /etc/letsencrypt/live/$DOMAIN/fullchain.pem ]]; then
  echo "→ First certificate: temporary HTTP-only site"
  install -m 644 "$SRC/nginx/$DOMAIN.http-only.conf" "$SITE"
  ln -sfn "$SITE" "/etc/nginx/sites-enabled/$DOMAIN.conf"
  nginx -t && systemctl reload nginx
  certbot certonly --webroot -w /var/www/certbot -d "$DOMAIN" \
    --email "$EMAIL" --agree-tos --non-interactive
fi
echo "→ HTTPS site"
install -m 644 "$SRC/nginx/$DOMAIN.conf" "$SITE"
ln -sfn "$SITE" "/etc/nginx/sites-enabled/$DOMAIN.conf"
nginx -t && systemctl reload nginx

# Reload nginx after automatic renewals.
mkdir -p /etc/letsencrypt/renewal-hooks/deploy
cat > /etc/letsencrypt/renewal-hooks/deploy/reload-nginx.sh <<'HOOK'
#!/bin/sh
systemctl reload nginx
HOOK
chmod +x /etc/letsencrypt/renewal-hooks/deploy/reload-nginx.sh

# systemd unit (started by the first deploy, once a release exists).
install -m 644 "$SRC/systemd/$SERVICE.service" "/etc/systemd/system/$SERVICE.service"
systemctl daemon-reload
systemctl enable "$SERVICE" >/dev/null
rm -rf "$SRC"
echo "→ Setup done"
REMOTE
  success "VPS ready. Next: ./scripts/deploy.sh db-init (if the database is new), then ./scripts/deploy.sh"
}

# ─── db-init: database + user in the existing MariaDB ─────
cmd_db_init() {
  check_ssh
  [[ -f "$ENV_FILE" ]] || die "Missing $ENV_FILE (cp .env.production.example .env.production)"
  local dsn user pass db
  dsn="$(env_value DB_DSN "$ENV_FILE")"
  user="${dsn%%:*}"
  pass="${dsn#*:}"; pass="${pass%%@tcp(*}"
  db="${dsn#*)/}"; db="${db%%\?*}"
  [[ -n "$user" && -n "$pass" && -n "$db" && "$pass" != "CHANGE_DB_PASSWORD" ]] ||
    die "Set a real user/password/database in DB_DSN of $ENV_FILE"
  [[ "$db" =~ ^[A-Za-z0-9_]+$ && "$user" =~ ^[A-Za-z0-9_]+$ ]] || die "Database/user names: letters, digits, _ only"
  log "Creating database '$db' and user '$user'@'localhost' (if missing) as MariaDB root"
  # The password travels inside the script (stdin, base64), never on a command line.
  local pass_b64
  pass_b64="$(printf '%s' "$pass" | base64 | tr -d '\n')"
  {
    echo "PASS_B64='$pass_b64'"
    cat <<'REMOTE'
set -euo pipefail
DB="$1"; DB_USER="$2"
PASS="$(printf '%s' "$PASS_B64" | base64 -d)"
# SQL string literal: double backslashes and quotes (sed behaves the same everywhere).
PASS_SQL="$(printf '%s' "$PASS" | sed -e 's/\\/\\\\/g' -e "s/'/''/g")"
MYSQL=mariadb; command -v mariadb >/dev/null || MYSQL=mysql
"$MYSQL" -uroot <<SQL
CREATE DATABASE IF NOT EXISTS \`$DB\` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE USER IF NOT EXISTS '$DB_USER'@'localhost' IDENTIFIED BY '$PASS_SQL';
CREATE USER IF NOT EXISTS '$DB_USER'@'127.0.0.1' IDENTIFIED BY '$PASS_SQL';
GRANT ALL PRIVILEGES ON \`$DB\`.* TO '$DB_USER'@'localhost';
GRANT ALL PRIVILEGES ON \`$DB\`.* TO '$DB_USER'@'127.0.0.1';
FLUSH PRIVILEGES;
SQL
echo "→ Database ready"
REMOTE
  } | remote_root "$db" "$user"
  success "Database '$db' ready (tables are created by the API on first start)"
}

# ─── deploy ───────────────────────────────────────────────
preflight_env() {
  [[ -f "$ENV_FILE" ]] || die "Missing $ENV_FILE (cp .env.production.example .env.production)"
  local secret dsn url
  secret="$(env_value JWT_SECRET "$ENV_FILE")"
  dsn="$(env_value DB_DSN "$ENV_FILE")"
  url="$(env_value PUBLIC_BASE_URL "$ENV_FILE")"
  [[ "$(env_value APP_ENV "$ENV_FILE")" == "production" ]] || die "APP_ENV=production missing in $ENV_FILE"
  [[ ${#secret} -ge 32 && "$secret" != *change-me* ]] || die "JWT_SECRET in $ENV_FILE: at least 32 random chars (openssl rand -hex 32)"
  [[ -n "$dsn" && "$dsn" != *CHANGE_DB_PASSWORD* ]] || die "Set DB_DSN in $ENV_FILE"
  [[ "$dsn" == *loc=UTC* ]] || die "DB_DSN must contain loc=UTC"
  [[ "$url" == https://* ]] || die "PUBLIC_BASE_URL must be https://... in $ENV_FILE"
  [[ "$(env_value HTTP_ADDR "$ENV_FILE")" == 127.0.0.1:* ]] ||
    warn "HTTP_ADDR is not 127.0.0.1:<port> — the API would be reachable without nginx"
}

build() {
  require_cmd go
  if [[ "$RUN_TESTS" == "true" ]]; then
    log "Running tests (RUN_TESTS=false to skip)"
    (cd "$ROOT" && go test ./... >>"$LOG_FILE" 2>&1) || die "Tests failed — see $LOG_FILE"
    success "Tests passed"
  fi
  log "Building linux/$TARGET_ARCH binaries"
  rm -rf "$BUILD_DIR"
  mkdir -p "$BUILD_DIR/bin" "$BUILD_DIR/configs" "$BUILD_DIR/api"
  local cmd
  for cmd in api migrate seed; do
    (cd "$ROOT" && CGO_ENABLED=0 GOOS=linux GOARCH="$TARGET_ARCH" \
      go build -trimpath -ldflags="-s -w" -o "$BUILD_DIR/bin/$cmd" "./cmd/$cmd") ||
      die "Build of cmd/$cmd failed"
  done
  cp "$ROOT/configs/config.yaml" "$BUILD_DIR/configs/config.yaml"
  cp "$ROOT/api/openapi.yaml" "$BUILD_DIR/api/openapi.yaml"
  # Commit of the build; "-dirty" when it includes uncommitted changes.
  local rev
  rev="$(git -C "$ROOT" rev-parse --short HEAD 2>/dev/null || echo unknown)"
  [[ -n "$(git -C "$ROOT" status --porcelain 2>/dev/null)" ]] && rev="$rev-dirty"
  echo "$rev" >"$BUILD_DIR/REVISION"
  success "Built $(du -sh "$BUILD_DIR/bin" | cut -f1) ($(cat "$BUILD_DIR/REVISION"))"
}

upload_env() {
  [[ "$UPLOAD_ENV" == "true" ]] || { log "UPLOAD_ENV=false: keeping the VPS .env"; return; }
  local remote_secret local_secret
  local_secret="$(env_value JWT_SECRET "$ENV_FILE")"
  remote_secret="$(remote "sed -n 's/^JWT_SECRET=//p' $VPS_PATH/.env 2>/dev/null | tail -1" || true)"
  if [[ -n "$remote_secret" && "$remote_secret" != "$local_secret" && "$FORCE_JWT_CHANGE" != "true" ]]; then
    die "JWT_SECRET differs from the VPS — every user would be signed out. Re-run with FORCE_JWT_CHANGE=true if intended."
  fi
  if remote "cmp -s - $VPS_PATH/.env" <"$ENV_FILE" 2>/dev/null; then
    log ".env unchanged"
    return
  fi
  [[ "$DRY_RUN" == "true" ]] && { log "[dry-run] would upload .env"; return; }
  log "Uploading .env (previous kept as .env.bak-$TIMESTAMP)"
  upload "$ENV_FILE" "$VPS_PATH/.env.new"
  remote_root "$VPS_PATH" "$TIMESTAMP" <<'REMOTE'
set -euo pipefail
cd "$1"
if [[ -f .env ]]; then cp -p .env ".env.bak-$2"; fi
mv .env.new .env && chown root:root .env && chmod 600 .env
# Keep the 5 newest backups (none yet on the first deploy).
for old in $(ls -1t .env.bak-* 2>/dev/null | tail -n +6); do rm -f "$old"; done
true
REMOTE
}

# Dumps the app database on the VPS before the new release (and its
# migrations) starts: /root/db-backups/<db>-<time>.sql.gz, newest 10 kept.
backup_db() {
  [[ "$BACKUP_DB" == "true" ]] || { log "BACKUP_DB=false: no database backup"; return; }
  local dsn db
  dsn="$(env_value DB_DSN "$ENV_FILE")"
  db="${dsn#*)/}"; db="${db%%\?*}"
  [[ "$db" =~ ^[A-Za-z0-9_]+$ ]] || { warn "Cannot read the database name from DB_DSN — skipping backup"; return; }
  log "Backing up database '$db' on the VPS"
  remote_root "$db" "$TIMESTAMP" <<'REMOTE'
set -euo pipefail
DB="$1"; TS="$2"; DIR=/root/db-backups
mkdir -p "$DIR" && chmod 700 "$DIR"
DUMP=mariadb-dump; command -v mariadb-dump >/dev/null || DUMP=mysqldump
"$DUMP" -uroot --single-transaction --routines "$DB" | gzip > "$DIR/$DB-$TS.sql.gz"
echo "→ $DIR/$DB-$TS.sql.gz ($(du -h "$DIR/$DB-$TS.sql.gz" | cut -f1))"
for old in $(ls -1t "$DIR/$DB"-*.sql.gz 2>/dev/null | tail -n +11); do rm -f "$old"; done
true
REMOTE
}

health_check() { # waits for /healthz on the VPS, then the public URL
  log "Health check"
  local ok=false i
  for i in $(seq 1 20); do
    if remote "curl -fsS -m 3 http://127.0.0.1:$API_PORT/healthz >/dev/null"; then ok=true; break; fi
    sleep 1
  done
  [[ "$ok" == true ]] || return 1
  if curl -fsS -m 10 "https://$DOMAIN/healthz" >/dev/null 2>&1; then
    success "https://$DOMAIN/healthz is up"
  else
    warn "API is up on the VPS but https://$DOMAIN/healthz is not reachable from here (DNS / nginx / TLS?)"
  fi
}

cmd_deploy() {
  require_cmd rsync
  preflight_env
  check_ssh
  remote "test -d $VPS_PATH/releases" || die "VPS not set up — run: ./scripts/deploy.sh setup"
  build

  local release="$VPS_PATH/releases/$TIMESTAMP"
  log "Uploading release $TIMESTAMP"
  remote "mkdir -p $release"
  upload "$BUILD_DIR/" "$release/"
  [[ "$DRY_RUN" == "true" ]] && { remote "rm -rf $release"; success "Dry run done (nothing changed)"; return; }
  upload_env
  backup_db

  log "Switching to the new release and restarting $SERVICE_NAME"
  remote_root "$VPS_PATH" "$release" "$SERVICE_NAME" "$KEEP_RELEASES" <<'REMOTE'
set -euo pipefail
APP_DIR="$1"; RELEASE="$2"; SERVICE="$3"; KEEP="$4"
chmod 755 "$RELEASE" "$RELEASE"/bin/*
# Remember the release we are leaving (for rollback).
if [[ -L "$APP_DIR/current" ]]; then readlink "$APP_DIR/current" > "$APP_DIR/.previous"; fi
ln -sfn "$RELEASE" "$APP_DIR/current.tmp" && mv -Tf "$APP_DIR/current.tmp" "$APP_DIR/current"
systemctl restart "$SERVICE"
# Keep the newest releases (+ the current one).
cd "$APP_DIR/releases"
ls -1dt */ | sed 's#/$##' | tail -n +$((KEEP + 1)) | while read -r old; do
  [[ "$APP_DIR/releases/$old" == "$(readlink "$APP_DIR/current")" ]] || rm -rf "$old"
done
REMOTE

  if health_check; then
    success "Deployed $(cat "$BUILD_DIR/REVISION") → https://$DOMAIN (release $TIMESTAMP)"
  else
    warn "API did not become healthy. Last log lines:"
    remote "journalctl -u $SERVICE_NAME -n 40 --no-pager" | tee -a "$LOG_FILE" || true
    die "Deploy failed — roll back with: ./scripts/deploy.sh rollback"
  fi
}

# ─── other commands ───────────────────────────────────────
cmd_rollback() {
  check_ssh
  log "Rolling back to the previous release"
  remote_root "$VPS_PATH" "$SERVICE_NAME" <<'REMOTE'
set -euo pipefail
APP_DIR="$1"; SERVICE="$2"
CURRENT="$(readlink "$APP_DIR/current" || true)"
TARGET="$(cat "$APP_DIR/.previous" 2>/dev/null || true)"
if [[ -z "$TARGET" || ! -d "$TARGET" || "$TARGET" == "$CURRENT" ]]; then
  # Fall back to the newest release older than the current one.
  TARGET="$(ls -1dt "$APP_DIR"/releases/*/ | sed 's#/$##' | grep -vx "$CURRENT" | head -1 || true)"
fi
[[ -n "$TARGET" && -d "$TARGET" ]] || { echo "No previous release to roll back to"; exit 1; }
echo "$CURRENT" > "$APP_DIR/.previous"
ln -sfn "$TARGET" "$APP_DIR/current.tmp" && mv -Tf "$APP_DIR/current.tmp" "$APP_DIR/current"
systemctl restart "$SERVICE"
echo "→ Now on $(basename "$TARGET")"
REMOTE
  health_check && success "Rolled back" || die "Still unhealthy — check: ./scripts/deploy.sh logs"
}

cmd_seed() {
  check_ssh
  log "Creating the owner account (OWNER_USERNAME/OWNER_PASSWORD from the VPS .env)"
  remote_root "$VPS_PATH" <<'REMOTE'
set -euo pipefail
cd "$1/current"
# Read .env like systemd does: KEY=value, optional quotes, no shell parsing
# (DB_DSN contains & and ?, which `. .env` would break).
while IFS= read -r line || [[ -n "$line" ]]; do
  [[ "$line" =~ ^[[:space:]]*# || "$line" != *=* ]] && continue
  key="${line%%=*}"; value="${line#*=}"
  [[ "$value" =~ ^\"(.*)\"$ || "$value" =~ ^\'(.*)\'$ ]] && value="${BASH_REMATCH[1]}"
  export "$key=$value"
done < "$1/.env"
[[ -n "${OWNER_USERNAME:-}" && -n "${OWNER_PASSWORD:-}" ]] || { echo "Set OWNER_USERNAME and OWNER_PASSWORD in .env.production, deploy, then seed"; exit 1; }
CONFIG_PATH="$1/current/configs/config.yaml" ./bin/seed
REMOTE
  success "Owner account ready — remove OWNER_PASSWORD from .env.production afterwards"
}

cmd_status() {
  check_ssh
  remote "systemctl --no-pager status $SERVICE_NAME | head -12; echo; echo \"release: \$(basename \$(readlink $VPS_PATH/current 2>/dev/null) 2>/dev/null) (\$(cat $VPS_PATH/current/REVISION 2>/dev/null))\"; ls -1t $VPS_PATH/releases | head -$KEEP_RELEASES"
  curl -fsS -m 10 "https://$DOMAIN/healthz" && echo || warn "https://$DOMAIN/healthz not reachable"
}

cmd_logs() {
  check_ssh
  remote -t "journalctl -u $SERVICE_NAME -f -n 100"
}

# ─── main ─────────────────────────────────────────────────
main() {
  local cmd="${1:-deploy}"
  load_config
  log "tinh-tien-api: $cmd → $VPS_USER@$VPS_HOST ($DOMAIN)"
  case "$cmd" in
    setup) cmd_setup ;;
    db-init) cmd_db_init ;;
    deploy) cmd_deploy ;;
    seed) cmd_seed ;;
    rollback) cmd_rollback ;;
    status) cmd_status ;;
    logs) cmd_logs ;;
    -h | --help | help) sed -n '2,20p' "$0" ;;
    *) die "Unknown command: $cmd (setup | db-init | deploy | seed | rollback | status | logs)" ;;
  esac
}

main "$@"
