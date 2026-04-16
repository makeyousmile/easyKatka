#!/usr/bin/env bash
set -u

BASE_URL="https://api.opendota.com/api"
TIMEOUT=20
ENDPOINTS=(
  "/"
  "/health"
  "/heroes"
  "/heroStats"
  "/metadata"
)

has_failures=0

printf 'Проверка OpenDota API\n\n'

for endpoint in "${ENDPOINTS[@]}"; do
  if [ "$endpoint" = "/" ]; then
    url="$BASE_URL"
  else
    url="$BASE_URL$endpoint"
  fi

  result="$(curl -sS -o /dev/null -w "%{http_code} %{time_total} %{errormsg}" --max-time "$TIMEOUT" "$url" 2>&1)"
  status_code="$(printf '%s' "$result" | awk '{print $1}')"
  time_total="$(printf '%s' "$result" | awk '{print $2}')"
  details="$(printf '%s' "$result" | cut -d' ' -f3-)"

  if [ -z "$details" ]; then
    details="OK"
  fi

  if [ "$status_code" = "200" ]; then
    printf '%-12s %-6s %8sms  %s\n' "$endpoint" "$status_code" "$(awk "BEGIN { printf \"%.0f\", $time_total * 1000 }")" "$details"
  else
    has_failures=1
    if [ "$status_code" = "000" ]; then
      status_code="ERROR"
    fi
    printf '%-12s %-6s %8sms  %s\n' "$endpoint" "$status_code" "$(awk "BEGIN { printf \"%.0f\", $time_total * 1000 }")" "$details"
  fi
done

printf '\n'
if [ "$has_failures" -ne 0 ]; then
  printf "Есть недоступные или нестабильные endpoint'ы.\n"
  exit 1
fi

printf "Все проверенные endpoint'ы ответили успешно.\n"
