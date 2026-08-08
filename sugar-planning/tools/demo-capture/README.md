# Demonstration capture

These two scripts read the running service over its own HTTP API and write the
JSON that the demonstration pages are built from. Nothing on those pages is
typed by hand: when the plan changes, these are re-run and the pages change with
them. A demonstration quoting last month's tonnage teaches its reader to
distrust the system rather than the page.

| Script | Writes | Feeds |
| --- | --- | --- |
| `capture.py` | every screen, per account, plus the permission matrix | the working sign-in system |
| `hub.py` | the season headline and the version comparison | the demonstration hub |

## Running them

Both take `API` (default `http://localhost:8080/api/v1`) and `OUT` from the
environment. They need a server in development auth mode with the full
demonstration data:

```sh
export DATABASE_URL=postgres://…  AUTH_MODE=dev  AUTH_DEV_SECRET=…
export SEED_DEMO=true SEED_EXECUTION=true SEED_ACTUAL_DAYS=14
migrate up && server &
OUT=system-data.json python3 capture.py
```

## Run `capture.py` once per seed

Its last section is a permission matrix, and it is measured rather than
described: every account signs in and makes every request, and the status codes
that come back are what the page shows. Two of those requests are writes, so
they really do write.

Run it twice against one database and the second run's *reads* see the first
run's probe rows. The fortnight of actuals grows a lone extra day with a
six-day hole before it, and the executive overview reports the mill as having
stopped for a week — a plausible-looking screenshot of a season nobody planned.
That happened, so the script now checks the last recorded actual against the
seeded date and refuses rather than producing it quietly. Re-seed between runs.
