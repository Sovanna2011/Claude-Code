#!/usr/bin/env python3
"""Capture the whole system, per user, from the running API.

The artifact cannot reach a Go server - a published page may not call another
host - so it carries a snapshot instead. This script is what makes the snapshot
real: every screen's rows and every refusal below came back from the running
service, including the problem-detail bodies. Nothing is written by hand.
"""
import json
import os
import urllib.error
import urllib.request

B = os.environ.get("API", "http://localhost:8080/api/v1")
# The last day the seed records an actual for, with SEED_ACTUAL_DAYS=14.
SEEDED_AS_OF = "2026-12-14"
OUT = os.environ.get("OUT", "system-data.json")

# The accounts the service itself offers in development mode, taken from its own
# roster rather than invented for the page: config.defaultDevUsers.
USERS = [
    ("planner", "Plan the season"),
    ("approver", "Approve and release"),
    ("supervisor", "Run the shift"),
    ("weighbridge", "The cane gate"),
    ("warehouse", "Move the stock"),
    ("shipping", "Ship to customers"),
    ("quality", "The laboratory"),
    ("controller", "Cost the season"),
    ("engineer", "Maintenance"),
    ("executive", "Read the season"),
    ("auditor", "Read the trail"),
    ("admin", "Administer"),
    ("interface", "Machine account"),
]


def raw(path, token=None, method="GET", body=None):
    data = json.dumps(body).encode() if body is not None else None
    req = urllib.request.Request(B + path, data=data, method=method)
    req.add_header("Content-Type", "application/json")
    if token:
        req.add_header("Authorization", "Bearer " + token)
    try:
        with urllib.request.urlopen(req) as r:
            return r.status, json.load(r)
    except urllib.error.HTTPError as e:
        payload = e.read().decode()
        try:
            return e.code, json.loads(payload)
        except ValueError:
            return e.code, {"detail": payload[:300]}


def get(path, token):
    status, body = raw(path, token)
    if status != 200:
        raise SystemExit(f"{path} -> {status}: {body}")
    return body


tokens = {}
principals = {}
for name, _ in USERS:
    status, body = raw("/auth/dev-login", method="POST", body={"username": name})
    if status != 200:
        raise SystemExit(f"login {name}: {status} {body}")
    tokens[name] = body["accessToken"]
    principals[name] = body

P, S, K = tokens["planner"], tokens["supervisor"], tokens["warehouse"]
L, E = tokens["quality"], tokens["executive"]

season = get("/seasons", P)["value"][0]
factory = get("/master/factories", P)["value"][0]
sid, fid = season["id"], factory["id"]
versions = get(f"/seasons/{sid}/versions", P)["value"]
by_code = {v["code"]: v for v in versions}
v1 = by_code["V1"]["id"]

# The write probes at the end of this script really do write. Run it twice
# against one database and the second run's *reads* see the first run's probe
# rows: the fortnight of actuals grows a lone day with a six-day hole before it,
# and the dashboard reports the mill as having stopped for a week. That happened,
# and the screenshot looked plausible enough to nearly ship. So the run refuses
# rather than quietly producing a season nobody planned.
dash = get(f"/dashboard?seasonId={sid}", E)
if dash["asOf"] != SEEDED_AS_OF:
    raise SystemExit(
        f"the actuals end {dash['asOf']}, not the seeded {SEEDED_AS_OF}: this "
        "database has been probed already. Re-seed it and run this once.")
detail = get(f"/versions/{v1}", P)
profile = get(f"/versions/{v1}/crushing-profile", P)
supply = get(f"/versions/{v1}/supply", P)

matrix = raw("/versions/compare-matrix", P, "POST", {
    "versionIds": [by_code[c]["id"] for c in ("V1", "V2", "V3") if c in by_code],
    "includeActual": True, "dimension": "DATE"})[1]


def rows(path, token, limit=None):
    """Fetch a list whole.

    An earlier version named the fields each screen wanted, and every name that
    did not match came back as a null - a screen full of blanks that looked like
    missing data rather than a wrong guess. These collections are a handful of
    rows each, so they are kept entire and the page picks from what is there.
    """
    body = get(path, token)
    items = body.get("value") if isinstance(body, dict) else body
    items = items or []
    return items[:limit] if limit else items


