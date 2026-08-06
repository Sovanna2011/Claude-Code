# Deployment

## Prerequisites

| Component | Version |
|-----------|---------|
| .NET SDK | 10.0 |
| SQL Server | 2019 or later; verified against **SQL Server 2025** (17.0, RTM-CU7) and 2022 (Azure SQL and LocalDB also work) |
| dotnet-ef | `dotnet tool install --global dotnet-ef --version 10.*` (only to regenerate migrations) |

## 1. Database

Either let the API create it on start-up — `Database:MigrateOnStartup` defaults to `true` —
or apply the checked-in script, which is idempotent and safe to re-run:

```bash
sqlcmd -S localhost -Q "CREATE DATABASE SugarcanePlanning"
sqlcmd -S localhost -d SugarcanePlanning -i database/01_schema.sql
```

To regenerate the script after a model change:

```bash
dotnet ef migrations add <Name> -p src/SugarcanePlanning.Infrastructure -s src/SugarcanePlanning.Api -o Persistence/Migrations
database/generate-schema.sh          # regenerates database/01_schema.sql
```

Use the script rather than calling `dotnet ef migrations script` directly. The schema contains
filtered indexes, and SQL Server refuses to create those while `QUOTED_IDENTIFIER` is OFF —
which is how `sqlcmd` connects by default. `generate-schema.sh` prepends the required `SET`
options; without them the raw generator output fails on the first `CREATE INDEX` with
*Msg 1934* after creating a single table.

## 2. Configuration

`src/SugarcanePlanning.Api/appsettings.json`:

```jsonc
{
  "ConnectionStrings": {
    "SugarcanePlanning": "Server=…;Database=SugarcanePlanning;…"
  },
  "Jwt": {
    "Issuer": "SugarcanePlanning",
    "Audience": "SugarcanePlanningClient",
    "SigningKey": "at least 32 characters — override in production",
    "ExpiryMinutes": 480
  },
  "Planning": {
    "WorkOnSaturday": true,          // company working calendar
    "WorkOnSunday": false,
    "StandardWorkingHoursPerDay": 8,
    "AtRiskThreshold": 0.90,         // coverage below this is a shortage, not "at risk"
    "MaterialDeliveryLeadDays": 7
  },
  "Database": {
    "MigrateOnStartup": true,
    "SeedSampleData": true           // set false for a production tenant
  },
  "Cors": { "AllowedOrigins": [ "https://plan.example.com" ] }
}
```

**Never ship the default signing key.** Supply it from the environment or a secret store:

```bash
export Jwt__SigningKey="$(openssl rand -base64 48)"
export ConnectionStrings__SugarcanePlanning="Server=…;User Id=…;Password=…;Encrypt=True"
export Database__SeedSampleData=false
```

The Blazor client reads its API address from `wwwroot/appsettings.json`:

```json
{ "ApiBaseUrl": "https://api.plan.example.com/" }
```

The API and the client run on different origins, so `Cors:AllowedOrigins` must list the
client's address. The policy also exposes `Content-Disposition`; without that the browser hides
the header and every exported report is saved as `report.pdf` rather than its real name.

## 3. Build and publish

```bash
dotnet restore
dotnet build -c Release
dotnet test  -c Release                      # 173 tests

# Optional: also run the 17 tests that need a real SQL Server.
export SUGARCANE_TEST_SQLSERVER="Server=127.0.0.1,1433;User Id=sa;Password=…;TrustServerCertificate=True"
dotnet test  -c Release                      # 190 tests

dotnet publish src/SugarcanePlanning.Api    -c Release -o ./publish/api
dotnet publish src/SugarcanePlanning.Client -c Release -o ./publish/client
```

The client publishes to `./publish/client/wwwroot` as static files.

## 4. Hosting options

### IIS (Windows)

1. Install the **.NET 10 Hosting Bundle** and restart IIS.
2. Create an application pool with **No Managed Code**; point a site at `publish/api`.
3. Create a second site (or a sub-application) for `publish/client/wwwroot` and add the
   Blazor WebAssembly MIME types (`.wasm`, `.dat`, `.blat`, `.dll`, `.br`) plus a URL-rewrite
   rule sending unknown paths to `index.html`.
4. Give the app-pool identity `db_datareader`, `db_datawriter` and `db_ddladmin` (the last one
   only while `MigrateOnStartup` is true).

### Linux with systemd + nginx

```ini
# /etc/systemd/system/sugarcane-api.service
[Unit]
Description=Sugarcane Planning API
After=network.target

[Service]
WorkingDirectory=/opt/sugarcane/api
ExecStart=/usr/bin/dotnet /opt/sugarcane/api/SugarcanePlanning.Api.dll
Restart=always
RestartSec=10
User=www-data
Environment=ASPNETCORE_ENVIRONMENT=Production
Environment=ASPNETCORE_URLS=http://127.0.0.1:5100
EnvironmentFile=/etc/sugarcane/api.env

[Install]
WantedBy=multi-user.target
```

