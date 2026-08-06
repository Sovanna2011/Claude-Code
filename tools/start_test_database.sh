#!/usr/bin/env bash
#
# Brings up a SQL Server 2025 container, installs the ERP schema into it, and
# prints the connection string the integration tests want.
#
#   ./tools/start_test_database.sh          # start (or restart) and install
#   ./tools/start_test_database.sh --reset  # drop the database and reinstall
#
# The password is a local development one and is meant to be: this container
# listens on localhost and holds nothing but sample data.

set -euo pipefail

CONTAINER=erps4-sql
IMAGE=mcr.microsoft.com/mssql/server:2025-latest
PASSWORD='ErpS4!Local#2026'
PORT=1433
SQLCMD=/opt/mssql-tools18/bin/sqlcmd
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

reset=false
[[ "${1:-}" == "--reset" ]] && reset=true

if ! docker info >/dev/null 2>&1; then
    echo "The Docker daemon is not running." >&2
    exit 1
fi

case "$(docker inspect -f '{{.State.Status}}' "$CONTAINER" 2>/dev/null || echo missing)" in
    running) echo "$CONTAINER is already running." ;;
    missing)
        echo "Creating $CONTAINER from $IMAGE ..."
        docker run -d --name "$CONTAINER" \
            -e ACCEPT_EULA=Y \
            -e "MSSQL_SA_PASSWORD=$PASSWORD" \
            -e MSSQL_PID=Developer \
            -p "$PORT:1433" "$IMAGE" >/dev/null
        ;;
    *)
        echo "Starting $CONTAINER ..."
        docker start "$CONTAINER" >/dev/null
        ;;
esac

# The engine accepts connections a good while after the container reports up,
# so wait for a query to succeed rather than for a fixed number of seconds.
printf 'Waiting for SQL Server '
for _ in $(seq 1 60); do
    if docker exec "$CONTAINER" "$SQLCMD" -S localhost -U sa -P "$PASSWORD" -C \
        -Q 'SELECT 1' >/dev/null 2>&1; then
        echo ' ready.'
        break
    fi
    printf '.'
    sleep 2
done

if $reset; then
    docker exec "$CONTAINER" "$SQLCMD" -S localhost -U sa -P "$PASSWORD" -C -Q \
        "IF DB_ID('ErpS4') IS NOT NULL
         BEGIN
             ALTER DATABASE [ErpS4] SET SINGLE_USER WITH ROLLBACK IMMEDIATE;
             DROP DATABASE [ErpS4];
         END" >/dev/null
    echo "Dropped ErpS4."
fi

# A fresh directory each time: docker cp cannot overwrite the previous copy,
# whose files belong to the mssql user inside the container.
TARGET="/tmp/s4hana-$(date +%s)"
docker cp "$ROOT/database/s4hana" "$CONTAINER:$TARGET" >/dev/null

echo "Installing the schema ..."
docker exec -w "$TARGET" "$CONTAINER" "$SQLCMD" \
    -S localhost -U sa -P "$PASSWORD" -C -i run_all.sql -b \
    | grep -iE 'Msg |complete|loaded' || true

CONNECTION="Server=127.0.0.1,$PORT;Database=ErpS4;User Id=sa;Password=$PASSWORD;TrustServerCertificate=True;Encrypt=False"

cat <<EOF

Ready. To run the integration tests:

    export ERPS4_TEST_CONNECTION='$CONNECTION'
    dotnet test backend/ErpS4.IntegrationTests
EOF
