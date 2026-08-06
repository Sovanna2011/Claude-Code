# Build context is the solution root (sugarcane/), so the Api project can restore its
# project references. See test-system/docker-compose.yml.

FROM mcr.microsoft.com/dotnet/sdk:10.0 AS build
WORKDIR /src

# Behind a TLS-intercepting proxy the public NuGet feed presents a certificate the SDK image
# does not trust, and restore fails with NU1301 / UntrustedRoot. Any *.crt dropped into
# test-system/ca/ is installed here, before restore. The directory ships with only a .keep
# file, so on a normal network this layer installs nothing and costs one command.
COPY test-system/ca/ /usr/local/share/ca-certificates/extra/
RUN update-ca-certificates

# Only the build needs the proxy; the runtime image below must not inherit it.
ARG HTTPS_PROXY=""
ENV HTTPS_PROXY=${HTTPS_PROXY}

# Restore from the project files alone: this layer is reused on every rebuild in which no
# dependency changed, which is most of them.
COPY SugarcanePlanning.slnx ./
COPY src/SugarcanePlanning.Domain/*.csproj          src/SugarcanePlanning.Domain/
COPY src/SugarcanePlanning.Contracts/*.csproj       src/SugarcanePlanning.Contracts/
COPY src/SugarcanePlanning.Application/*.csproj     src/SugarcanePlanning.Application/
COPY src/SugarcanePlanning.Infrastructure/*.csproj  src/SugarcanePlanning.Infrastructure/
COPY src/SugarcanePlanning.Api/*.csproj             src/SugarcanePlanning.Api/
RUN dotnet restore src/SugarcanePlanning.Api/SugarcanePlanning.Api.csproj

COPY src/ src/
RUN dotnet publish src/SugarcanePlanning.Api/SugarcanePlanning.Api.csproj \
    -c Release -o /app/publish --no-restore

FROM mcr.microsoft.com/dotnet/aspnet:10.0 AS final
# curl is here only so the compose health check can call /health; the runtime image ships
# without any HTTP client of its own.
RUN apt-get update \
    && apt-get install -y --no-install-recommends curl \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY --from=build /app/publish .
EXPOSE 8080
ENTRYPOINT ["dotnet", "SugarcanePlanning.Api.dll"]
