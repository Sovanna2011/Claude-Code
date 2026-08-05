"""Shared model for the S/4HANA-inspired ERP catalogue.

Reads docs/s4hana/table_catalogue.csv (produced by generate_table_catalogue.py)
and resolves foreign keys, so the DDL and the EF Core generators always emit the
same physical model.
"""

from __future__ import annotations

import csv
import pathlib
import re
import sys
from dataclasses import dataclass, field as dc_field

ROOT = pathlib.Path(__file__).resolve().parent.parent
CATALOGUE_CSV = ROOT / "docs" / "s4hana" / "table_catalogue.csv"

# Schema order used for file numbering and for the run_all script.
SCHEMA_ORDER = ["org", "cfg", "mdm", "fin", "co", "wf", "sec", "audit", "rpt", "intg"]

# Foreign keys whose target cannot be derived from the column name.
# Keyed by "schema.Table.Field" first, then by bare field name.
FK_OVERRIDES: dict[str, str] = {
    # self references
    "cfg.FinancialStatementNode.ParentNodeId": "cfg.FinancialStatementNode",
    "co.HierarchyNode.ParentNodeId": "co.HierarchyNode",
    "cfg.TaxJurisdiction.ParentJurisdictionId": "cfg.TaxJurisdiction",
    # business partner used under a different column name
    "AlternativePayerPayeeId": "mdm.BusinessPartner",
    "HeadOfficePartnerId": "mdm.BusinessPartner",
    "ClearingPartnerId": "mdm.BusinessPartner",
    "DunningRecipientPartnerId": "mdm.BusinessPartner",
    "ResponsiblePersonPartnerId": "mdm.BusinessPartner",
    "ApplicantPartnerId": "mdm.BusinessPartner",
    # dictionary objects
    "SearchHelpId": "cfg.DictionarySearchHelp",
    "DataElementId": "cfg.DictionaryDataElement",
    "ForeignKeyId": "cfg.DictionaryForeignKey",
    # type tables
    "OrderTypeId": "co.InternalOrderType",
    "TransactionTypeId": "fin.AssetTransactionType",
    "FormId": "cfg.CorrespondenceForm",
    "TransportRequestId": "cfg.CustomObjectRequest",
}

# Columns marked FK that intentionally get no database constraint. The value is
# the reason, emitted as a comment in the foreign-key script.
FK_LOGICAL_ONLY: dict[str, str] = {
    "ProjectId": "project / WBS element belongs to a future module",
    "TaxJurisdictionSchemaId": "jurisdiction schema key, not a jurisdiction row",
    "SenderObjectId": "polymorphic: qualified by SenderObjectType",
}

# Natural-key (code) foreign keys, pointing at the business key of the target.
#   match  - "suffix" or "exact" on the referencing column name
#   value  - the suffix or exact column name
#   target - target table
#   map    - referencing columns paired with target columns; "$" is the matched
#            column itself. TenantId is prepended when the target is tenant
#            dependent.
CODE_FK_RULES: list[dict] = [
    {"match": "suffix", "value": "CurrencyCode", "target": "cfg.Currency",
     "map": [("$", "CurrencyCode")]},
    {"match": "suffix", "value": "CountryCode", "target": "cfg.Country",
     "map": [("$", "CountryCode")]},
    {"match": "suffix", "value": "LanguageCode", "target": "cfg.Language",
     "map": [("$", "LanguageCode")]},
    {"match": "exact", "value": "MaintenanceLanguage", "target": "cfg.Language",
     "map": [("$", "LanguageCode")]},
    {"match": "suffix", "value": "UnitOfMeasure", "target": "cfg.UnitOfMeasure",
     "map": [("$", "UnitOfMeasure")]},
    {"match": "exact", "value": "PostingKey", "target": "cfg.PostingKey",
     "map": [("$", "PostingKey")]},
    {"match": "exact", "value": "AuthorizationGroup",
     "target": "cfg.TableAuthorizationGroup", "map": [("$", "AuthorizationGroup")]},
    {"match": "exact", "value": "TradingPartnerCompany", "target": "org.Company",
     "map": [("$", "CompanyCodeGroup")]},
    {"match": "exact", "value": "RegionCode", "target": "cfg.Region",
     "map": [("CountryCode", "CountryCode"), ("$", "RegionCode")]},
    {"match": "exact", "value": "BankKey", "target": "mdm.Bank",
     "map": [("BankCountryCode", "BankCountryCode"), ("$", "BankKey")]},
]


