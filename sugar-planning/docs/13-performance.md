# 13. Performance at ten years of history

```bash
./load-test.sh --postgres "postgres://localhost/sugarplan_perf"
```

Section 25 sets two numbers and a volume:

| | |
| --- | --- |
| List APIs | p95 under **500 ms** for normal indexed filters |
| Dashboards | p95 under **2 s**, with caching or pre-aggregation where needed |
| At | "at least 10 years of daily history and high-volume transaction and audit data" |

A target nobody has measured is a wish, and a measurement taken against a
fortnight of demonstration data is a measurement of nothing: every query is fast
when the table has four hundred rows in it.

---

## 13.1 The fixture

Ten seasons of a busy mill, generated in about thirty seconds:

| Table | Rows |
| --- | --- |
| `inventory_document_items` | 616,511 |
| `inventory_documents` | 205,510 |
| `audit_events` | 205,550 |
| `daily_storage_plans` | 22,605 |
| `daily_product_plans` | 11,427 |
| `daily_cane_plans` | 2,891 |

Everywhere else in this repository, data is produced by driving the services,
because a figure written past the rules is a figure that does not reconcile.
**Here that rule is deliberately broken**, and it is worth being explicit: the
point of this fixture is *volume*, not correctness. Ten seasons through the
posting engine one transaction at a time would take hours and would prove
nothing the service tests already prove. What a query planner cares about is how
many rows there are, how they are distributed, and whether the statistics
reflect that — and a `COPY` gives all three in seconds. The rows are not
guaranteed to reconcile with each other; nothing in them should be read as a
business figure.

`-docs-per-day` and `-seasons` scale it. The default is 150 postings per
crushing day, which is a busy mill.

---

## 13.2 Measured twice, on purpose

Every endpoint is measured **alone** and then **under concurrent load**, and the
two answer different questions:

- **Alone** is the query — whether the schema, the indexes and the SQL are
  right. No amount of hardware hides a bad plan.
- **Under load** is the machine — whether this hardware can serve that query to
  that many callers at once.

A report that gave only the second would blame an index for a small server. The
verdict distinguishes them: `SLOW` fails the run, `saturated` does not, and says
so in as many words.

---

## 13.3 What one run finds

Ten seasons, 4 cores, load generator and PostgreSQL sharing them, 8 concurrent
callers:

| | alone p95 | under load p95 | target |
| --- | --- | --- | --- |
| Daily cane, one season | 5 ms | 28 ms | 500 ms |
| Daily production, one season | 4 ms | 17 ms | 500 ms |
| Daily stock ledger | 5 ms | 22 ms | 500 ms |
| Stock position | 1 ms | 4 ms | 500 ms |
| Inventory documents | 32 ms | 196 ms | 500 ms |
| Documents, one month | 23 ms | 113 ms | 500 ms |
| **Documents, one warehouse** | **74 ms** | **494 ms** | **500 ms** |
| Audit trail | 13 ms | 87 ms | 500 ms |
| Executive dashboard | 32 ms | 110 ms | 2000 ms |
| Season summary report | 30 ms | 113 ms | 2000 ms |
| Capacity forecast report | 28 ms | 104 ms | 2000 ms |

Everything is inside target. **Documents filtered to one warehouse is at 494 ms
against 500 ms under load, which is no margin at all** — on this hardware, with
everything co-located. It is the endpoint to watch, and the first one to
re-measure on real hardware.

---

## 13.4 What the first run found, and what was fixed

**The count beside the page cost more than the page.** A page of a hundred
documents is an index scan that stops after a hundred rows: 1.5 ms. The exact
total beside it is an aggregate over every matching row: 84 ms, and it grows
with the table forever. Filtered to one warehouse under eight callers, that was
**p95 829 ms against a 500 ms target**.

The count is now bounded — the match is wrapped in a `LIMIT`, so the planner
stops early and picks a merge semi-join with an early exit instead of a parallel
hash join over the table. **84 ms exact, 21 ms bounded**, and 0.1 ms when the
filter matches nothing. Past `store.CountLimit` (10,000) the response sets
`countCapped`, so a screen can say "10,000+" rather than a number that is a
floor pretending to be a total. Nobody paging a list needs to know there are
exactly 205,510 of them.

**A hundred documents was a hundred round trips.** `ListDocuments` fetched each
document's lines in its own query. The page cost grew with the page size rather
than with the work in it. One `WHERE document_id = ANY($1)` now fills them all.

**An index that could not be used index-only.** The warehouse filter asks "which
documents touched this store?" through an `EXISTS` on the item table. The
existing index is `(warehouse_id, product_id)` — it answers "what is in this
store", but the semi-join had to fetch every matching row from the heap to read
its `document_id`. Migration 0012 adds `(warehouse_id, document_id)`: heap
fetches went from 153,450 to zero.

**And the one that matters most, because it is the one an `EXPLAIN` would have
got wrong.** That index made the isolated query faster — 84 ms to 68 ms — and
made it **worse under concurrency**, 829 ms to 1236 ms. The index-only scan
encouraged a parallel plan, and eight concurrent requests each spawning workers
on four shared cores is slower than not doing that. A plan that wins alone can
lose under load, which is exactly why this harness measures both and why an
`EXPLAIN ANALYZE` on a quiet database is not the measurement that decides
anything.

---

## 13.5 What this does not cover

- **Not a soak test.** Sixty requests an endpoint, not sixty minutes. Nothing
  here would find a leak, a pool that never returns a connection, or an index
  that bloats over a season.
- **Not the write path.** The postings, the plan generator and the import commit
  are not measured. The fixture writes with `COPY`, which is not what the
  posting engine does.
- **One factory.** Data scope narrows every query in production; a single
  tenant's numbers are the pessimistic case for reads and say nothing about
  contention between tenants.
- **Not partitioning.** Section 25 suggests partitioning the posting and audit
  tables by business date "when justified". At 205,510 documents it is not yet
  justified — the indexed reads are 32 ms alone. The number to watch is the
  documents-by-warehouse row above; when it stops fitting, partitioning by
  business date is the next move, and the fixture is how to know.
