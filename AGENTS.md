# AGENTS.md

## Overview

Single-package Go bot (`package main`) — a Telegram shopping/to-do list bot backed by Redis. Three source files: `main.go` (bot logic), `db.go` (Redis layer), `main_test.go` (unit tests).

## Commands

```
go build ./...          # build
go test ./...           # run all unit tests (pure functions only, no Redis needed)
go vet ./...            # static analysis (no linter configured)
docker-compose up --build  # run with Redis via Docker
```

No linting tool is configured. No CI pipeline exists.

## Environment

Required env vars (set in `.env` or environment):

| Variable | Purpose |
|---|---|
| `TELEGRAM_BOT_TOKEN` | Telegram Bot API token (sensitive) |
| `REDIS_ADDR` | e.g. `localhost:6379` locally, `redis:6379` in Docker |
| `REDIS_USERNAME` | Redis auth username (optional) |
| `REDIS_PASSWORD` | Redis auth password (optional) |
| `REDIS_DB` | Redis DB number, default `0` |

`example.env` exists but is missing `REDIS_ADDR` — needed for local (non-Docker) runs.

## Security

- `.env` contains a real bot token and is correctly gitignored.
- The compiled binary `gotelegramtodo` is untracked but NOT gitignored — add it to `.gitignore` to avoid accidentally committing builds (they embed the bot token or change behavior silently).
- Do not log or expose `TELEGRAM_BOT_TOKEN`.

## Architecture

- All code lives in a flat `main` package — no `cmd/`, `internal/`, or `pkg/` directories.
- Bot listens via long-polling (`b.Start(ctx)`). Key handler: `/make_list` command creates an interactive inline-keyboard list; `defaultHandler` appends new items from any message when the list is unlocked.
- State is per-chat-thread, keyed by `chatID:threadID` in Redis as JSON.
- The `Locked` flag on `ChatListData` prevents new items from being added — toggled by an inline button.

## Testing

Tests cover only pure functions (`buttonText`, `lockedImage`, `parseShoppingList`, `generateChatKey`, `formatListDataButton`, `applyCallback`, `ensureItemIDs`). No integration tests, no mock Redis, no bot handler tests. Tests run without Redis.

## Gotchas

- `parseShoppingList` splits on newlines; each line becomes a separate item.
- `REDIS_ADDR` is set via `docker-compose.yml` override (`redis:6379`), not in `example.env` — local dev requires setting it manually.
- No generated code, no codegen, no migrations.
