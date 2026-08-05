#!/usr/bin/env python3
"""Generate the SQL Server DDL for the S/4HANA-inspired ERP.

Source of truth: docs/s4hana/table_catalogue.csv.
Output: database/s4hana/*.sql

    python3 tools/generate_sql_ddl.py
"""

from __future__ import annotations

import pathlib
import sys

sys.path.insert(0, str(pathlib.Path(__file__).resolve().parent))

from catalogue_model import (  # noqa: E402
    ROOT,
    SCHEMA_ORDER,
    Table,
    load_tables,
    resolve_foreign_keys,
)

OUT = ROOT / "database" / "s4hana"
DATABASE = "ErpS4"

FILE_NUMBERS = {
    "org": "10",
    "cfg": "20",
    "mdm": "30",
    "fin": "40",
    "co": "50",
    "wf": "60",
    "sec": "70",
    "audit": "75",
    "rpt": "80",
    "intg": "85",
}

SCHEMA_TITLES = {
    "org": "Enterprise structure",
    "cfg": "Configuration, data dictionary, custom objects",
    "mdm": "Business Partner and master data",
    "fin": "Financial accounting and asset accounting",
    "co": "Controlling",
    "wf": "Workflow and approval",
    "sec": "Users and security",
    "audit": "Audit and change documents",
    "rpt": "Reporting metadata",
    "intg": "Integration and API management",
}

BANNER = """/* ============================================================================
   {title}
   {subtitle}

   GENERATED FILE - do not edit by hand.
   Source: docs/s4hana/table_catalogue.csv
   Regenerate: python3 tools/generate_sql_ddl.py
   ============================================================================ */
"""


ASCII_FOLD = {
    "—": "-",   # em dash
    "–": "-",   # en dash
    "‘": "'",
    "’": "'",
    "“": '"',
    "”": '"',
    "·": "-",
    "→": "->",
}


def write(name: str, content: str) -> None:
    """UTF-8 with BOM so SSMS and sqlcmd both read the file correctly."""
    (OUT / name).write_text(content, encoding="utf-8-sig")


def clean(text: str) -> str:
    """Plain ASCII: sqlcmd is routinely run with a code page that mangles
    anything else, and these strings end up inside N'...' literals."""
    text = text.replace("`", "").replace("\r", " ").replace("\n", " ").strip()
    for source, target in ASCII_FOLD.items():
        text = text.replace(source, target)
    return text.encode("ascii", "replace").decode("ascii")


def quote(text: str) -> str:
    return text.replace("'", "''")


def column_ddl(table: Table) -> list[str]:
    lines: list[str] = []
    width = max(len(f.name) for f in table.fields) + 2
    single_pk = len(table.primary_key) == 1
    for f in table.fields:
        identity = (
            single_pk
            and f.is_pk
            and f.name == "Id"
            and f.sql_type in ("bigint", "int")
        )
        parts = [f"        [{f.name}]".ljust(8 + width), f.sql_type]
        if identity:
            parts.append("IDENTITY(1,1)")
        parts.append("NULL" if f.nullable else "NOT NULL")
        if f.name == "CreatedAt" and not f.nullable:
            parts.append(
                f"CONSTRAINT [DF_{table.schema}_{table.name}_CreatedAt] "
                "DEFAULT (SYSUTCDATETIME())"
            )
        elif f.sql_type == "bit" and not f.nullable:
            default = "1" if f.name == "IsActive" else "0"
            parts.append(
                f"CONSTRAINT [DF_{table.schema}_{table.name}_{f.name}] "
                f"DEFAULT ({default})"
            )
        line = " ".join(parts) + ","
        description = clean(f.description)
        if description:
            line = f"{line:<118}-- {description}"
        lines.append(line)
    return lines


