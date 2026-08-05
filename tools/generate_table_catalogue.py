#!/usr/bin/env python3
"""Generate docs/s4hana/table_catalogue.csv from the markdown catalogue.

The markdown files are the source of truth. Each table entry looks like:

    ### `fin.JournalEntryHeader`
    **Universal journal document header** · reference: `BKPF`

    | Field Name | Data Type | Key | Null | Description |
    |---|---|---|---|---|
    | `Id` | `bigint` | PK | no | Surrogate key |
    | *include* | `#AUDIT` | | | Standard audit columns |

Rows marked `*include*` are expanded from the standard column groups defined in
README.md, so the CSV is a complete Schema / Table / Field / DataType list.

Usage:  python3 tools/generate_table_catalogue.py
"""

from __future__ import annotations

import csv
import pathlib
import re
import sys

DOCS = pathlib.Path(__file__).resolve().parent.parent / "docs" / "s4hana"
OUTPUT = DOCS / "table_catalogue.csv"

# Catalogue files, in catalogue order. 10_sap_reference_mapping.md is a mapping
# document, not a table definition, so it is deliberately excluded.
CATALOGUE_FILES = [
    "01_enterprise_structure.md",
    "02_configuration.md",
    "03_data_dictionary.md",
    "04_business_partner.md",
    "05_financial_accounting.md",
    "06_asset_accounting.md",
    "07_controlling.md",
    "08_workflow_security_audit.md",
    "09_reporting_integration.md",
]

TABLE_HEADING = re.compile(r"^###\s+`(?P<schema>[a-z]+)\.(?P<table>[A-Za-z0-9_]+)`\s*$")
INCLUDE_HEADING = re.compile(r"^###\s+`(?P<name>#[A-Z]+)`\s*$")
SUBTITLE = re.compile(r"^\*\*(?P<desc>.+?)\*\*(?:\s*·\s*reference:\s*(?P<ref>.+))?\s*$")
CODE = re.compile(r"^`(.*)`$")


def strip_code(value: str) -> str:
    value = value.strip()
    match = CODE.match(value)
    return match.group(1) if match else value


SEPARATOR = object()


def split_row(line: str):
    """Split a markdown table row into its cells.

    Returns None when the line is not part of a table, and the SEPARATOR
    sentinel for the `|---|---|` alignment row.
    """
    line = line.strip()
    if not line.startswith("|") or not line.endswith("|"):
        return None
    cells = [cell.strip() for cell in line[1:-1].split("|")]
    if all(cell and set(cell) <= {"-", ":"} for cell in cells):
        return SEPARATOR
    return cells


def parse_field_rows(lines: list[str], start: int) -> tuple[list[dict], int]:
    """Read the field table starting at or after `start`. Returns rows and the
    index of the first line after the table."""
    fields: list[dict] = []
    index = start
    seen_table = False
    while index < len(lines):
        line = lines[index]
        cells = split_row(line)
        if cells is None:
            if seen_table and line.strip() == "":
                index += 1
                continue
            if seen_table:
                break
            if line.startswith("###") or line.startswith("## "):
                break
            index += 1
            continue
        seen_table = True
        index += 1
        if cells is SEPARATOR:
            continue
        if cells[0].lower().startswith("field name"):
            continue  # header row
        if len(cells) < 2:
            continue
        name, data_type = cells[0], strip_code(cells[1])
        key = cells[2] if len(cells) > 2 else ""
        nullable = cells[3] if len(cells) > 3 else ""
        description = cells[4] if len(cells) > 4 else ""
        if name.lower() in ("*include*", "_include_"):
            fields.append({"include": data_type})
        else:
            fields.append(
                {
                    "field": strip_code(name),
                    "type": data_type,
                    "key": key,
                    "null": nullable,
                    "description": description,
                }
            )
    return fields, index


def parse_includes() -> dict[str, list[dict]]:
    """Read the standard column groups from README.md."""
    includes: dict[str, list[dict]] = {}
    lines = (DOCS / "README.md").read_text(encoding="utf-8").splitlines()
    index = 0
    while index < len(lines):
        match = INCLUDE_HEADING.match(lines[index])
        if not match:
            index += 1
            continue
        fields, index = parse_field_rows(lines, index + 1)
        includes[match.group("name")] = [f for f in fields if "field" in f]
    return includes


def parse_catalogue(includes: dict[str, list[dict]]) -> list[dict]:
    rows: list[dict] = []
    for file_name in CATALOGUE_FILES:
        path = DOCS / file_name
        if not path.exists():
            sys.exit(f"missing catalogue file: {path}")
        lines = path.read_text(encoding="utf-8").splitlines()
        index = 0
        while index < len(lines):
            match = TABLE_HEADING.match(lines[index])
            if not match:
                index += 1
                continue
            schema, table = match.group("schema"), match.group("table")
            index += 1
            table_desc, reference = "", ""
            while index < len(lines) and not lines[index].strip().startswith("|"):
                subtitle = SUBTITLE.match(lines[index].strip())
                if subtitle:
                    table_desc = subtitle.group("desc")
                    reference = strip_code(subtitle.group("ref") or "")
                index += 1
            fields, index = parse_field_rows(lines, index)
            position = 0
            seen: dict[str, bool] = {}
            for field in fields:
                from_include = "include" in field
                expanded = (
                    includes.get(field["include"], []) if from_include else [field]
                )
                if from_include and not expanded:
                    sys.exit(f"unknown include {field['include']} in {schema}.{table}")
                for item in expanded:
                    # A table may state a field explicitly that one of its
                    # includes also brings in (IsActive is the common case).
                    # The explicit definition wins; two explicit rows with the
                    # same name are an error.
                    if item["field"] in seen:
                        if from_include:
                            continue
                        sys.exit(
                            f"{schema}.{table}: field {item['field']} defined twice"
                        )
                    seen[item["field"]] = from_include
                    position += 1
                    rows.append(
                        {
                            "Schema": schema,
                            "Table": table,
                            "FullTableName": f"{schema}.{table}",
                            "Position": position,
                            "Field": item["field"],
                            "DataType": item["type"],
                            "Key": item["key"],
                            "Nullable": item["null"],
                            "Description": item["description"],
                            "TableDescription": table_desc,
                            "SapReference": reference,
                            "SourceFile": file_name,
                        }
                    )
    return rows


def main() -> None:
    includes = parse_includes()
    if not includes:
        sys.exit("no standard column groups found in README.md")
    rows = parse_catalogue(includes)
    if not rows:
        sys.exit("no tables parsed — check the markdown format")

    with OUTPUT.open("w", newline="", encoding="utf-8") as handle:
        writer = csv.DictWriter(handle, fieldnames=list(rows[0].keys()))
        writer.writeheader()
        writer.writerows(rows)

    tables = {row["FullTableName"] for row in rows}
    schemas: dict[str, set[str]] = {}
    for row in rows:
        schemas.setdefault(row["Schema"], set()).add(row["FullTableName"])
    print(f"{OUTPUT.relative_to(OUTPUT.parent.parent.parent)}: "
          f"{len(tables)} tables, {len(rows)} fields")
    for schema in sorted(schemas):
        print(f"  {schema:<6} {len(schemas[schema]):>3} tables")


if __name__ == "__main__":
    main()