def match_code_rule(field_name: str) -> dict | None:
    for rule in CODE_FK_RULES:
        if rule["match"] == "exact" and field_name == rule["value"]:
            return rule
        if rule["match"] == "suffix" and field_name.endswith(rule["value"]):
            return rule
    return None

# Insert-only tables: the application refuses updates and deletes on them.
# Posted-document immutability is enforced by the posting engine, not here.
APPEND_ONLY_TABLES = {
    "audit.AuditLog",
    "audit.ChangeDocumentHeader",
    "audit.ChangeDocumentItem",
    "audit.DataAccessLog",
    "cfg.BrowserQueryLog",
    "cfg.DictionaryChangeLog",
    "cfg.NumberRangeGap",
    "fin.RecurringEntryExecution",
    "intg.WebhookDelivery",
    "rpt.ReportExecutionLog",
    "sec.LoginHistory",
    "sec.PasswordHistory",
    "wf.WorkflowHistory",
}

DECIMAL = re.compile(r"^decimal\((?P<p>\d+),(?P<s>\d+)\)$")
NVARCHAR = re.compile(r"^nvarchar\((?P<len>\d+|max)\)$")


@dataclass
class Field:
    name: str
    sql_type: str
    key: str
    nullable: bool
    description: str
    position: int

    @property
    def is_pk(self) -> bool:
        return "PK" in self.key.split(",")

    @property
    def is_ak(self) -> bool:
        return "AK" in self.key.split(",")

    @property
    def is_fk(self) -> bool:
        return "FK" in self.key.split(",")

    @property
    def is_indexed(self) -> bool:
        return "IX" in self.key.split(",")

    @property
    def clr_type(self) -> str:
        base = sql_to_clr(self.sql_type)
        if base == "byte[]":
            return base
        return f"{base}?" if self.nullable else base

    @property
    def max_length(self) -> int | None:
        match = NVARCHAR.match(self.sql_type)
        if match and match.group("len") != "max":
            return int(match.group("len"))
        return None


@dataclass
class Table:
    schema: str
    name: str
    description: str
    sap_reference: str
    fields: list[Field] = dc_field(default_factory=list)

    @property
    def full_name(self) -> str:
        return f"{self.schema}.{self.name}"

    @property
    def clr_name(self) -> str:
        return self.name

    @property
    def has_tenant(self) -> bool:
        return any(f.name == "TenantId" for f in self.fields)

    @property
    def is_append_only(self) -> bool:
        return self.full_name in APPEND_ONLY_TABLES

    def field(self, name: str) -> Field | None:
        return next((f for f in self.fields if f.name == name), None)

    @property
    def primary_key(self) -> list[Field]:
        return [f for f in self.fields if f.is_pk]

    @property
    def alternate_key(self) -> list[Field]:
        """Business key. Tenant-dependent tables lead with TenantId, which is
        the isolation rule stated in the catalogue README."""
        ak = [f for f in self.fields if f.is_ak]
        if not ak:
            return []
        if self.has_tenant:
            return [self.field("TenantId")] + ak  # type: ignore[list-item]
        return ak


def sql_to_clr(sql_type: str) -> str:
    simple = {
        "bigint": "long",
        "int": "int",
        "smallint": "short",
        "tinyint": "byte",
        "bit": "bool",
        "date": "DateOnly",
        "datetime2(3)": "DateTime",
        "time(0)": "TimeOnly",
        "uniqueidentifier": "Guid",
        "rowversion": "byte[]",
    }
    if sql_type in simple:
        return simple[sql_type]
    if NVARCHAR.match(sql_type):
        return "string"
    if DECIMAL.match(sql_type):
        return "decimal"
    sys.exit(f"unmapped SQL type: {sql_type}")


def load_tables() -> list[Table]:
    if not CATALOGUE_CSV.exists():
        sys.exit(
            f"{CATALOGUE_CSV} not found — run tools/generate_table_catalogue.py first"
        )
    tables: dict[str, Table] = {}
    with CATALOGUE_CSV.open(encoding="utf-8") as handle:
        for row in csv.DictReader(handle):
            key = row["FullTableName"]
            table = tables.get(key)
            if table is None:
                table = Table(
                    schema=row["Schema"],
                    name=row["Table"],
                    description=row["TableDescription"],
                    sap_reference=row["SapReference"],
                )
                tables[key] = table
            table.fields.append(
                Field(
                    name=row["Field"],
                    sql_type=row["DataType"],
                    key=row["Key"],
                    nullable=row["Nullable"] == "yes",
                    description=row["Description"],
                    position=int(row["Position"]),
                )
            )
    ordered = sorted(
        tables.values(), key=lambda t: (SCHEMA_ORDER.index(t.schema), t.name)
    )
    for table in ordered:
        if not table.primary_key:
            sys.exit(f"{table.full_name} has no primary key")
    return ordered


