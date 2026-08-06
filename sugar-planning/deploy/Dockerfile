# Build the API and the migration tool.
#
# The frontend is copied in as static files so a single container can serve
# both, which is the simplest on-premises install: one image plus a database.
FROM golang:1.25-alpine AS build

WORKDIR /src
# Dependencies first, so a source-only change does not re-download them.
COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ ./
# CGO off gives a static binary that runs on a distroless or scratch base.
# The build stamps the version so /healthz reports what is actually running.
# The running version is reported by /healthz from the APP_VERSION environment
# variable, so it does not need to be linked into the binary.
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" \
        -o /out/server ./cmd/server && \
    CGO_ENABLED=0 GOOS=linux go build -trimpath \
        -ldflags="-s -w" \
        -o /out/migrate ./cmd/migrate

# ---------------------------------------------------------------------------

FROM alpine:3.20

# ca-certificates for the OIDC provider; tzdata because business dates are
# calendar days in the factory time zone and the container needs the database.
RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -g 10001 -S sugarplan && \
    adduser -u 10001 -S -G sugarplan -h /app sugarplan

WORKDIR /app
COPY --from=build /out/server /app/server
COPY --from=build /out/migrate /app/migrate
COPY --chown=sugarplan:sugarplan frontend/webapp /app/webapp

# Never root.
USER 10001:10001

ENV HTTP_ADDR=:8080 \
    HTTP_STATIC_DIR=/app/webapp \
    FACTORY_TIMEZONE=Asia/Phnom_Penh

EXPOSE 8080

# Readiness, not liveness: this checks the database too, which is what an
# orchestrator should gate traffic on.
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -qO- http://127.0.0.1:8080/readyz >/dev/null || exit 1

ENTRYPOINT ["/app/server"]