data = {
    "capturedFrom": "the running service, over its own HTTP API",
    "season": season,
    "factory": {k: factory.get(k) for k in ("id", "code", "name", "timezone")},
    "principals": principals,
    "users": [{"username": u, "purpose": t, "principal": principals[u]}
              for u, t in USERS],

    "versions": [{k: v.get(k) for k in
                  ("id", "code", "description", "planType", "status", "versionNo")}
                 for v in sorted(versions, key=lambda v: v["versionNo"])],
    "assumptions": detail.get("assumptions", []),
    "productMix": detail.get("productMix", []),
    "crushing": {
        "preview": {k: v for k, v in profile["preview"].items()
                    if not isinstance(v, list)},
        "profile": profile["profile"],
        "daily": profile["preview"].get("daily", [])[:400],
    },
    "supply": {
        "reconciliation": supply["reconciliation"],
        "entries": supply["entries"],
        "warnings": supply["warnings"],
    },
    "dashboard": {
        "asOf": dash["asOf"],
        "planVersion": {k: dash["planVersion"][k] for k in ("code", "status", "planType")},
        "cane": dash["cane"],
        "rawSugar": dash["rawSugar"],
        "products": dash["products"],
        "storage": dash["storage"],
        "shipments": dash["shipments"],
        "downtime": dash["downtime"],
        "alerts": dash["alerts"],
        "caneTrend": dash["caneTrend"],
        "caneRollingAverage": dash["caneRollingAverage"],
        "recoveryTrend": dash["recoveryTrend"],
        "productTrend": dash["productTrend"],
    },
    "matrix": matrix,

    "downtime": rows(f"/downtime?factoryId={fid}", S),
    "orders": rows(f"/production-orders?factoryId={fid}", S),
    "documents": rows(f"/inventory/documents?factoryId={fid}", K),
    "stock": rows("/stock", K),
    "samples": rows(f"/quality/samples?factoryId={fid}", L),
    "holds": rows("/quality/holds", L),
    "materials": get(f"/versions/{v1}/material-requirements", P),

    # The governance screens. Each is read by the one account that may: showing
    # an empty screen to an auditor who is allowed in would misreport the system
    # as having no trail.
    "costing": rows("/costing/runs", tokens["controller"]),
    "audit": rows("/audit?$top=40", tokens["auditor"]),
    "events": rows("/integration/events?$top=40", tokens["auditor"]),

    # Lookups, so a screen can show "Refined sugar" where a row carries an id.
    # Without these the tables read as columns of UUIDs.
    "master": {
        name: rows(f"/master/{name}", P)
        for name in ("products", "warehouses", "reason-codes", "packaging-types",
                     "shipment-channels", "cane-sources")
    },
}

# --- the permission matrix, measured rather than described -------------------
#
# Reads come first and change nothing. The write probes genuinely write, so this
# section runs last and the database is re-seeded before any re-capture - a
# probe that lied about being harmless would be worse than no probe.
#
# V3 is the guinea pig for the workflow probes so V1 stays the draft budget the
# rest of the demonstration shows.
actual_id = next(v["id"] for v in versions if v["planType"] == "ACTUAL")
v3 = by_code["V3"]["id"] if "V3" in by_code else v1
cane_row = {"rows": [{"versionId": actual_id, "factoryId": fid,
                      "businessDate": "2026-12-20", "series": "ACTUAL",
                      "caneAvailable": "18000", "caneDelivered": "18000",
                      "caneAccepted": "18000", "caneCrushed": "18000",
                      "availableHours": "24", "crushRateTph": "750"}]}

PROBES = [
    ("See the executive overview", "GET", f"/dashboard?seasonId={sid}", None),
    ("See the season's plans", "GET", f"/seasons/{sid}/versions", None),
    ("See stock on hand", "GET", "/stock", None),
    ("See quality samples", "GET", f"/quality/samples?factoryId={fid}", None),
    ("See the costing runs", "GET", "/costing/runs", None),
    ("See the audit trail", "GET", "/audit", None),
    ("See interface events", "GET", "/integration/events", None),
    ("Record a day of cane", "POST", f"/versions/{actual_id}/cane", cane_row),
    ("Generate a plan", "POST", f"/versions/{v3}/generate", {"replace": True}),
]

matrix_rows = []
for label, method, path, body in PROBES:
    cells = {}
    for name, _ in USERS:
        status, resp = raw(path, tokens[name], method, body)
        cells[name] = {
            "status": status,
            "allowed": status < 400,
            "detail": (resp.get("detail") or resp.get("title") or "")[:200]
            if isinstance(resp, dict) else "",
        }
    matrix_rows.append({"action": label, "method": method,
                        "path": path.split("?")[0], "cells": cells})

data["permissions"] = matrix_rows

with open(OUT, "w") as f:
    json.dump(data, f, separators=(",", ":"))

print("wrote", OUT, os.path.getsize(OUT), "bytes")
for r in matrix_rows:
    print("  %-24s %s" % (r["action"], " ".join(
        ("%s:%d" % (u[:4], r["cells"][u]["status"])) for u, _ in USERS)))
