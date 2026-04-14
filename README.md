# easyKatka

## Что делает программа

`easyKatka` следит за матчами указанных Dota 2 аккаунтов через OpenDota.

Программа:
- читает список Steam/Dota аккаунтов из файла `account_id`
- получает свежие матчи и информацию по игрокам
- формирует сводку по матчам
- может работать как Telegram-бот, если задан `TELEGRAM_BOT_TOKEN`
- может отправлять уведомления о новых матчах в Telegram, если задан `TELEGRAM_NOTIFY_CHAT_ID`

Если Telegram-токен не задан, программа выводит отчёт в консоль и продолжает мониторинг матчей в фоне.

Поддерживаемые команды бота:
- `/stat`
- `/rating`
- `/friends <число>`
- `/chatid`

## Что нужно перед запуском

В корне проекта должны быть:

- файл `account_id` с аккаунтами, по одному ID на строку
- файл `.env` с переменными окружения

Пример `.env`:

```env
TELEGRAM_BOT_TOKEN=your_telegram_bot_token
TELEGRAM_NOTIFY_CHAT_ID=your_telegram_notify_chat_id
```

Если нужен только запуск без Telegram-бота, `TELEGRAM_BOT_TOKEN` можно не задавать.

## Как запускать

### Windows

Есть автоматический скрипт:

```powershell
powershell -ExecutionPolicy Bypass -File .\install.ps1
```

Скрипт:
- проверяет наличие Docker
- при необходимости устанавливает Docker Desktop
- ждёт готовности Docker
- создаёт `.env` из `.env.example`, если файла нет
- запускает `docker compose up -d --build`

### Ubuntu

Есть отдельный скрипт:

```bash
chmod +x ./install-ubuntu.sh
./install-ubuntu.sh
```

Скрипт:
- проверяет наличие Docker
- при необходимости устанавливает Docker Engine и `docker compose plugin`
- создаёт `.env` из `.env.example`, если файла нет
- запускает контейнер через `docker compose`

### Ручной запуск через Docker Compose

Если Docker уже установлен:

```bash
docker compose up -d --build
```

Остановить сервис:

```bash
docker compose down
```

Посмотреть логи:

```bash
docker compose logs -f
```
