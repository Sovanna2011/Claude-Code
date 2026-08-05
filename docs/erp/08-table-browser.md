# 8. General Table Browser (SE16N-like) — Query & Authorization Architecture

A secure, **read-only**, metadata-driven data browser. It serves the purpose of
SAP SE16N — look at any authorized table with filters, layouts, and export — with
an original implementation and, deliberately, **no editing capability of any
kind**.

> **§11 hard rule:** *"Never provide a hidden editing function for posted
> accounting records."* This design has no write path at all. The service
> contract exposes only query operations; the connection used is opened with a
> **read-only application user** (`erp_reader`) that has `SELECT` and nothing else
> — so even a code defect cannot mutate data through this feature.

## 8.1 Request → result pipeline

```
┌──────────────────────────────────────────────────────────────────────────────┐
│ 1. Resolve table   name → cfg.DictionaryTable   (must exist, Active,          │
│                    BrowsableInTableBrowser = true)                            │
│ 2. Table authz     user has TABLE_BROWSE for the table's AuthorizationGroup   │
│ 3. Field authz     project only fields with IsBrowsable AND field-level grant; │
│                    fields marked IsMasked render masked unless UNMASK granted  │
│ 4. Filter parse    each filter → (field, operator, values); field must be in   │
│                    the authorized projection; operator allowed for its type    │
│ 5. Predicate build parameterized expression tree — NO string concatenation     │
│ 6. Mandatory scope inject TenantId = @tenant  (always, non-removable)         │
│                    inject CompanyCodeId IN @authorizedCompanyCodes            │
│                    inject CostCenterId / ProfitCenterId scope where applicable │
│                    inject row-level policy predicates from sec.RowLevelPolicy  │
│ 7. Sort/paginate   whitelisted sort fields; keyset or OFFSET/FETCH; row cap    │
│ 8. Execute         Dapper, read-only connection, CommandTimeout = configured   │
│ 9. Mask            apply masking rules to the materialized rows                │
│10. Audit           write audit.TableBrowserQueryLog (always, even on 0 rows)   │
│11. Return          rows + column metadata + total count (capped) + truncation  │
└──────────────────────────────────────────────────────────────────────────────┘
```

Steps 6 and 3 are the security core: **authorization is added to the query, not
checked after it**. A user cannot phrase a filter that removes the tenant or
company-code predicate, because those are appended by the builder after all
user input has been consumed.

## 8.2 No arbitrary SQL — the safe query builder

User input never reaches SQL as text. The request is a typed object:

```csharp
public sealed record TableBrowserQuery(
    string Schema, string Table,
    IReadOnlyList<string> OutputFields,
    IReadOnlyList<FilterCriterion> Filters,   // include + exclude
    IReadOnlyList<SortCriterion> Sort,
    int Page, int PageSize,
    Guid? VariantId, Guid? LayoutId);

public sealed record FilterCriterion(
    string Field, FilterOperator Operator,
    IReadOnlyList<string?> Values,            // 1 value, 2 for Between, N for In
    bool IsExclusion);                        // "exclude" filters (§11)
```

The builder:

- resolves `Field` against dictionary metadata → real column name (so a user
  cannot inject a column expression);
- converts `Values` to the field's CLR type via the domain definition, rejecting
  anything that does not parse (no implicit conversions in SQL);
- emits `@p0, @p1, …` parameters only;
- refuses any operator not permitted for the field's data type (e.g. `Contains`
  on a decimal);
- caps the number of filters, `IN` list length, and `LIKE` leading-wildcard usage
  (leading `%` requires an additional permission because of the scan cost).

### Operators (§11)

| Operator | SQL | Applies to |
|---|---|---|
| Equal / Not equal | `= / <>` | all |
| Greater / Less / ≥ / ≤ | `> < >= <=` | numeric, date, string |
| Between | `BETWEEN @a AND @b` | numeric, date, string |
| Contains / Starts with / Ends with | `LIKE` with escaped wildcards | string only |
| Is empty / Is not empty | `IS NULL OR = ''` / negation | nullable |
| In list / Not in list | `IN (…) / NOT IN (…)` | all, length-capped |

Exclusion filters are combined as `AND NOT (…)`, include filters as `AND (…)`,
with multiple values on one field OR-ed inside their group — the standard
selection-criteria semantics users expect.

## 8.3 Authorization layers

