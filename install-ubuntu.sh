#!/usr/bin/env bash
set -euo pipefail

log_step() {
  echo "==> $1"
}

require_sudo() {
  if ! command -v sudo >/dev/null 2>&1; then
    echo "sudo не найден. Установите sudo или запустите скрипт от root." >&2
    exit 1
  fi
}

run_as_root() {
  if [ "$(id -u)" -eq 0 ]; then
    "$@"
  else
    sudo "$@"
  fi
}

install_docker() {
  require_sudo

  log_step "Устанавливаю Docker Engine и docker compose plugin"
  run_as_root apt-get update
  run_as_root apt-get install -y ca-certificates curl
  run_as_root install -m 0755 -d /etc/apt/keyrings

  if [ ! -f /etc/apt/keyrings/docker.asc ]; then
    run_as_root curl -fsSL https://download.docker.com/linux/ubuntu/gpg -o /etc/apt/keyrings/docker.asc
    run_as_root chmod a+r /etc/apt/keyrings/docker.asc
  fi

  local arch
  arch="$(dpkg --print-architecture)"
  local codename
  codename="$(. /etc/os-release && echo "$VERSION_CODENAME")"

  echo \
    "deb [arch=${arch} signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/ubuntu ${codename} stable" \
    | run_as_root tee /etc/apt/sources.list.d/docker.list >/dev/null

  run_as_root apt-get update
  run_as_root apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
  run_as_root systemctl enable --now docker
}

wait_docker_ready() {
  local timeout=180
  local elapsed=0

  while [ "$elapsed" -lt "$timeout" ]; do
    if docker version >/dev/null 2>&1 && docker compose version >/dev/null 2>&1; then
      return 0
    fi
    sleep 5
    elapsed=$((elapsed + 5))
  done

  echo "Docker не стал доступен за ${timeout} секунд." >&2
  exit 1
}

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

log_step "Проверяю Docker"
if ! command -v docker >/dev/null 2>&1; then
  install_docker
else
  if ! docker compose version >/dev/null 2>&1; then
    install_docker
  fi
fi

log_step "Жду готовности Docker"
wait_docker_ready

if [ ! -f ".env" ]; then
  if [ ! -f ".env.example" ]; then
    echo "Не найден .env и отсутствует .env.example." >&2
    exit 1
  fi

  log_step "Создаю .env из .env.example"
  cp .env.example .env
fi

if grep -q "your_telegram_" .env; then
  echo "Заполните .env реальными значениями TELEGRAM_BOT_TOKEN и TELEGRAM_NOTIFY_CHAT_ID, затем запустите install-ubuntu.sh снова." >&2
  exit 1
fi

if [ ! -f "account_id" ]; then
  echo "Не найден файл account_id." >&2
  exit 1
fi

log_step "Собираю и запускаю контейнер"
docker compose up -d --build

log_step "Готово"
echo "Сервис easykatka запущен через Docker Compose."
