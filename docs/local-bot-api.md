# Local Bot API Server

Removes the 30 msg/s Telegram cloud limit. Local server limit: **1000 msg/s**.

## 1. Get API credentials

Go to **https://my.telegram.org** → Log in → **API development tools** → Create application.

You'll get:
- `api_id` (integer, e.g. `12345678`)
- `api_hash` (string, e.g. `abcdef1234567890abcdef1234567890`)

These are account-level credentials, not bot-specific. Any Telegram account works.

## 2. Add to .env

```env
TELEGRAM_API_ID=12345678
TELEGRAM_API_HASH=abcdef1234567890abcdef1234567890
```

## 3. Log out the bot from Telegram cloud (one-time, required)

The bot must disconnect from the cloud API before connecting to the local server.
Run this once in your terminal (replace with your token):

```bash
curl "https://api.telegram.org/bot<YOUR_TOKEN>/logOut"
# Expected: {"ok":true,"result":true}
```

**After this the bot stops receiving updates from the cloud until you reconnect it.**

## 4. Start

```bash
docker compose up --build
```

The `telegram-bot-api` container starts automatically. The app connects to it via
`http://telegram-bot-api:8081/bot%s/%s` (set in docker-compose.yml).

## 5. Verify

Check that the local server is running:
```bash
curl http://localhost:8081/
# Expected: {"ok":false,"error_code":404,"description":"Not Found"}
# (404 is correct — it means the server is up, just no route at /)
```

Check bot is connected via the admin panel: **🤖 Bot → ● Connected as @yourbot**

## Rollback to cloud API

If you need to switch back:

```bash
# 1. Stop the local server
docker compose stop telegram-bot-api

# 2. Re-connect to cloud (call from local server while it's still running, or restart bot)
curl "http://localhost:8081/bot<YOUR_TOKEN>/logOut"

# 3. Change endpoint in .env
PUBLIKA_AUCTION_BOT_TG_ENDPOINT=https://api.telegram.org/bot%s/%s

# 4. Restart app
```

## Local vs Cloud summary

| | Cloud | Local Server |
|---|---|---|
| msg/s limit | 30 | 1000 |
| Setup | Zero | ~5 min |
| File size limit | 50 MB | 2000 MB |
| Runs on your server | No | Yes |
