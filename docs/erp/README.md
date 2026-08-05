# ERP Platform — Phase 1: Solution Blueprint

> **Status: FOR REVIEW — no production code is to be written until this
> blueprint is approved.**
> Deliverable of Phase 1 as defined in the project brief (§25). Everything here
> is design: architecture, models, schema strategy, and roadmap.

An original, web-based ERP built on ASP.NET Core 10 / SQL Server 2025 / React,
**inspired by the business processes** of SAP S/4HANA. SAP concepts (enterprise
structure, Business Partner, transaction codes, posting keys, fiscal periods,
document flow, universal journal, the purpose of SE11 and SE16N) are used as
*functional references only*. No SAP source code, table definitions, or screen
designs are reproduced — all table names, keys, APIs, and layouts in this
blueprint are original.

---

## How to read this blueprint

The twelve items requested for Phase 1 map to these documents:

| # | Phase-1 item | Document |
|---|--------------|----------|
| 1 | Proposed architecture | [01-architecture.md](01-architecture.md) |
| 2 | Module boundaries | [02-module-boundaries.md](02-module-boundaries.md) |
| 3 | Enterprise-structure model | [03-enterprise-structure.md](03-enterprise-structure.md) |
| 4 | Business Partner + customer/vendor synchronization | [04-business-partner.md](04-business-partner.md) |
| 5 | Universal journal & core accounting model | [05-universal-journal.md](05-universal-journal.md) |
| 5b | Central posting engine & currency architecture | [06-posting-engine.md](06-posting-engine.md) |
| 6 | Data Dictionary (SE11-like) metadata architecture | [07-data-dictionary.md](07-data-dictionary.md) |
| 7 | Table Browser (SE16N-like) query & authorization architecture | [08-table-browser.md](08-table-browser.md) |
| 8 | User-defined tables & custom fields framework | [09-customization-framework.md](09-customization-framework.md) |
| 9 | User & authorization model | [10-security-model.md](10-security-model.md) |
| 10 | Project structure | [11-project-structure.md](11-project-structure.md) |
| 11 | Database schema strategy | [12-database-schema-strategy.md](12-database-schema-strategy.md) |
| — | T-Code catalogue (annex) | [13-tcode-catalogue.md](13-tcode-catalogue.md) |
| 12 | Implementation roadmap | [14-roadmap.md](14-roadmap.md) |
| — | Financial process diagrams (§16) | [processes.md](processes.md) |
| — | Phase-1 closing report (§26.17) | [15-phase1-report.md](15-phase1-report.md) |

Cross-cutting design rationale — the decisions that are expensive to reverse —
is collected in [decisions.md](decisions.md) (ADR-001 … ADR-014). **Reviewers
should start there**: those fourteen decisions are what the rest of the
blueprint is built on.

---

## Executive summary

**Shape.** A *modular monolith*: one ASP.NET Core 10 deployable, one SQL Server
database, thirteen independently-compiled bounded modules that talk to each
other only through published contracts and in-process integration events. This
gives a DDD module boundary without distributed-transaction pain — critical,
because a financial posting must commit **atomically** across FI, CO, Assets,
subledger open items, and audit in a single SQL transaction. Modules are
physically separated so that any of them can be lifted into its own service
later if scale demands it, but the *posting boundary* never will be.

**The financial core is one table.** All accounting — G/L, receivables,
payables, assets, cost centers, profit centers, internal orders — writes lines
to a single universal journal (`fin.JournalEntryLine`) that carries both the
financial and the managerial dimensions on every line. Subledgers and CO are
*views over* that table, not parallel bookkeeping. This is what makes "every
report reconciles with the journal" (§26.10) true by construction instead of by
reconciliation job.

**One posting engine.** No module writes to the journal directly. Every module —
today FI/AR/AP/AA/CO, tomorrow Purchasing, Inventory, Sales, Production,
Payroll, Projects, Maintenance, QM — submits a `PostingRequest` to
`IPostingEngine`, which runs the 19-step pipeline (§15), is idempotent on a
caller-supplied key, and either commits everything or nothing. Adding a future
module means adding a *document type and account determination rules*, not a new
accounting path.

**Extensibility is metadata-first, not JSON-first.** SE11-like dictionary
objects describe every table; custom (`Z*`/`Y*`) tables and custom (`ZZ*`)
fields become **real, typed, indexed SQL columns** generated through a reviewed
migration and an approval gate — never uncontrolled DDL from a web page, and
never important accounting values buried in an unvalidated JSON blob (§12.3).

**Security is organizational, not just role-based.** Authorization is evaluated
as *(transaction code × activity × organizational scope)* — tenant, company
code, cost center, profit center — enforced in the application layer and again
as EF Core global query filters, so SE16N and the reporting APIs cannot be used
to read around it.

---

## Scope of Phase 1

**In scope (this document set):** architecture, module boundaries, domain
models, ER diagrams, schema strategy, security model, dictionary and table
browser design, customization framework, T-code catalogue, project structure,
roadmap, and the accounting decisions behind them.

**Out of scope until approved:** DDL, EF Core configurations, migrations, C#
projects, React application, tests, Docker, CI/CD. Those are Phases 2–5.

---

## Relationship to the existing HR module in this repository

This repository already contains an SAP-ECC-style **HR/HCM module** (SAPUI5 +
ASP.NET Core 8 + SQL Server) under `backend/`, `frontend/`, and `database/`.
It is **not** modified or migrated by this blueprint. The ERP is a new solution
tree (`erp/`, see [11-project-structure.md](11-project-structure.md)).

The two meet at exactly two planned seams, both deferred past the initial
modules:

1. **Employee ↔ Business Partner** — the HR personnel number (PERNR) maps to a
   Business Partner carrying the *Employee* role, so travel expense, vendor-like
   employee reimbursement, and cost-center ownership resolve to one identity.
2. **Payroll → posting engine** — HR payroll results post to the universal
   journal through the same `IPostingEngine` contract as any other module
   (§20 "Payroll integration"), never by writing journal rows directly.

Both seams are one-directional (HR → ERP), so the ERP financial core carries no
dependency on the HR module.
