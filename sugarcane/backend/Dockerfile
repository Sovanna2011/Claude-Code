# Build the API and ship it with the migrations it applies on start-up.
FROM golang:1.24-alpine AS build
WORKDIR /src
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

FROM alpine:3.20
RUN adduser -D -u 10001 farmarea
WORKDIR /app
COPY --from=build /out/api /app/api
COPY db/migrations /app/migrations
USER farmarea
EXPOSE 8080
ENTRYPOINT ["/app/api"]