```nginx
server {
    listen 443 ssl http2;
    server_name plan.example.com;

    root /opt/sugarcane/client;
    location / { try_files $uri $uri/ /index.html; }

    location /api/ {
        proxy_pass         http://127.0.0.1:5100;
        proxy_set_header   Host $host;
        proxy_set_header   X-Real-IP $remote_addr;
        proxy_set_header   X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header   X-Forwarded-Proto $scheme;
    }

    # Blazor payloads compress very well.
    gzip on;
    gzip_types application/wasm application/octet-stream application/javascript text/css;
}
```

### Docker

```dockerfile
FROM mcr.microsoft.com/dotnet/aspnet:10.0 AS base
WORKDIR /app
EXPOSE 8080

FROM mcr.microsoft.com/dotnet/sdk:10.0 AS build
WORKDIR /src
COPY . .
RUN dotnet publish src/SugarcanePlanning.Api -c Release -o /app/publish

FROM base AS final
WORKDIR /app
COPY --from=build /app/publish .
ENV ASPNETCORE_URLS=http://+:8080
ENTRYPOINT ["dotnet", "SugarcanePlanning.Api.dll"]
```

```bash
docker build -t sugarcane-api .
docker run -d -p 8080:8080 \
  -e ConnectionStrings__SugarcanePlanning="Server=sql;Database=SugarcanePlanning;User Id=sa;Password=…;TrustServerCertificate=True" \
  -e Jwt__SigningKey="…" \
  sugarcane-api
```

### Azure

- **App Service** for the API (.NET 10 stack); set the connection string and `Jwt__SigningKey`
  in *Configuration*, or reference Key Vault.
- **Static Web Apps** or a storage account with CDN for the published client.
- **Azure SQL Database**; add the App Service outbound IPs to the firewall, or use a private
  endpoint and managed identity.

## 5. First run

The API applies pending migrations and, when `Database:SeedSampleData` is true, seeds the ten
roles, ten demo users (password `Planner#2026`) and a complete demo tenant — master data, an
approved projection, its activity plans and material requirements, live resource bookings and
part-recorded progress.

For a production tenant set `SeedSampleData` to `false`, then create the first administrator:

```http
POST /api/auth/users
Authorization: Bearer <token of an existing System Administrator>

{ "userName":"admin", "email":"admin@example.com", "password":"…",
  "fullName":"System Administrator", "companyId":1, "roles":["System Administrator"] }
```

Because the endpoint itself requires the `perm:administer` policy, bootstrap the very first
account by seeding once with `SeedSampleData=true`, changing the `admin` password, and then
removing the other demo users.

## 6. Health and monitoring

- `GET /health` — liveness plus an EF Core connectivity probe. Point the load balancer at it.
- `GET /swagger` — enabled in Development only.
- Logs go to the standard ASP.NET Core providers; `Microsoft.EntityFrameworkCore.Database.Command`
  is at `Warning` in production.
- The audit log (`/api/audit`, System Administrator only) is the functional trail: who changed
  what, when, from which address.

## 7. Operational notes

- **Backups.** Standard SQL Server backups. Nothing is stored outside the database.
- **Scaling.** The API is stateless — scale out behind a load balancer; JWTs need no affinity.
- **Stock interface.** `POST /api/materials/stock/sync` upserts stock positions from the ERP.
  The planning system never posts movements back.
- **Working calendar.** Changing `Planning:WorkOnSaturday`/`WorkOnSunday` changes every
  duration and requirement calculation; regenerate activity plans afterwards.
- **Booking and submission take application locks.** Resource bookings and projection submissions
  call `sp_getapplock` with `@LockOwner = 'Transaction'`, which is what stops two simultaneous
  requests double-booking a tractor or committing the same block. No grant is needed — `public`
  may take application locks — but the login must not be denied `EXECUTE` on `sp_getapplock`.
  A caller that waits longer than fifteen seconds is refused with `RESOURCE_BUSY`; if that ever
  shows up in the logs, look for a long-running transaction rather than raising the timeout.
  `sys.dm_tran_locks` with `resource_type = 'APPLICATION'` shows who holds what.
- **Transient-fault retries are deliberately off.** EF Core's `EnableRetryOnFailure` installs an
  execution strategy that refuses the explicit transactions used by the create, revise and
  plan-generation paths. If you need resilience on Azure SQL, route each of those operations
  through `Database.CreateExecutionStrategy()` and rebuild the change tracker per attempt —
  simply switching the option on will break them. `docs/architecture.md` has the detail.
