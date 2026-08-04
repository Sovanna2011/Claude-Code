# PO Approval API - multi-stage build.
# Build context is this directory (backend/PoApproval.Api).

FROM mcr.microsoft.com/dotnet/sdk:8.0 AS build
WORKDIR /src

# Restore first (layer cached until the project file changes).
COPY PoApproval.Api.csproj ./
RUN dotnet restore

# Build & publish.
COPY . ./
RUN dotnet publish -c Release -o /app --no-restore

FROM mcr.microsoft.com/dotnet/aspnet:8.0 AS runtime
WORKDIR /app
COPY --from=build /app ./
EXPOSE 5000
ENV ASPNETCORE_URLS=http://+:5000
ENTRYPOINT ["dotnet", "PoApproval.Api.dll"]
