# Minimal Newsletter Survey — Design

Date: 2026-06-04
Owner: sspaeti

## Goal

Embed simple rating links in a markdown newsletter that record one anonymous vote per reader per issue into a DuckDB file on the author's FreeBSD server. The question and answer slugs live in the newsletter markdown — the backend is generic, so each issue can ask something different without code or schema changes.

## Non-goals

- No login, no cookies, no JavaScript on the click path.
- No comment box, no follow-up form (could be added later).
- No admin UI — analytics happens by attaching DuckDB to the server's Quack endpoint from the author's laptop.
- No cross-issue identity. The same reader voting on two different newsletters produces two unrelated hashes.

## Architecture

One Go binary running on `ti.sspaeti.duckdns.org` (FreeBSD). It:

1. Serves `GET /survey/{survey_id}/{answer}` on a localhost HTTP port → records a vote → redirects to `/thanks`.
2. Loads the `quack` DuckDB extension in the same process and calls `quack_serve(..., token => ..., allow_other_hostname => true)` so the author can `ATTACH 'quack:quack.sspaeti.duckdns.org' AS s` from their laptop.
3. Holds a single DuckDB connection. One process = no file-lock conflicts between writes and Quack reads. Click volume (~4000 subs × ~10% click-through) is far below DuckDB's single-writer limit.

Caddy (already running for Listmonk) reverse-proxies and terminates TLS:

- `ti.sspaeti.duckdns.org` → `localhost:8080` (HTTP/click handler)
- `quack.sspaeti.duckdns.org` → `localhost:9494` (Quack remote-protocol endpoint)

## Newsletter link format

```markdown
What did you think of today's newsletter?

[Awesome!](https://ti.sspaeti.duckdns.org/survey/2026-06-04/awesome)
[Pretty Good](https://ti.sspaeti.duckdns.org/survey/2026-06-04/good)
[Could be better](https://ti.sspaeti.duckdns.org/survey/2026-06-04/better)
```

`survey_id` and `answer` are arbitrary slugs (regex `^[a-z0-9][a-z0-9_-]{0,63}$`). Defined entirely in the newsletter markdown — no pre-registration step.

## Data model

```sql
CREATE TABLE IF NOT EXISTS votes (
    ts        TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    survey_id VARCHAR   NOT NULL,
    answer    VARCHAR   NOT NULL,
    voter     VARCHAR   NOT NULL,  -- 32-char hex, anonymous
    PRIMARY KEY (survey_id, voter)
);
```

`voter` = `hex(sha256(ip || \0 || ua || \0 || daily_salt || \0 || survey_id))[:32]`.

- `daily_salt` = 32 random bytes generated in-memory at startup, rotated at midnight UTC and on every process restart. **Never written to disk.**
- Once rotation happens, yesterday's hashes cannot be reproduced — that's the Goatcounter-style property that makes this anonymous-by-construction.
- Including `survey_id` in the hash prevents cross-newsletter linkability.

Duplicate-click behaviour: `INSERT … ON CONFLICT (survey_id, voter) DO UPDATE SET answer = excluded.answer, ts = excluded.ts`. The reader is treated as changing their mind; last click wins.

## Routes

| Method | Path | Behaviour |
|---|---|---|
| GET    | `/survey/{survey_id}/{answer}` | Validate slugs → compute voter hash → upsert → 302 redirect to `/thanks?id={survey_id}` |
| HEAD   | `/survey/{survey_id}/{answer}` | 200 OK, no vote recorded. Suppresses Microsoft Safe Links / Gmail / Outlook pre-fetches. |
| GET    | `/thanks`                      | Embedded HTML page: "Thanks!" + link back to blog. |
| GET    | `/healthz`                     | 200 "ok" for monitoring. |

## Configuration (env vars only)

| Var | Default | Purpose |
|---|---|---|
| `SURVEY_DB_PATH`     | `/var/db/survey/votes.duckdb` | DuckDB file path |
| `SURVEY_HTTP_ADDR`   | `127.0.0.1:8080` | HTTP listen (Caddy proxies to this) |
| `SURVEY_QUACK_ADDR`  | `127.0.0.1:9494` | Quack listen (Caddy proxies to this) |
| `SURVEY_QUACK_TOKEN` | (required)       | Quack auth token |
| `SURVEY_BLOG_URL`    | `https://www.ssp.sh` | "Back to blog" link on thanks page |

## Deployment

Build on the FreeBSD host (avoids CGO cross-compile pain):

```sh
make deploy   # rsync source → ssh `go build` → atomic mv → service restart
```

Service runs under a dedicated `survey` user via FreeBSD `rc.d` and `daemon(8)`.

## Privacy posture

- No cookies, no JavaScript, no fingerprinting.
- IP and User-Agent are read, hashed with the in-memory daily salt, and discarded. Never written to disk.
- Daily salt rotation means past hashes cannot be reproduced — even from server logs.
- Access log records `survey_id` and `answer` only. No IP, no UA.
- GDPR posture: "anonymous aggregate statistics" under Recital 26. A one-line mention on the privacy page is appropriate.

## Operations

- Backup: `votes.duckdb` is one file. `scp` to Unraid nightly via cron.
- Storage: trivial (≈20 KB per newsletter at 4000 subs × 10% CTR).
- Failure mode: if the process dies, rc.d restarts it. If the file corrupts, restore the previous night's backup — worst case one issue is lost.

## What is NOT in this design (deliberate YAGNI)

- Comment box / free-text follow-up (was considered, dropped to keep "minimal").
- Pre-registration of surveys / questions table. The newsletter markdown is the source of truth for question text.
- Showing the running tally on the thanks page (avoids influencing late voters).
- Admin UI or auth-protected dashboard. Querying is `duckdb` → `ATTACH 'quack:…'`.
- Bot/scanner detection beyond the HEAD-200 trick.
