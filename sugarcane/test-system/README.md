# Test system

A complete, disposable instance of the planning system — SQL Server, the API and the Blazor
client — seeded with the demo tenant, so the whole application can be clicked through without
installing anything but Docker.

```bash
cd sugarcane/test-system
docker compose up -d --build      # first run pulls the images and seeds the database
docker compose ps                 # wait until api and client report "healthy"
```

| Service | Address | Notes |
|---------|---------|-------|
| Blazor client | <http://localhost:5150> | start here |
| API | <http://localhost:5100> | Swagger at `/swagger`, liveness at `/health` |
| SQL Server 2025 | `127.0.0.1,11433` | user `sa`, password `Sugarcane#2026dev` |

Sign in with any of the demo accounts — password `Planner#2026` for all of them. `planner`
builds projections, `machinery` schedules tractors, `director` approves, `admin` sees the audit
log; the full list is in the main [README](../README.md).

```bash
docker compose logs -f api        # follow start-up, migration and seeding
docker compose restart api        # after changing an environment variable
docker compose up -d --build api  # after changing code
docker compose down               # stop, keep the database
docker compose down -v            # stop and delete the database as well
```

The first start takes a couple of minutes: SQL Server initialises, then the API applies the
migration and seeds an estate with 3 farms, 6 zones and 24 blocks, a 2026 season, 19 planting
activities, 10 tractors, 14 implements, an approved projection and its activity plans, material
requirements, live bookings and part-recorded progress. Later starts reuse the `sqldata` volume
and come up in seconds.

## What it is not

This is a **test** system. The sa password and the JWT signing key are fixed and public, sample
data seeding is on, and Swagger is exposed — so the ports are bound to `127.0.0.1` and must stay
that way. For a real deployment follow [docs/deployment.md](../docs/deployment.md), which covers
IIS, systemd + nginx, Docker and Azure with secrets supplied from the environment.

The client is served here by the small `StaticHost` project rather than by nginx, because this
environment can reach Microsoft's container registry but not Docker Hub. It is not part of the
solution and is never deployed with the product: in production the published client is plain
static output behind IIS, nginx or a CDN. Whatever serves it must send `no-store` for
`index.html` and revalidate the rest — the published asset names are stable, so caching them
outright would keep serving the previous deployment's code.

## Behind a proxy that intercepts TLS

If `docker compose build` fails with `NU1301 … UntrustedRoot`, the NuGet feed is being presented
a certificate signed by an internal CA that the .NET SDK image does not trust. Drop that CA in
PEM form into [`ca/`](ca/) and rebuild; anything there is installed before restore runs. The
build also forwards `HTTPS_PROXY` and runs on the host network, so a proxy listening on
`127.0.0.1` is reachable from the build container.

## Layout

| File | Purpose |
|------|---------|
| `docker-compose.yml` | the three services, their health checks and start-up order |
| `api.Dockerfile` | publishes the API onto the ASP.NET runtime image |
| `client.Dockerfile` | publishes the Blazor client and serves it with `StaticHost` |
| `StaticHost/` | ~40-line static file server: correct MIME types, SPA fallback, cache headers |
| `ca/` | optional extra root certificates for the build |