def table_ddl(table: Table) -> str:
    pk = ", ".join(f"[{f.name}]" for f in table.primary_key)
    body = column_ddl(table)
    body.append(
        f"        CONSTRAINT [PK_{table.schema}_{table.name}] PRIMARY KEY CLUSTERED ({pk})"
    )
    ak = table.alternate_key
    if ak:
        columns = ", ".join(f"[{f.name}]" for f in ak)
        body[-1] += ","
        body.append(
            f"        CONSTRAINT [UQ_{table.schema}_{table.name}] UNIQUE ({columns})"
        )
    header = f"/* {table.full_name} - {clean(table.description)}"
    if table.sap_reference:
        header += f" (reference: {clean(table.sap_reference)})"
    header += " */"
    return (
        f"{header}\n"
        f"IF OBJECT_ID(N'{table.full_name}', N'U') IS NULL\n"
        f"BEGIN\n"
        f"    CREATE TABLE [{table.schema}].[{table.name}]\n"
        f"    (\n" + "\n".join(body) + "\n"
        f"    );\n"
        f"END\n"
        f"GO\n"
    )


def write_schema_files(tables: list[Table]) -> list[str]:
    written: list[str] = []
    for schema in SCHEMA_ORDER:
        schema_tables = [t for t in tables if t.schema == schema]
        name = f"{FILE_NUMBERS[schema]}_schema_{schema}.sql"
        parts = [
            BANNER.format(
                title=f"S/4HANA-inspired ERP - schema [{schema}]",
                subtitle=f"{SCHEMA_TITLES[schema]} ({len(schema_tables)} tables)",
            ),
            f"USE [{DATABASE}];\nGO\n",
        ]
        parts.extend(table_ddl(t) for t in schema_tables)
        write(name, "\n".join(parts))
        written.append(name)
    return written


def write_database_file(tables: list[Table]) -> str:
    name = "00_create_database.sql"
    schemas = "\n".join(
        f"IF SCHEMA_ID(N'{s}') IS NULL EXEC(N'CREATE SCHEMA [{s}];');\nGO"
        for s in SCHEMA_ORDER
    )
    content = (
        BANNER.format(
            title="S/4HANA-inspired ERP - database and schemas",
            subtitle=f"{len(tables)} tables across {len(SCHEMA_ORDER)} schemas",
        )
        + f"""
IF DB_ID(N'{DATABASE}') IS NULL
BEGIN
    CREATE DATABASE [{DATABASE}];
END
GO

ALTER DATABASE [{DATABASE}] SET READ_COMMITTED_SNAPSHOT ON WITH ROLLBACK IMMEDIATE;
GO

USE [{DATABASE}];
GO

{schemas}
"""
    )
    write(name, content)
    return name


def write_foreign_keys(tables: list[Table]) -> str:
    foreign_keys, logical = resolve_foreign_keys(tables)
    name = "90_foreign_keys.sql"
    parts = [
        BANNER.format(
            title="S/4HANA-inspired ERP - foreign keys",
            subtitle=f"{len(foreign_keys)} constraints, added after all tables exist",
        ),
        f"USE [{DATABASE}];\nGO\n",
    ]
    for fk in foreign_keys:
        columns = ", ".join(f"[{c}]" for c in fk.columns)
        target_columns = ", ".join(f"[{c}]" for c in fk.target_columns)
        kind = "business key" if fk.is_code_key else "surrogate key"
        parts.append(
            f"-- {fk.table.full_name}.{fk.columns[-1]} -> "
            f"{fk.target_schema}.{fk.target_table} ({kind})\n"
            f"IF NOT EXISTS (SELECT 1 FROM sys.foreign_keys WHERE name = N'{fk.name}')\n"
            f"    ALTER TABLE [{fk.table.schema}].[{fk.table.name}] WITH CHECK\n"
            f"        ADD CONSTRAINT [{fk.name}] FOREIGN KEY ({columns})\n"
            f"        REFERENCES [{fk.target_schema}].[{fk.target_table}] ({target_columns});\n"
            f"GO\n"
        )
    parts.append(
        "/* References the application enforces instead of the database:\n"
        + "\n".join(f"     {t}.{f} - {reason}" for t, f, reason in logical)
        + "\n */\n"
    )
    write(name, "\n".join(parts))
    return name