| Layer | Mechanism | Failure mode |
|---|---|---|
| **Feature** | T-code `SE16N` assigned to the role | 403 |
| **Table** | `sec.AuthorizationValue` for object `S_TABLE` field `AUTH_GROUP` matching `cfg.DictionaryObject.AuthorizationGroup`, activity `03` | 403, audited |
| **Field** | `cfg.DictionaryTableField.IsBrowsable` + role field-exclusion list | Field silently absent from the projection and from the field picker |
| **Sensitive field** | `IsMasked` + `MaskingRule` (e.g. show last 4 of a bank account) | Masked value returned; unmasking requires `DATA_UNMASK` and is separately audited |
| **Row — tenant** | Always injected | Impossible to bypass |
| **Row — company code** | `IN` the user's authorized company codes | Rows outside scope never returned |
| **Row — CO objects** | Cost center / profit center scope when the table carries those columns | |
| **Row — custom policy** | `sec.RowLevelPolicy` (table, predicate template, role) | e.g. HR-sensitive tables restricted to owning department |
| **Export** | Separate permission `TABLE_EXPORT`; row cap for export is separately configured | Download button disabled |
| **System tables** | `sec.*`, `audit.*`, dictionary internals: **not browsable at all** (`BrowsableInTableBrowser = false`) except for dedicated auditor roles with a purpose-built read view | Not listed, 403 if requested |

Two rules that the §23 test suite proves:

1. **`SE16N cannot bypass row or field authorization`** — a test issues the same
   query as two users with different company-code scopes and asserts disjoint
   results and identical, non-leaking metadata.
2. A user without `DATA_UNMASK` receives masked values *from the database
   projection level*, not merely hidden in the UI.

## 8.4 Limits and protection

| Control | Default | Configurable per |
|---|---|---|
| Max rows returned | 10,000 | table (`cfg.DictionaryTable.MaxBrowseRows`), role |
| Max export rows | 100,000 | role |
| Query timeout | 30 s | role (auditor role may be higher) |
| Max concurrent browser queries per user | 2 | global |
| Leading-wildcard search | requires permission | global |
| Result caching | none | — (always fresh; stale financial data is worse than slow) |

Every result carries `IsTruncated` and `RowLimitApplied` so the UI can state
plainly that the user is not seeing everything — a silently truncated table
browser is how people draw wrong conclusions from correct data.

## 8.5 Variants and layouts

| Table | Purpose |
|---|---|
| `cfg.TableBrowserVariant` | Saved selection: table, filters, name, owner, `IsShared`, `IsDefault` |
| `cfg.TableBrowserLayout` | Saved output: field list, order, widths, sort, subtotals, `IsShared` |

Sharing a variant/layout shares *the query shape only* — never data, and never
elevated authorization. A shared variant executed by another user runs entirely
under that user's own scope.

## 8.6 Navigation and documentation

- **Foreign-key navigation**: where `cfg.DictionaryForeignKey` declares a check
  table, the value renders as a link that opens the browser on the target table
  filtered to that key — but only if the user is authorized for the target table;
  otherwise it renders as plain text.
- **Field documentation**: clicking a column header shows the data element's
  labels, description, help text, domain, fixed values, and check table — read
  straight from the dictionary ([07](07-data-dictionary.md)).
- **Record count**: an explicit "Count only" mode that runs `COUNT(*)` under the
  same predicates without materializing rows, for users sizing an extract.

## 8.7 Audit

`audit.TableBrowserQueryLog` — written for **every** execution:

```
Id, TenantId, UserId, ExecutedAt (UTC), Schema, TableName,
FieldsRequested (json), FilterJson, SortJson, RowCount, IsTruncated,
DurationMs, WasExported, ExportFormat, ClientIp, CorrelationId,
UnmaskedFieldsAccessed (json), Outcome (Success/Denied/Timeout)
```

Denied attempts are logged with the reason. A dedicated report ("who looked at
what") is part of the audit report set (§21), because for a finance system the
question *"who extracted the customer list last quarter"* must be answerable.

## 8.8 API

```
POST /api/v1/table-browser/query          → execute (body = TableBrowserQuery)
POST /api/v1/table-browser/count          → count only
GET  /api/v1/table-browser/tables         → authorized, browsable tables (paged)
GET  /api/v1/table-browser/tables/{s}/{t} → column metadata + allowed operators
GET  /api/v1/table-browser/variants       → own + shared variants
POST /api/v1/table-browser/variants       → save variant
POST /api/v1/table-browser/export         → Excel/CSV (permission-gated, audited)
```

All responses use RFC 7807 for errors, with distinct problem types for
`table-not-browsable`, `field-not-authorized`, `row-limit-exceeded`,
`query-timeout`, and `export-not-authorized` — so the UI can explain *why*
instead of showing a generic failure.
