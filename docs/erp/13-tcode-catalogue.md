# 13. Transaction-Code Framework & Catalogue

## 13.1 What a T-code is here

A transaction code is **a configurable navigation alias bound to a route and an
authorization object** (§9). It is data, not code:

```
sec.TransactionCode
  TCode              'FB50'
  Name               'Enter G/L Journal'
  ModuleKey          'FIN'
  Route              '/gl/journal/new'
  AuthorizationObjectId → F_JOURNAL
  DefaultActivity    10 (Post)
  ParameterTemplate  '{"documentType":"SA"}'      -- optional route defaults
  IconKey            'journal'
  IsActive           1
```

Consequences:

- A new screen becomes reachable by adding a row, not by editing a menu component.
- Roles grant **T-codes**, and the T-code carries the authorization object — so
  "what can this user do" is answerable as a list, which is what auditors ask for.
- Custom objects ([09](09-customization-framework.md)) register their own T-codes
  (`ZCN01`) and appear in the command box alongside standard ones.

## 13.2 Global command box (§9)

One input, keyboard-first (`Ctrl/⌘ + K`), that resolves in this order:

| Input pattern | Resolves to |
|---|---|
| Known T-code (`FB50`, `/nFB50`) | Navigate to the route (activity checked first) |
| `/n<tcode>` / `/o<tcode>` | Navigate / open in new tab (familiar muscle memory, original implementation) |
| Application name text (`journal`, `កត់ត្រា`) | Fuzzy match on T-code names in the user's language |
| Numeric, matches a BP number range | Business Partner |
| Numeric, matches a customer/vendor account | BP in that role |
| Matches a G/L account in the current CoA | G/L account master |
| Matches an asset number | Asset master |
| `<company>/<year>/<docnumber>` or a document number | Accounting document (FB03) |
| Report name | Report launcher with that report |

Results are grouped by type, ranked by the user's recent usage, and filtered to
what the user is authorized for — an unauthorized object is not listed at all
(no existence disclosure).

Also in the shell: **Favorites** (pin any T-code or object), **Recent
transactions** (last 20, per user), and per-role default start pages.

## 13.3 Catalogue

### Configuration

| T-Code | Function | Route | Auth object · activity |
|---|---|---|---|
| `SPRO` | Configuration center | `/configuration` | `S_CONFIG` · 03 |
| `OBY6` | Company code settings | `/configuration/company-codes` | `S_CONFIG` · 02 |
| `OB13` | Chart of accounts | `/configuration/chart-of-accounts` | `S_CONFIG` · 02 |
| `OB52` | Posting periods | `/configuration/posting-periods` | `S_PERIOD` · 02 |
| `OB29` | Fiscal year variants | `/configuration/fiscal-year-variants` | `S_CONFIG` · 02 |
| `OBA7` | Document types | `/configuration/document-types` | `S_CONFIG` · 02 |
| `SNRO` | Number ranges | `/configuration/number-ranges` | `S_NUMRANGE` · 02 |
| `OB08` | Exchange rates | `/configuration/exchange-rates` | `S_RATES` · 02 |
| `OBCP` | Posting keys | `/configuration/posting-keys` | `S_CONFIG` · 02 |
| `OBC4` | Field status variants | `/configuration/field-status` | `S_CONFIG` · 02 |
| `FTXP` | Tax codes | `/configuration/tax-codes` | `S_TAX` · 02 |
| `OB58` | Financial statement versions | `/configuration/fsv` | `S_CONFIG` · 02 |

### Master data

| T-Code | Function | Route | Auth object · activity |
|---|---|---|---|
| `BP` | Business Partner (create/change/display) | `/business-partners` | `BP_MASTER` · 01/02/03 |
| `BUP1` / `BUP2` / `BUP3` | Create / change / display BP | `/business-partners/new` · `/{id}/edit` · `/{id}` | `BP_MASTER` · 01/02/03 |
| `BP_ROLE` | Maintain BP roles | `/business-partners/{id}/roles` | `BP_MASTER` · 02 |
| `BP_SYNC` | Synchronize customer/vendor roles | `/business-partners/{id}/sync` | `BP_MASTER` · 02 |
| `BP_CHECK` | BP consistency check | `/business-partners/consistency` | `BP_MASTER` · 03 |
| `FS00` | G/L account maintenance | `/gl/accounts` | `F_GLACCT` · 01/02/03 |
| `AS01` / `AS02` / `AS03` | Create / change / display asset | `/assets/new` · `/{id}/edit` · `/{id}` | `A_ASSET` · 01/02/03 |
| `KS01` / `KS02` / `KS03` | Create / change / display cost center | `/co/cost-centers/…` | `CO_COSTCTR` · 01/02/03 |
| `KE51` / `KE52` / `KE53` | Create / change / display profit center | `/co/profit-centers/…` | `CO_PRCTR` · 01/02/03 |
| `KO01` / `KO02` / `KO03` | Create / change / display internal order | `/co/internal-orders/…` | `CO_ORDER` · 01/02/03 |

### Financial transactions

