# Architecture Decision Records (Phase 1)

The decisions that are expensive to reverse. Reviewers should challenge these
first — the rest of the blueprint follows from them.

---

### ADR-001 · Modular monolith, not microservices
**Context** A posting must atomically write journal header, lines, open items,
asset transactions, CO postings, audit, and outbox.
**Decision** One deployable, one database, thirteen compile-time-separated modules.
**Consequences** Real ACID across the financial core; no sagas for internal
plumbing; simpler operations. Costs: one scaling unit, discipline required to keep
boundaries honest (enforced by architecture tests). Reporting, Integration, and
Workflow are the modules that could be extracted later if needed.
**Rejected** Microservices per module (distributed transactions in a ledger);
single-project monolith (no boundaries at all).

---

### ADR-002 · One universal journal table
**Decision** `fin.JournalEntryLine` carries FI and CO dimensions; subledgers and
CO reporting are predicates over it.
**Consequences** Reconciliation is structural, drill-down is trivial, new
dimensions are additive. Costs: a very wide, very large table — answered with
partitioning by fiscal year, filtered indexes, and a columnstore index on closed
years.
**Rejected** Separate FI and CO documents with a link table (the classic drift
problem); per-subledger tables (permanent reconciliation jobs).

---

### ADR-003 · One posting engine, no exceptions
**Decision** Every module posts via `IPostingEngine`. No direct writes to `fin.*`
transaction tables, enforced by architecture test and database permissions.
**Consequences** Validation, numbering, currency, tax, CO derivation, audit, and
idempotency are implemented once. Future modules add document types and account
determination rules, not posting code.
**Rejected** Per-module posting services (guaranteed divergence in period control
and currency handling).

---

### ADR-004 · Shared-database multi-tenancy with a `TenantId` discriminator
**Decision** Shared schema, `TenantId` on every tenant-owned table, EF global
query filters plus mandatory predicate injection in reporting and SE16N.
**Consequences** One migration set, one dictionary activation. Isolation depends
on correct filtering — mitigated by a test that fails when any tenant-owned entity
lacks a filter, and by centralizing tenant resolution in two classes.
**Rejected** Database-per-tenant (N-fold migration and DDL-activation complexity);
schema-per-tenant (same, plus object-count limits).

---

### ADR-005 · Selective CQRS
**Decision** Full CQRS for postings and complex financial operations; plain
command+aggregate for master data; Dapper query services for reports and browsing.
Reads and writes share one database.
**Consequences** Complexity where it pays, none where it does not. No read-model
lag in financial reports.
**Rejected** Uniform CQRS with projections (a trial balance that lags the posting
it must reconcile with is a defect, not a trade-off).

---

### ADR-006 · Draw document numbers inside the posting transaction
**Decision** Atomic `UPDATE … OUTPUT` on the number-range status row, inside the
transaction; gaps from rollbacks are logged, not prevented.
**Consequences** Duplicate numbers are impossible under concurrency (the stated
requirement). Gaps can occur on rollback and are explained by `NumberRangeGapLog`.
A separate post-commit legal numbering service is designed for jurisdictions that
mandate gapless invoice numbers.
**Rejected** Pre-allocation outside the transaction (crash gaps anyway, plus a
duplicate window); SQL `SEQUENCE` (caching produces larger, unexplained gaps).

---

### ADR-007 · Mandatory idempotency keys on postings
**Decision** Every `PostingRequest` carries an idempotency key persisted with a
unique constraint inside the posting transaction; replays return the original result.
**Consequences** Retries, double-clicks, and job re-runs are safe — critical for
the payment run and the depreciation run. Costs: callers must produce stable keys;
background jobs use deterministic ones.

---

### ADR-008 · Store document, local, and group amounts on every line
**Decision** Rates applied at posting time are stored, never recomputed.
Rounding differences post to a rounding account so the ledger balances *in every
currency*.
**Consequences** Historic documents stay reproducible when configuration changes;
group reporting requires no re-translation. Costs: wider rows, more validation at
step 12.
**Rejected** Storing only document currency and converting at read time (the
answer would change over time — unacceptable in accounting).

---

### ADR-009 · Posted documents are immutable, enforced in four layers
**Decision** Domain (no mutators), EF interceptor, column-level database grants,
audit. Only clearing-status columns are updatable after posting.
**Consequences** Corrections must be reversals or adjustments. An ORM bypass still
cannot rewrite an amount.
**Rejected** Application-only enforcement (a script with a connection string
defeats it).

---

### ADR-010 · Metadata-first: the dictionary is a runtime contract
**Decision** Labels, help, F4, validation, browsability, masking, and change
logging all derive from `cfg.Dictionary*`. Core tables are seeded into the
dictionary, not just custom ones.
**Consequences** Consistent UI, translatable by data, extensible without code.
Costs: dictionary metadata must be maintained with the schema — enforced by a
Phase-2 completeness test (100 % field coverage).

---

### ADR-011 · Dictionary activation generates reviewed migrations; the UI never runs production DDL
**Decision** Validate → impact review → generated EF migration + rollback script →
approval (author ≠ approver) → CI/CD deployment → audit.
**Consequences** Schema history lives in version control; production changes are
reviewable and reversible. Costs: slower turnaround than click-to-apply — accepted
deliberately for a financial system.

---

### ADR-012 · Custom fields become real, typed columns in per-host extension tables
**Decision** `ext.<Host>Ext` tables with generated columns; JSON only for
explicitly non-reportable, non-accounting annotations; custom tables live in the
`z` schema.
**Consequences** Custom data is indexable, reportable, FK-capable, and visible to
SE16N. Core tables stay under our control. Costs: a migration per field change —
which is the point.
**Rejected** EAV; JSON-only extension (both violate §12.3).

---

### ADR-013 · Authorization is organizational and evaluated in the query
**Decision** Decisions are `(user, T-code, activity, org scope, object state)`;
scope predicates are injected into queries rather than filtered after retrieval;
UI hiding is usability only.
**Consequences** SE16N and reporting cannot read around authorization. Costs:
every query path must go through the scope provider — covered by tests that run
identical queries as differently-scoped users.

---

### ADR-014 · Hangfire over Quartz.NET
**Decision** Hangfire for background processing.
**Consequences** Persistent job storage in SQL Server, built-in retries, and a job
history/dashboard that satisfies the audit question "who ran the payment run, when,
with what outcome". Costs: a schema in the application database; less flexible
scheduling grammar than Quartz — sufficient for period-driven finance jobs.