def write_indexes(tables: list[Table]) -> str:
    name = "91_indexes.sql"
    parts = [
        BANNER.format(
            title="S/4HANA-inspired ERP - secondary indexes",
            subtitle="Indexes for the access paths marked IX in the catalogue",
        ),
        f"USE [{DATABASE}];\nGO\n",
        "/* Primary and business keys are created with the tables. The indexes\n"
        "   below cover the documented read paths (posting date, document number,\n"
        "   partner, status, correlation id). Add further indexes for foreign key\n"
        "   columns once a real workload has been measured - indexing all 838 of\n"
        "   them up front costs more on write than it returns on read. */\n",
    ]
    count = 0
    for table in tables:
        tenant = "[TenantId], " if table.has_tenant else ""
        for f in table.fields:
            if not f.is_indexed or f.name == "TenantId":
                continue
            index_name = f"IX_{table.schema}_{table.name}_{f.name}"
            parts.append(
                f"IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'{index_name}'\n"
                f"               AND object_id = OBJECT_ID(N'{table.full_name}'))\n"
                f"    CREATE NONCLUSTERED INDEX [{index_name}]\n"
                f"        ON [{table.schema}].[{table.name}] ({tenant}[{f.name}]);\n"
                f"GO\n"
            )
            count += 1
    write(name, "\n".join(parts))
    print(f"  {name}: {count} indexes")
    return name


