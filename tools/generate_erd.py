#!/usr/bin/env python3
"""Generate the entity relationship diagrams for the catalogue.

One Mermaid erDiagram per catalogue file, drawn from the resolved foreign keys
so the diagrams cannot disagree with the DDL. Entities show their primary and
business keys; relationships are labelled with the foreign key column.

Two kinds of edge are left out of the diagrams on purpose:
  * TenantId -> org.Tenant, which every tenant-dependent table has and which
    would connect all 226 of them to one box;
  * business-key edges to the code tables (currency, country, language, unit),
    which would do the same for cfg.Currency.
Both are listed under each diagram instead.

    python3 tools/generate_erd.py
"""

from __future__ import annotations

import pathlib
import sys

sys.path.insert(0, str(pathlib.Path(__file__).resolve().parent))

from catalogue_model import ROOT, Table, load_tables, resolve_foreign_keys  # noqa: E402

OUTPUT = ROOT / "docs" / "s4hana" / "11_entity_relationships.md"

GROUP_TITLES = {
    "01_enterprise_structure.md": ("Enterprise structure", "org"),
    "02_configuration.md": ("Configuration", "cfg"),
    "03_data_dictionary.md": ("Data dictionary, table browser, custom objects", "cfg"),
    "04_business_partner.md": ("Business Partner and master data", "mdm"),
    "05_financial_accounting.md": ("Financial accounting", "fin"),
    "06_asset_accounting.md": ("Asset accounting", "fin"),
    "07_controlling.md": ("Controlling", "co"),
    "08_workflow_security_audit.md": ("Workflow, security, audit", "wf / sec / audit"),
    "09_reporting_integration.md": ("Reporting and integration", "rpt / intg"),
}

GROUP_ORDER = list(GROUP_TITLES)

def mermaid_type(sql_type: str) -> str:
    """Mermaid attribute types are bare identifiers - parentheses break the ER
    parser, so lengths and precisions are dropped here. They are in the
    catalogue, which is where anyone needing them will look."""
    return sql_type.split("(")[0]


def entity_name(table: Table) -> str:
    return f"{table.schema}_{table.name}"


def entity_block(table: Table, detailed: bool) -> list[str]:
    lines = [f"    {entity_name(table)} {{"]
    if detailed:
        shown = table.primary_key + [
            f for f in table.alternate_key if f.name != "TenantId"
        ]
        for f in shown:
            marker = "PK" if f.is_pk else "UK"
            lines.append(f"        {mermaid_type(f.sql_type)} {f.name} {marker}")
    lines.append("    }")
    return lines


def diagram(group: str, tables: list[Table], foreign_keys: list) -> str:
    members = {t.full_name: t for t in tables}
    inside: list = []
    outside: dict[str, list[tuple[str, str]]] = {}

    for fk in foreign_keys:
        if fk.table.full_name not in members:
            continue
        if fk.columns[-1] == "TenantId" or fk.is_code_key:
            continue
        target = f"{fk.target_schema}.{fk.target_table}"
        if target in members:
            inside.append(fk)
        else:
            outside.setdefault(target, []).append(
                (fk.table.full_name, fk.columns[-1])
            )

    lines = ["```mermaid", "erDiagram"]
    for table in tables:
        lines.extend(entity_block(table, detailed=True))
    for fk in inside:
        source = entity_name(fk.table)
        target = f"{fk.target_schema}_{fk.target_table}"
        optional = fk.table.field(fk.columns[-1]).nullable  # type: ignore[union-attr]
        connector = "||--o{" if optional else "||--|{"
        lines.append(f"    {target} {connector} {source} : \"{fk.columns[-1]}\"")
    lines.append("```")

    body = ["\n".join(lines)]
    if outside:
        body.append("\n**References into other modules**\n")
        body.append("| From | Column | To |")
        body.append("|------|--------|----|")
        for target in sorted(outside):
            for source, column in sorted(outside[target]):
                body.append(f"| `{source}` | `{column}` | `{target}` |")
    return "\n".join(body)


def main() -> None:
    tables = load_tables()
    foreign_keys, _ = resolve_foreign_keys(tables)

    tenant_dependent = sum(1 for t in tables if t.has_tenant)
    code_edges = sum(1 for fk in foreign_keys if fk.is_code_key)

    parts = [
        "# 11 — Entity Relationships",
        "",
        "Generated from the resolved foreign keys, so these diagrams and",
        "`database/s4hana/90_foreign_keys.sql` always describe the same model.",
        "Regenerate with `python3 tools/generate_erd.py`.",
        "",
        "One diagram per catalogue file. Entities list their primary key (`PK`)",
        "and business key (`UK`); every other column is in the catalogue itself.",
        "A solid line is a required reference, a circle-ended line an optional one.",
        "",
        "Two edge sets are omitted from every diagram to keep them readable:",
        "",
        f"* **Tenant** — {tenant_dependent} of {len(tables)} tables carry",
        "  `TenantId` with a foreign key to `org.Tenant`.",
        f"* **Code tables** — {code_edges} business-key references to",
        "  `cfg.Currency`, `cfg.Country`, `cfg.Language`, `cfg.UnitOfMeasure`,",
        "  `cfg.PostingKey`, `cfg.Region`, `cfg.TableAuthorizationGroup`,",
        "  `mdm.Bank` and `org.Company` (trading partner).",
        "",
        "## Contents",
        "",
    ]
    for index, group in enumerate(GROUP_ORDER, start=1):
        title = GROUP_TITLES[group][0]
        anchor = title.lower().replace(" ", "-").replace(",", "").replace("/", "")
        parts.append(f"{index}. [{title}](#{index}-{anchor})")
    parts.append("")

    for index, group in enumerate(GROUP_ORDER, start=1):
        title, schemas = GROUP_TITLES[group]
        group_tables = [t for t in tables if t.source_file == group]
        parts.append(f"## {index}. {title}")
        parts.append("")
        parts.append(
            f"Schema `{schemas}` - {len(group_tables)} tables, "
            f"catalogue file [`{group}`]({group})."
        )
        parts.append("")
        parts.append(diagram(group, group_tables, foreign_keys))
        parts.append("")

    OUTPUT.write_text("\n".join(parts), encoding="utf-8")
    drawn = sum(
        1
        for fk in foreign_keys
        if fk.columns[-1] != "TenantId" and not fk.is_code_key
    )
    print(f"{OUTPUT.relative_to(ROOT)}: {len(GROUP_ORDER)} diagrams, {drawn} edges drawn")


if __name__ == "__main__":
    main()
