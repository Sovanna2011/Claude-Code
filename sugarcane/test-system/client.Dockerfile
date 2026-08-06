# The Blazor client publishes to plain static files. In a real deployment those sit behind IIS,
# nginx or a CDN (see docs/deployment.md); here they are served by the small StaticHost project,
# because this environment can reach Microsoft's container registry but not Docker Hub.
# Build context is the solution root (sugarcane/).

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

COPY SugarcanePlanning.slnx ./
COPY src/SugarcanePlanning.Domain/*.csproj     src/SugarcanePlanning.Domain/
COPY src/SugarcanePlanning.Contracts/*.csproj  src/SugarcanePlanning.Contracts/
COPY src/SugarcanePlanning.Client/*.csproj     src/SugarcanePlanning.Client/
COPY test-system/StaticHost/*.csproj           test-system/StaticHost/
RUN dotnet restore src/SugarcanePlanning.Client/SugarcanePlanning.Client.csproj \
    && dotnet restore test-system/StaticHost/StaticHost.csproj

COPY src/ src/
COPY test-system/StaticHost/ test-system/StaticHost/
RUN dotnet publish src/SugarcanePlanning.Client/SugarcanePlanning.Client.csproj \
        -c Release -o /app/client --no-restore \
    && dotnet publish test-system/StaticHost/StaticHost.csproj \
        -c Release -o /app/host --no-restore

FROM mcr.microsoft.com/dotnet/aspnet:10.0 AS final
RUN apt-get update \
    && apt-get install -y --no-install-recommends curl \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY --from=build /app/host ./
COPY --from=build /app/client/wwwroot ./wwwroot
ENV ASPNETCORE_URLS=http://+:80
EXPOSE 80
ENTRYPOINT ["dotnet", "StaticHost.dll"]