def write_dictionary_seed(tables: list[Table]) -> str:
    """Load the catalogue into cfg.DictionaryTable / cfg.DictionaryTableField so
    SE11 and SE16N describe the real database from day one."""
    name = "92_seed_dictionary.sql"
    parts = [
        BANNER.format(
            title="S/4HANA-inspired ERP - dictionary seed",
            subtitle="Populates cfg.DictionaryTable and cfg.DictionaryTableField",
        ),
        f"USE [{DATABASE}];\nGO\n",
        "/* Seed tenant 1. The dictionary rows below are tenant dependent and\n"
        "   carry a foreign key to org.Tenant, so it has to exist first. */\n"
        "IF NOT EXISTS (SELECT 1 FROM org.Tenant WHERE Id = 1)\n"
        "BEGIN\n"
        "    SET IDENTITY_INSERT org.Tenant ON;\n"
        "    INSERT INTO org.Tenant\n"
        "        (Id, TenantCode, Name, DefaultLanguage, TimeZoneId, IsProduction,\n"
        "         AllowCustomizingChanges, ValidFrom, ValidTo, CreatedBy)\n"
        "    VALUES\n"
        "        (1, N'100', N'Default tenant', N'EN', N'Asia/Phnom_Penh', 0,\n"
        "         1, '2000-01-01', '9999-12-31', N'SYSTEM');\n"
        "    SET IDENTITY_INSERT org.Tenant OFF;\n"
        "END\n"
        "GO\n"
        "\n"
        "/* Re-runnable: clear the generated dictionary rows for tenant 1 first. */\n"
        "DELETE f FROM cfg.DictionaryTableField AS f\n"
        "JOIN   cfg.DictionaryTable AS t ON t.Id = f.DictionaryTableId\n"
        "WHERE  t.TenantId = 1;\n"
        "DELETE FROM cfg.DictionaryTable WHERE TenantId = 1;\n"
        "GO\n",
    ]

    table_rows = []
    for t in tables:
        category = (
            "Configuration"
            if t.schema == "cfg"
            else "Master"
            if t.schema in ("mdm", "org")
            else "Transaction"
        )
        immutable = 1 if t.is_append_only else 0
        pk = ",".join(f.name for f in t.primary_key)
        table_rows.append(
            f"(1, N'{t.schema}', N'{t.name}', N'{quote(clean(t.description))}', "
            f"N'{category}', N'A', N'Maintain', {1 if t.has_tenant else 0}, "
            f"{1 if t.field('CompanyCodeId') else 0}, 3, N'None', 1, {immutable}, "
            f"N'{pk}', N'Active', N'SYSTEM')"
        )
    parts.append(
        "INSERT INTO cfg.DictionaryTable\n"
        "    (TenantId, SchemaName, TableName, ShortDescription, TableCategory,\n"
        "     DeliveryClass, MaintenanceType, IsTenantDependent, IsCompanyCodeDependent,\n"
        "     SizeCategory, BufferingType, IsLogged, IsImmutable, PrimaryKeyFields,\n"
        "     Status, CreatedBy)\n"
        "VALUES\n" + ",\n".join(table_rows) + ";\nGO\n"
    )

    field_rows: list[str] = []
    for t in tables:
        for f in t.fields:
            field_rows.append(
                f"(1, N'{t.schema}', N'{t.name}', N'{f.name}', {f.position}, "
                f"N'{f.sql_type}', {1 if f.is_pk else 0}, {0 if f.nullable else 1}, "
                f"{1 if f.is_pk and f.name == 'Id' else 0}, "
                f"N'{quote(clean(f.description))}')"
            )
    parts.append(
        "/* Fields are staged, then matched to their table by schema and name. */\n"
        "CREATE TABLE #DictionaryField\n"
        "(\n"
        "    TenantId int, SchemaName nvarchar(20), TableName nvarchar(64),\n"
        "    FieldName nvarchar(64), FieldPosition int, SqlType nvarchar(40),\n"
        "    IsKey bit, IsRequired bit, IsIdentity bit, ShortDescription nvarchar(255)\n"
        ");\nGO\n"
    )
    for start in range(0, len(field_rows), 900):
        chunk = field_rows[start : start + 900]
        parts.append(
            "INSERT INTO #DictionaryField\n    (TenantId, SchemaName, TableName, FieldName,\n"
            "     FieldPosition, SqlType, IsKey, IsRequired, IsIdentity, ShortDescription)\n"
            "VALUES\n" + ",\n".join(chunk) + ";\nGO\n"
        )
    parts.append(
        "INSERT INTO cfg.DictionaryTableField\n"
        "    (TenantId, DictionaryTableId, FieldName, FieldPosition, SqlType,\n"
        "     IsKey, IsRequired, IsIdentity, IsCustomField, IsMasked, ShortDescription,\n"
        "     CreatedBy)\n"
        "SELECT s.TenantId, d.Id, s.FieldName, s.FieldPosition, s.SqlType,\n"
        "       s.IsKey, s.IsRequired, s.IsIdentity, 0, 0, s.ShortDescription, N'SYSTEM'\n"
        "FROM   #DictionaryField AS s\n"
        "JOIN   cfg.DictionaryTable AS d\n"
        "       ON  d.TenantId   = s.TenantId\n"
        "       AND d.SchemaName = s.SchemaName\n"
        "       AND d.TableName  = s.TableName;\n"
        "GO\n"
        "DROP TABLE #DictionaryField;\nGO\n"
    )
    write(name, "\n".join(parts))
    print(f"  {name}: {len(tables)} tables, {len(field_rows)} fields seeded")
    return name


def write_run_all(files: list[str]) -> None:
    listing = "\n".join(f":r {f}" for f in files)
    content = f"""/* ============================================================================
   S/4HANA-inspired ERP - master install script

   Run with SQLCMD from this directory:
       sqlcmd -S localhost -i run_all.sql
   or open each file in order in SSMS. Every script is idempotent.
   ============================================================================ */
{listing}
GO
PRINT 'ErpS4 database installation complete.';
GO
"""
    write("run_all.sql", content)


def main() -> None:
    OUT.mkdir(parents=True, exist_ok=True)
    tables = load_tables()
    files = [write_database_file(tables)]
    files.extend(write_schema_files(tables))
    files.append(write_foreign_keys(tables))
    files.append(write_indexes(tables))
    files.append(write_dictionary_seed(tables))
    write_run_all(files)
    columns = sum(len(t.fields) for t in tables)
    print(f"database/s4hana: {len(tables)} tables, {columns} columns, {len(files)} scripts")


if __name__ == "__main__":
    main()
