# ErpS4.Database — EF Core model for the S/4HANA-inspired ERP

228 entities and their `IEntityTypeConfiguration<T>` classes, generated from
[`docs/s4hana/table_catalogue.csv`](../../docs/s4hana/README.md) — the same
source the SQL scripts in [`database/s4hana`](../../database/s4hana) come from,
so the model and the physical schema cannot drift.

```
ErpS4.Database/
├── Entities/
│   ├── Abstractions.cs          hand written — ITenantScoped, IAppendOnly
│   └── <Schema>Entities.cs      generated  — 228 entity classes
├── Configurations/
│   └── <Schema>Configurations.cs generated — 228 configurations
├── ErpDbContext.cs              hand written — filters, audit stamping, write policy
├── ErpDbContext.Sets.cs         generated  — 228 DbSet properties
├── AmbientContext.cs            hand written — ITenantProvider, ICurrentUser
└── DependencyInjection.cs       hand written — AddErpDatabase(...)
```

Regenerate the three generated groups after editing the catalogue:

```bash
python3 tools/generate_table_catalogue.py   # markdown -> CSV
python3 tools/generate_ef_core.py           # CSV -> entities + configurations
```

## Usage

```csharp
services.AddScoped<ITenantProvider, HttpTenantProvider>();   // your resolution
services.AddScoped<ICurrentUser, HttpCurrentUser>();
services.AddErpDatabase(configuration.GetConnectionString("ErpS4")!);
```

```csharp
var openItems = await db.OpenItem
    .AsNoTracking()                                  // read model: never tracked
    .Where(o => o.Status == "Open" && o.DueDate < today)
    .OrderBy(o => o.DueDate).ThenBy(o => o.Id)       // stable, unique tie-break
    .Skip(page * size).Take(size)
    .ToListAsync(cancellationToken);
```

The tenant filter is applied automatically — there is no `TenantId` predicate in
that query, and one cannot be forgotten.

## Decisions worth knowing before you extend this

**The SQL scripts own the schema; this model is mapped onto it.** Do not run
`dotnet ef migrations add` against this context: EF would want to create the
indexes it infers for foreign keys and would fight the hand-tuned index set in
`91_indexes.sql`. Change the catalogue, regenerate both sides, and ship the
generated migration through the dictionary's approval flow instead.

**No navigation properties.** Relationships are configured with
`HasOne<TPrincipal>().WithMany().HasForeignKey(...)`, which gives EF the full
relational model — cascade behaviour, constraint names, required/optional —
without 800 navigation properties nobody asked for. Add navigations by hand to
`Abstractions.cs`-style partials where an aggregate genuinely needs them
(`JournalEntryHeader` → `JournalEntryLine` is the obvious first one).

**Deletes are `DeleteBehavior.Restrict`,** matching `NO ACTION` in the scripts.
Nothing in a financial system should disappear because its parent did.

**Business-key foreign keys are in the database, not in this model.** Columns
like `LocalCurrencyCode` are constrained in SQL against
`cfg.Currency (TenantId, CurrencyCode)`. Modelling those in EF would require an
alternate key on every code table and would push EF to emit its own unique
constraints, duplicating the ones the scripts already create. The database is
the enforcement point; EF just reads and writes the column.

**`SaveChanges` applies four policies** (see `ErpDbContext.ApplyWritePolicies`):
tenant defaulting on insert, `CreatedAt`/`CreatedBy` and
`ModifiedAt`/`ModifiedBy` stamping, `CreatedAt`/`CreatedBy`/`TenantId` frozen
against later edits, and a hard refusal to update or delete an `IAppendOnly`
entity (audit log, change documents, login history, technical logs).

**Posted-document immutability is not enforced here.** It is a business rule
with exceptions — clearing fields stay writable, corrections are made by
reversal — so it belongs to the posting engine, not to a context-level marker.

**Concurrency.** Every mutable table has a `rowversion` mapped with
`IsRowVersion()`, so a conflicting update raises
`DbUpdateConcurrencyException`; return HTTP 409 with a reload option rather
than overwriting another user's work.

**Property names.** A C# property may not carry the name of its own class, so
48 columns are mapped under a suffixed name — `org.Plant.Plant` becomes
`Plant.PlantCode`, `cfg.TaxCode.TaxCode` becomes `TaxCode.TaxCodeKey`. Each one
carries an explicit `HasColumnName(...)`; the database column keeps the SAP-like
name. `tools/generate_ef_core.py` prints the full list.

## Target framework

`net10.0` with **EF Core 10.0.10** (`Microsoft.EntityFrameworkCore.SqlServer`),
matching the design prompt. Requires the .NET 10 SDK:

```bash
dotnet --version          # 10.0.x
dotnet build backend/ErpS4.Database
```

The HR module in `backend/HRModule.Api` stays on `net8.0`; the two target
frameworks coexist in one repository without a shared solution constraint.

Everything this project uses — `HasQueryFilter`, `ApplyConfigurationsFromAssembly`,
`IsRowVersion`, `HasPrecision`, `EnableRetryOnFailure`, `TimeProvider` — is
stable API in EF Core 10. Two EF 10 features are worth adopting as the
application layer grows, though nothing here depends on them: **named query
filters**, which would let a company-code filter sit alongside the tenant filter
instead of being `AND`-ed into it, and the **complex types** support that suits
the repeating amount/currency pairs on the journal line.

> Built with .NET SDK 10.0.110 and EF Core 10.0.10, and run against SQL Server
> 2025 (17.0.4065.4). The 228 tables, 837 foreign keys and 366 indexes install
> from `database/s4hana/run_all.sql` and were counted back out of `sys.tables`,
> `sys.foreign_keys` and `sys.indexes`.
