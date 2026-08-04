#!/bin/bash
# SessionStart hook for Claude Code on the web.
# Prepares the two modules in this repo (HR + PO Approval) so the C# backends
# and SAPUI5 front ends can be built/served during a web session:
#   - npm install for each UI5 app (npm registry is reachable directly)
#   - a best-effort .NET 8 SDK install + dotnet restore for each API
#
# Idempotent and non-interactive. Optional steps never fail the hook, so a
# session is never blocked if an upstream (e.g. the .NET SDK host) is
# unavailable under the environment's egress policy.
set -uo pipefail

# Web sessions only; do nothing on a local machine.
if [ "${CLAUDE_CODE_REMOTE:-}" != "true" ]; then
  exit 0
fi

ROOT="${CLAUDE_PROJECT_DIR:-$(pwd)}"
log() { echo "[session-start] $*"; }

# ---- SAPUI5 front ends -----------------------------------------------------
for app in "$ROOT/po-approval/frontend" "$ROOT/frontend"; do
  if [ -f "$app/package.json" ]; then
    log "npm install in ${app#"$ROOT"/}"
    (cd "$app" && npm install --no-audit --no-fund) \
      || log "npm install failed in ${app#"$ROOT"/} (continuing)"
  fi
done

# ---- .NET SDK (best effort) ------------------------------------------------
if ! command -v dotnet >/dev/null 2>&1 && [ ! -x "$HOME/.dotnet/dotnet" ]; then
  log "installing .NET SDK 8.0 (best effort)"
  if curl -fsSL https://dot.net/v1/dotnet-install.sh -o /tmp/dotnet-install.sh 2>/dev/null; then
    bash /tmp/dotnet-install.sh --channel 8.0 --install-dir "$HOME/.dotnet" --no-path >/dev/null 2>&1 \
      || log ".NET SDK install skipped (host may be blocked by egress policy)"
  else
    log "could not fetch dotnet-install.sh (continuing without .NET)"
  fi
fi

# Expose a locally installed SDK to this and future turns.
if [ -x "$HOME/.dotnet/dotnet" ]; then
  export DOTNET_ROOT="$HOME/.dotnet"
  export PATH="$HOME/.dotnet:$PATH"
  if [ -n "${CLAUDE_ENV_FILE:-}" ]; then
    echo 'export DOTNET_ROOT="$HOME/.dotnet"' >> "$CLAUDE_ENV_FILE"
    echo 'export PATH="$HOME/.dotnet:$PATH"' >> "$CLAUDE_ENV_FILE"
  fi
fi

# ---- Restore the C# backends if a SDK is available -------------------------
if command -v dotnet >/dev/null 2>&1; then
  for proj in "$ROOT/po-approval/backend/PoApproval.Api" "$ROOT/backend/HRModule.Api"; do
    if compgen -G "$proj/*.csproj" >/dev/null 2>&1; then
      log "dotnet restore ${proj#"$ROOT"/}"
      (cd "$proj" && dotnet restore) \
        || log "dotnet restore failed in ${proj#"$ROOT"/} (continuing)"
    fi
  done
else
  log ".NET SDK not available; C# build is skipped this session."
fi

log "setup complete"
exit 0
