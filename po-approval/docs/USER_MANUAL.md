# PO Approval — User Manual

This guide walks an end user through approving purchase orders in the PO
Approval app. It assumes the database, backend API and UI5 app are running (see
the [README](../README.md)).

## 1. Logging on

1. Open the app (`http://localhost:8080`). You are taken to the **Log On** page.
2. Enter your **user name** and **password** and choose **Log On**.
   - Demo users (password `Welcome1`): `jdoe`, `msmith`, `klee`, `rbuyer`.
3. On success you land on the **Release Worklist**. Your name is shown top-right,
   with a **Log Off** button.

> Your session is remembered for the browser tab. If your session expires, the
> app returns you to the Log On page automatically.

## 2. The release worklist

The worklist lists purchase orders and their release status.

| Column | Meaning |
|--------|---------|
| **Purchase Order** | PO number and document type |
| **Vendor** | supplier name and number |
| **Purchasing Group** | responsible buying group |
| **Net Value** | total order value and currency |
| **Strategy** | the determined release strategy |
| **Release Status** | ⏳ *Blocked – release pending* · ✔ *Released* |
| **Action** | *Awaiting you* when the PO is waiting for **your** release code |

Controls:

- **Search** — filter by PO number or vendor name.
- **Pending … release only** switch — on by default; turn it off to also see
  fully released POs.
- **Refresh** — reload the list.

Select a row to open the **purchase order detail**.

## 3. Purchase order detail

The header shows the vendor, value, document type, purchasing group, the
**release indicator** and the **strategy**. Three tabs follow:

- **Release Strategy** — each sign-off **step** with its release code and status:
  ✔ released (with who released it) or ⏳ pending. The **current step** (the one
  awaiting sign-off) is highlighted.
- **Items** — the line items with quantities, prices and net values.
- **Audit Log** — every release/reject action, with user, time and note.

At the bottom are the **Release** and **Reject** buttons. They are enabled only
when *you* are the approver for the current step (i.e. you hold the release code
the PO is waiting for).

## 4. Releasing a purchase order

1. Open a PO whose current step is your release code (look for *Awaiting you* in
   the worklist).
2. Review the items and value.
3. Choose **Release** and confirm.
4. The strategy updates:
   - if more steps remain, the PO stays **Blocked** and moves to the next code;
   - if yours was the last step, the PO becomes **Released** (✔).

*Example:* PO 4500000002 (€12,500, strategy `[01,02]`) already has code **01**
released. Log on as `msmith` (code 02), open it, and choose **Release** — the PO
becomes fully **Released**.

## 5. Rejecting a release

1. Open the PO and choose **Reject**.
2. Enter a **reason** (required) and confirm.
3. The release strategy is reset from your step: any later approvals are cleared
   and the PO is **Blocked** again, starting from your code. The rejection and
   its reason are recorded in the **Audit Log**.

## 6. Who can release what

Approval follows the strategy determined by the order value:

| Order value | Strategy | Steps (in order) |
|-------------|----------|------------------|
| up to €5,000 | S1 | 01 Department Manager |
| €5,000 – 25,000 | S2 | 01 Department Manager → 02 Finance Controller |
| €25,000 and above | S3 | 01 → 02 → 03 Chief Financial Officer |

You can only release a step whose **release code you hold**. If you try to
release a step that is not yours, the app reports that you are not authorised.

## 7. Logging off

Choose **Log Off** (top-right). Your token is discarded and you return to the
Log On page.
