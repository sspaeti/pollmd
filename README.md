# minimal-newsletter-survey

A ~200-line Go service that records anonymous reader ratings from newsletter
links into a [DuckDB](https://ssp.sh/brain/duckdb) file. Per-newsletter, per-answer, no cookies, no JS.
Query the results from your laptop over Quack.

Design doc: [`docs/superpowers/specs/2026-06-04-newsletter-survey-design.md`](docs/superpowers/specs/2026-06-04-newsletter-survey-design.md).

## What it looks like in a newsletter

```markdown
What did you think of today's newsletter?

[Awesome!](https://ti.sspaeti.duckdns.org/survey/2026-06-04/awesome)
[Pretty Good](https://ti.sspaeti.duckdns.org/survey/2026-06-04/good)
[Could be better](https://ti.sspaeti.duckdns.org/survey/2026-06-04/better)
```

Each click records one vote and redirects to a "Thanks!" page. The next
newsletter can use entirely different `survey_id` and `answer` slugs without
any code or schema change.

## How votes are deduplicated

`voter = sha256(ip || ua || daily_salt || survey_id)[:16]` (hex).

- The daily salt is 32 random bytes generated in memory at startup, rotated
  every midnight UTC, and regenerated on every process restart. It is
  **never written to disk**.
- After rotation, yesterday's hashes can no longer be reproduced from logs.
- Including `survey_id` in the hash means the same reader produces different
  hashes for different newsletters, so cross-issue tracking is impossible.

If the same reader clicks twice on the same newsletter (e.g. Awesome, then
Good), the second click replaces the first — last vote wins.

## One-time FreeBSD server setup

Run on `ti.sspaeti.duckdns.org` as root (or with `doas`):

```sh
# packages (duckdb is already installed; we need go, rsync, and libduckdb headers)
pkg install -y go rsync

# Verify libduckdb is present (the build tag duckdb_use_lib will link against it).
# The duckdb FreeBSD port installs the shared library and headers under /usr/local.
ls /usr/local/lib/libduckdb.so /usr/local/include/duckdb.h

# service user (no login, no home)
pw useradd survey -d /nonexistent -s /usr/sbin/nologin

# directories
mkdir -p /var/db/survey /var/log/survey /usr/local/etc/survey \
         /home/sspaeti/survey-src
chown survey:survey /var/db/survey /var/log/survey
chown sspaeti:sspaeti /home/sspaeti/survey-src

# env file
cp deploy/survey.env.example /usr/local/etc/survey/survey.env
TOKEN=$(head -c 32 /dev/urandom | base64)
sed -i '' "s|CHANGE_ME_TO_BASE64_TOKEN|${TOKEN}|" /usr/local/etc/survey/survey.env
chmod 600 /usr/local/etc/survey/survey.env
chown survey:survey /usr/local/etc/survey/survey.env
echo "Quack token: ${TOKEN}"   # save this; you'll need it on your laptop

# rc.d service
cp deploy/survey.rc /usr/local/etc/rc.d/survey
chmod +x /usr/local/etc/rc.d/survey
sysrc survey_enable=YES

# Caddy
cat deploy/Caddyfile.snippet >> /usr/local/etc/caddy/Caddyfile
service caddy reload
```

## Deploy

From your laptop, in this directory:

```sh
make deploy
```

This rsyncs the source to the FreeBSD host, builds there with
`-tags=duckdb_use_lib` (dynamically linked against your system libduckdb 1.5.3
since duckdb-go-bindings ships no pre-built FreeBSD library), atomically swaps
the binary, and restarts the service.

Other targets: `make test`, `make logs`, `make status`, `make query`.

## Query from your laptop

```sh
duckdb
```

```sql
CREATE SECRET (TYPE quack, TOKEN '<paste your token>');
ATTACH 'quack:quack.sspaeti.duckdns.org' AS s;

-- One newsletter's results
FROM s.votes
WHERE survey_id = '2026-06-04'
GROUP BY answer
ORDER BY count(*) DESC;

-- All-time rolling tally
SELECT survey_id, answer, count(*) AS votes
FROM s.votes
GROUP BY ALL
ORDER BY survey_id DESC, votes DESC;
```

You can also fall back to `ssh ti.sspaeti.duckdns.org "duckdb /var/db/survey/votes.duckdb -c 'FROM votes'"` if Quack is ever misbehaving.

## Privacy

- No cookies, no JavaScript, no fingerprinting.
- IP and User-Agent are read on each request, fed into the voter hash, and
  immediately discarded. Nothing identifying is persisted.
- The daily salt rotation means past hashes cannot be reproduced — even with
  access to server logs.
- Access logs record only `survey_id` and `answer`.

## Layout

```
.
├── cmd/survey/main.go             # entrypoint, env wiring
├── internal/
│   ├── server/server.go           # routes, click handler, X-Forwarded-For
│   ├── server/thanks.html         # embedded thanks page
│   ├── store/store.go             # DuckDB open, schema, quack_serve, upsert
│   └── voter/hash.go              # daily salt + voter hash
├── deploy/
│   ├── survey.rc                  # FreeBSD rc.d service script
│   ├── survey.env.example         # env-var template
│   └── Caddyfile.snippet          # reverse proxy + TLS
├── Makefile
└── go.mod
```