| T-Code | Function | Route | Auth object · activity |
|---|---|---|---|
| `FB50` | Enter G/L journal (enjoy-style single screen) | `/gl/journal/new` | `F_JOURNAL` · 10 |
| `FB01` | Post accounting document (full, all account types) | `/gl/document/new` | `F_JOURNAL` · 10 |
| `FB02` | Change permitted fields of a posted document | `/gl/document/{id}/edit` | `F_JOURNAL` · 02 |
| `FB03` | Display accounting document | `/gl/document/{id}` | `F_JOURNAL` · 03 |
| `FB08` | Reverse accounting document | `/gl/document/{id}/reverse` | `F_JOURNAL` · 85 |
| `FBV0` | Post/delete parked document | `/gl/parked` | `F_JOURNAL` · 10 |
| `F-28` | Incoming payment | `/ar/payments/incoming` | `F_PAYMENT` · 10 |
| `F-53` | Outgoing payment | `/ap/payments/outgoing` | `F_PAYMENT` · 10 |
| `F110` | Automatic payment run | `/ap/payment-run` | `F_PAYRUN` · 10 |
| `F150` | Dunning run | `/ar/dunning` | `F_DUNNING` · 10 |
| `FB1S`/`FB1D`/`FB1K` | Clear G/L · customer · vendor | `/gl|ar|ap/clearing` | `F_CLEARING` · 10 |
| `AFAB` | Post depreciation | `/assets/depreciation-run` | `A_DEPREC` · 10 |
| `ABZON` | Asset acquisition | `/assets/{id}/acquire` | `A_ASSET` · 10 |
| `ABAVN` | Asset retirement | `/assets/{id}/retire` | `A_ASSET` · 10 |
| `KO88` | Settle internal order | `/co/internal-orders/{id}/settle` | `CO_ORDER` · 10 |
| `KSU5` | Execute assessment cycle | `/co/allocations/assessment` | `CO_ALLOC` · 10 |
| `KSV5` | Execute distribution cycle | `/co/allocations/distribution` | `CO_ALLOC` · 10 |
| `FAGLGVTR` | Balance carry-forward | `/gl/year-end/carry-forward` | `F_CLOSE` · 10 |
| `FAGL_VAL` | Foreign-currency valuation | `/gl/period-end/fx-valuation` | `F_CLOSE` · 10 |

### Reporting & line items

| T-Code | Function | Route |
|---|---|---|
| `FBL1N` | Vendor line items | `/ap/line-items` |
| `FBL3N` | G/L line items | `/gl/line-items` |
| `FBL5N` | Customer line items | `/ar/line-items` |
| `FS10N` | G/L account balances | `/gl/balances` |
| `FD10N` / `FK10N` | Customer / vendor balances | `/ar/balances` · `/ap/balances` |
| `F.01` | Balance sheet / P&L (FSV) | `/reports/financial-statements` |
| `S_ALR_TB` | Trial balance | `/reports/trial-balance` |
| `AR01` | Asset register | `/reports/asset-register` |
| `AR02` | Asset history sheet | `/reports/asset-history` |
| `KSB1` | Cost center line items | `/co/cost-centers/line-items` |
| `KOB1` | Internal order line items | `/co/internal-orders/line-items` |
| `GR55` | Report launcher (all reports) | `/reports` |

### Tools & administration

| T-Code | Function | Route | Auth object · activity |
|---|---|---|---|
| `SE11` | Data Dictionary | `/dictionary` | `S_DICT` · 03/02 |
| `SE16N` | General table browser | `/table-browser` | `S_TABLE` · 03 |
| `ZTAB` | Custom table designer | `/customization/tables` | `S_CUSTOM` · 02 |
| `ZFLD` | Custom field designer | `/customization/fields` | `S_CUSTOM` · 02 |
| `ZCR` | Change requests | `/customization/change-requests` | `S_CHANGE` · 02 |
| `SU01` | User maintenance | `/admin/users` | `S_USER` · 01/02/03 |
| `PFCG` | Role maintenance | `/admin/roles` | `S_ROLE` · 01/02/03 |
| `SUIM` | Authorization / SoD reports | `/admin/authorization-reports` | `S_ROLE` · 03 |
| `SM04` | Active sessions | `/admin/sessions` | `S_ADMIN` · 03 |
| `SM37` | Background jobs | `/admin/jobs` | `S_ADMIN` · 03 |
| `SLG1` | Audit log viewer | `/admin/audit-log` | `S_AUDIT` · 03 |
| `SWI1` | Workflow instances | `/admin/workflow` | `S_WORKFLOW` · 03 |
| `INBOX` | Approval inbox | `/approvals` | `S_APPROVE` · 43 |
| `API1` | API clients & webhooks | `/admin/integration` | `S_INTEGRATION` · 02 |

> Familiar codes are used as *aliases* so experienced finance users are productive
> immediately. They are configuration rows in our own table pointing at our own
> routes and our own authorization objects — no SAP screen, code, or data
> definition is reproduced. Any customer may rename them, and a customer-defined
> alias set (e.g. all-Khmer codes) is a supported configuration.
