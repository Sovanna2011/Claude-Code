#!/usr/bin/env python3
"""Check the generated DDL before it ever reaches a server.

Runs three passes over database/s4hana:

1. T-SQL parse of every batch (sqlglot, tsql dialect) - catches syntax errors.
2. Referential checks - every foreign key column exists, and every referenced
   column list is exactly the primary key or the unique business key of the
   target table.
3. Identifier checks - unique constraint names, names within 128 characters.

    python3 tools/validate_generated_sql.py
"""

from __future__ import annotations

import collections
import pathlib
import re
import sys

sys.path.insert(0, str(pathlib.Path(__file__).resolve().parent))

from catalogue_model import ROOT, load_tables, resolve_foreign_keys  # noqa: E402

SQL_DIR = ROOT / "database" / "s4hana"
CONSTRAINT = re.compile(r"CONSTRAINT \[([A-Za-z0-9_]+)\]")
GO_SPLIT = re.compile(r"^GO\s*$", re.MULTILINE)


def parse_batches() -> list[str]:
    try:
        import sqlglot
    except ImportError:
        print("  sqlglot not installed - skipping the parse pass")
        return []
    problems: list[str] = []
    for path in sorted(SQL_DIR.glob("*.sql")):
        if path.name == "run_all.sql":
            continue  # sqlcmd :r directives, not T-SQL
        text = path.read_text(encoding="utf-8-sig")
        for number, batch in enumerate(GO_SPLIT.split(text), start=1):
            if not batch.strip():
                continue
            try:
                sqlglot.parse(batch, dialect="tsql")
            except Exception as error:  # noqa: BLE001 - report, do not raise
                first = batch.strip().splitlines()[0][:90]
                problems.append(f"{path.name} batch {number}: {error} | {first}")
    return problems


def check_references() -> list[str]:
    tables = load_tables()
    lookup = {t.full_name: t for t in tables}
    foreign_keys, _ = resolve_foreign_keys(tables)
    problems: list[str] = []
    for fk in foreign_keys:
        for column in fk.columns:
            if fk.table.field(column) is None:
                problems.append(f"{fk.name}: {fk.table.full_name} has no [{column}]")
        target = lookup[f"{fk.target_schema}.{fk.target_table}"]
        pk = [f.name for f in target.primary_key]
        ak = [f.name for f in target.alternate_key]
        if fk.target_columns not in (pk, ak):
            problems.append(
                f"{fk.name}: {target.full_name}{fk.target_columns} is neither the "
                f"primary key {pk} nor the business key {ak}"
            )
        types_local = [fk.table.field(c).sql_type for c in fk.columns]  # type: ignore[union-attr]
        types_target = [target.field(c).sql_type for c in fk.target_columns]  # type: ignore[union-attr]
        if types_local != types_target:
            problems.append(
                f"{fk.name}: type mismatch {types_local} vs {types_target}"
            )
    return problems


def check_identifiers() -> list[str]:
    problems: list[str] = []
    seen: collections.Counter[str] = collections.Counter()
    for path in sorted(SQL_DIR.glob("*.sql")):
        for name in CONSTRAINT.findall(path.read_text(encoding="utf-8-sig")):
            seen[name] += 1
            if len(name) > 128:
                problems.append(f"identifier longer than 128 characters: {name}")
    # DEFAULT constraints appear once per table, PK/UQ/FK names must be unique.
    for name, count in seen.items():
        if count > 1:
            problems.append(f"constraint name used {count} times: {name}")
    return problems


def main() -> None:
    failures = 0
    for title, problems in (
        ("T-SQL parse", parse_batches()),
        ("references", check_references()),
        ("identifiers", check_identifiers()),
    ):
        if problems:
            failures += len(problems)
            print(f"{title}: {len(problems)} problem(s)")
            for problem in problems[:20]:
                print(f"    {problem}")
            if len(problems) > 20:
                print(f"    ... and {len(problems) - 20} more")
        else:
            print(f"{title}: ok")
    sys.exit(1 if failures else 0)


if __name__ == "__main__":
    main()
