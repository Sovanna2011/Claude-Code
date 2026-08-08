#!/usr/bin/env python3
"""Capture everything the hub page shows, from the running API.

Nothing on the page is typed by hand: if the plan changes, this is re-run and
the page changes with it. A demonstration quoting figures the system no longer
produces teaches the reader to distrust the system.
"""
import json
import os
import urllib.request

B = os.environ.get("API", "http://localhost:8080/api/v1")


def call(path, token=None, method="GET", body=None):
    data = json.dumps(body).encode() if body is not None else None
    req = urllib.request.Request(B + path, data=data, method=method)
    req.add_header("Content-Type", "application/json")
    if token:
        req.add_header("Authorization", "Bearer " + token)
    with urllib.request.urlopen(req) as r:
        return json.load(r)


def tok(user):
    return call("/auth/dev-login", method="POST", body={"username": user})["accessToken"]


planner, executive = tok("planner"), tok("executive")

season = call("/seasons", planner)["value"][0]
versions = call(f"/seasons/{season['id']}/versions", planner)["value"]
by_code = {v["code"]: v for v in versions}

order = ["V1", "V2", "V3"]
ids = [by_code[c]["id"] for c in order if c in by_code]
matrix = call("/versions/compare-matrix", planner, "POST", {
    "versionIds": ids, "includeActual": True, "dimension": "DATE"})

# Regenerating is idempotent with replace, and it is the only way to read the
# warnings the generator raised: they belong to the run, not to a stored row.
shortfalls = {}
for code in order:
    if code not in by_code:
        continue
    run = call(f"/versions/{by_code[code]['id']}/generate", planner, "POST",
               {"replace": True})
    short = [w for w in (run.get("warnings") or [])
             if w["code"] == "REMELT_SUPPLY_SHORT"]
    shortfalls[code] = {
        "rawExpected": run["summary"]["rawSugarExpectedTons"],
        "finished": run["summary"]["finishedGoodsTons"],
        "remeltNeed": run["summary"]["remeltInputTons"],
        "detail": short[0]["detail"] if short else None,
    }

dash = call(f"/dashboard?seasonId={season['id']}", executive)
profile = call(f"/versions/{by_code['V1']['id']}/crushing-profile", planner)
supply = call(f"/versions/{by_code['V1']['id']}/supply", planner)

out = {
    "season": {k: season[k] for k in ("code", "name", "startDate", "endDate", "plannedDays")},
    "versions": [
        {"code": v["code"], "description": v["description"], "planType": v["planType"],
         "status": v["status"], "versionNo": v["versionNo"]}
        for v in sorted(versions, key=lambda v: v["versionNo"])
    ],
    "matrix": {
        "columns": [
            {"code": c["version"]["code"], "description": c["version"]["description"],
             "planType": c["version"]["planType"], "series": c["series"],
             "isBaseline": c["isBaseline"]}
            for c in matrix["columns"]
        ],
        "totals": [
            {"measure": t["measure"], "values": t["values"], "deltas": t["deltas"],
             "deltaPcts": t["deltaPcts"]}
            for t in matrix["totals"]
        ],
        "assumptions": [
            {"key": a["key"], "label": a["label"], "values": a["values"],
             "deltas": a["deltas"]}
            for a in matrix["assumptions"]
        ],
    },
    "headline": {
        "planVersion": dash["planVersion"]["code"],
        "planStatus": dash["planVersion"]["status"],
        "caneTarget": dash["cane"]["seasonTargetTons"],
        "crushed": dash["cane"]["cumulativeActualTons"],
        "recoveryTarget": dash["rawSugar"]["targetRecoveryPct"],
        "recoveryActual": dash["rawSugar"]["actualRecoveryPct"],
        "alerts": len(dash["alerts"]),
        "asOf": dash["asOf"],
        "plannedEnd": dash["cane"]["plannedEndDate"],
        "campaignDays": season["plannedDays"],
        "crushingDays": profile["preview"]["crushingDays"],
        "cleaningDays": profile["preview"]["cleaningDays"],
        "plateau": profile["preview"]["plateauRateTons"],
        "coverage": supply["reconciliation"]["coveragePct"],
        "committed": supply["reconciliation"]["committedTons"],
        "sources": len(supply["entries"]),
        "supplyWarnings": len(supply["warnings"]),
    },
    "shortfalls": shortfalls,
    "products": [
        {"code": p["productCode"], "name": p["productName"],
         "target": p["targetTons"], "actual": p["actualTons"]}
        for p in dash["products"]
    ],
    "alerts": [
        {"severity": a["severity"], "title": a["title"], "detail": a["detail"]}
        for a in dash["alerts"]
    ],
}

path = os.environ.get("OUT", "hub-data.json")
with open(path, "w") as f:
    json.dump(out, f, indent=1)
print(json.dumps(out["headline"], indent=1))
print("versions:", [(v["code"], v["planType"], v["status"]) for v in out["versions"]])
