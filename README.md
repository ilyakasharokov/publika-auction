# Publika Auction

A flexible Telegram auction service with a real-time admin panel, optimised for high load.

## Features

- **Multiple auctions** — create and manage any number of auctions with configurable lots and bid steps
- **Real-time admin panel** — live bid feed via SSE, htmx-powered UI, no page reloads
- **High-load bid placement** — Redis distributed lock + in-memory cache + async MongoDB writes
- **Telegram bot** — participants bid directly in Telegram; connect/disconnect the bot from the admin panel without restart
- **Prometheus metrics** — bids/sec, lock contention, TG queue depth, HTTP latencies at `/metrics`
- **Photo upload** — upload lot photos directly or provide a URL

## Stack

- **Go 1.19+** — backend
- **MongoDB** — persistent storage
- **Redis** — distributed bid locking
- **Telegram Bot API** — participant interface
- **Prometheus** — metrics
- **htmx + Pico CSS** — admin panel (no build step)

## Quick start

### Local development

```bash
# Start MongoDB and Redis
docker compose up redis mongo -d

# Copy and configure env
cp .env.example .env

# Run
make run
```

Open **http://localhost:8002/admin** — default login `admin` / `changeme`.

### Full Docker

```bash
docker compose up --build
```

## Configuration

| Variable | Default | Description |
|---|---|---|
| `PUBLIKA_AUCTION_BOT_TOKEN` | — | Telegram bot token (or set from admin panel) |
| `PUBLIKA_AUCTION_BOT_ADDR` | `:8002` | HTTP listen address |
| `PUBLIKA_AUCTION_BOT_MONGO_URI` | `mongodb://localhost:27017` | MongoDB URI |
| `PUBLIKA_AUCTION_BOT_MONGO_DB` | `auction` | MongoDB database name |
| `PUBLIKA_AUCTION_BOT_REDIS_ADDR` | `localhost:6379` | Redis address |
| `PUBLIKA_AUCTION_BOT_ADMIN_USER` | `admin` | Admin panel username |
| `PUBLIKA_AUCTION_BOT_ADMIN_PASSWORD` | `changeme` | Admin panel password |
| `PUBLIKA_AUCTION_BOT_SESSION_SECRET` | — | HMAC session signing key (change in production) |
| `PUBLIKA_AUCTION_BOT_BID_STEP` | `2000` | Default minimum bid increment |

## Admin panel

| Route | Description |
|---|---|
| `/admin/auctions` | List, create, activate, and end auctions |
| `/admin/auctions/{slug}` | Auction detail — lot grid with live status |
| `/admin/auctions/{slug}/lots/{num}` | Lot detail — live bid table, sell/cancel bids |
| `/admin/clients` | Registered participants |
| `/admin/clients/{phone}` | Client detail — bid history and chat |
| `/admin/settings` | Connect / disconnect Telegram bot |
| `/metrics` | Prometheus metrics |
| `/health` | Liveness probe |

## Auction lifecycle

```
draft → active → ended
```

1. Create an auction (slug + bid step)
2. Add lots (title, description, photo, starting price)
3. **Start** — activates the auction; bot begins accepting bids immediately
4. On each lot detail page click **✓ Sell** next to the winning bid
5. **End** — closes the auction

## Architecture

```
cmd/main.go
internal/
├── domain/      Auction, Lot, Bid, Client types
├── repo/        Interfaces + MongoDB implementations + in-memory caches
├── service/     Bid placement, auction lifecycle, client management
├── lock/        Redis distributed lock (SET NX PX + Lua release)
├── tgqueue/     Buffered TG send queue (1000 cap, 3 workers)
├── hub/         Telegram chat state machine
├── tg/          Bot runner + hot-plug manager
├── metrics/     Prometheus metric definitions
└── admin/       HTTP handlers, SSE hub, embedded HTML templates
```

### Bid placement (~3ms critical section)

1. Check in-memory cache — reject early if amount is too low (no Redis round-trip)
2. Acquire Redis lock on `lock:{auctionID}:{lotID}` (500ms TTL)
3. Re-check under lock
4. Update cache, release lock
5. Async: write to MongoDB + notify outbid participant + publish SSE event

## License

MIT