@dataclass
class ForeignKey:
    table: Table
    columns: list[str]
    target_schema: str
    target_table: str
    target_columns: list[str]
    is_code_key: bool = False

    @property
    def name(self) -> str:
        return f"FK_{self.table.schema}_{self.table.name}_{'_'.join(self.columns[-1:])}"


def resolve_foreign_keys(
    tables: list[Table],
) -> tuple[list[ForeignKey], list[tuple[str, str, str]]]:
    """Returns the resolvable foreign keys plus the ones left to the
    application, as (table, field, reason)."""
    by_name: dict[str, list[Table]] = {}
    for table in tables:
        by_name.setdefault(table.name, []).append(table)
    lookup = {t.full_name: t for t in tables}

    foreign_keys: list[ForeignKey] = []
    logical: list[tuple[str, str, str]] = []

    for table in tables:
        for f in table.fields:
            if not f.is_fk:
                continue
            code_rule = match_code_rule(f.name) if not f.name.endswith("Id") else None
            if code_rule:
                target = lookup[code_rule["target"]]
                if target.full_name == table.full_name:
                    continue  # a code table does not reference itself
                columns = [f.name if s == "$" else s for s, _ in code_rule["map"]]
                target_columns = [t for _, t in code_rule["map"]]
                missing = [c for c in columns if table.field(c) is None]
                if missing:
                    logical.append(
                        (table.full_name, f.name, f"missing companion column {missing}")
                    )
                    continue
                if target.has_tenant:
                    if not table.has_tenant:
                        # e.g. org.Tenant referencing a tenant-dependent code
                        # table: there is no TenantId to join on.
                        logical.append(
                            (
                                table.full_name,
                                f.name,
                                f"{target.full_name} is tenant dependent, "
                                f"{table.full_name} is not",
                            )
                        )
                        continue
                    columns = ["TenantId"] + columns
                    target_columns = ["TenantId"] + target_columns
                expected = [x.name for x in target.alternate_key] or [
                    x.name for x in target.primary_key
                ]
                if target_columns != expected:
                    sys.exit(
                        f"{table.full_name}.{f.name}: target columns {target_columns} "
                        f"do not match the business key of {target.full_name} {expected}"
                    )
                foreign_keys.append(
                    ForeignKey(
                        table=table,
                        columns=columns,
                        target_schema=target.schema,
                        target_table=target.name,
                        target_columns=target_columns,
                        is_code_key=True,
                    )
                )
                continue

            if not f.name.endswith("Id"):
                logical.append(
                    (table.full_name, f.name, "natural-key reference, checked in code")
                )
                continue

            if f.name in FK_LOGICAL_ONLY:
                logical.append((table.full_name, f.name, FK_LOGICAL_ONLY[f.name]))
                continue

            target_full = FK_OVERRIDES.get(
                f"{table.full_name}.{f.name}"
            ) or FK_OVERRIDES.get(f.name)
            if target_full is None:
                stem = f.name[:-2]
                candidates = [
                    name for name in by_name if stem == name or stem.endswith(name)
                ]
                if not candidates:
                    logical.append((table.full_name, f.name, "UNRESOLVED"))
                    continue
                best = max(candidates, key=len)
                matches = by_name[best]
                if len(matches) > 1:
                    logical.append(
                        (table.full_name, f.name, f"AMBIGUOUS: {best} in several schemas")
                    )
                    continue
                target = matches[0]
            else:
                target = lookup[target_full]

            target_pk = target.primary_key
            if len(target_pk) != 1:
                logical.append(
                    (table.full_name, f.name, f"composite primary key on {target.full_name}")
                )
                continue
            foreign_keys.append(
                ForeignKey(
                    table=table,
                    columns=[f.name],
                    target_schema=target.schema,
                    target_table=target.name,
                    target_columns=[target_pk[0].name],
                )
            )
    return foreign_keys, logical
