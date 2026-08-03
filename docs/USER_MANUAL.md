# HR Module — User Manual

**Human Capital Management (HCM)** · modelled on SAP ECC 6.0 EHP8
Version 1.0

---

## Table of contents

1. [Introduction](#1-introduction)
2. [Key concepts](#2-key-concepts-read-this-first)
3. [Getting started](#3-getting-started)
4. [Navigation overview](#4-navigation-overview)
5. [Working with employees](#5-working-with-employees)
   - 5.1 [Employee list & search](#51-employee-list--search)
   - 5.2 [Using the key date (time travel)](#52-using-the-key-date-time-travel)
   - 5.3 [Hiring an employee](#53-hiring-an-employee)
   - 5.4 [Viewing employee master data](#54-viewing-employee-master-data)
   - 5.5 [Changing personal data](#55-changing-personal-data)
   - 5.6 [Organizational reassignment](#56-organizational-reassignment)
6. [Time & leave](#6-time--leave)
   - 6.1 [Viewing leave balances](#61-viewing-leave-balances)
   - 6.2 [Recording an absence](#62-recording-an-absence)
7. [Organizational Management](#7-organizational-management)
   - 7.1 [Organizational structure (org chart)](#71-organizational-structure-org-chart)
   - 7.2 [Positions](#72-positions)
8. [Field reference](#8-field-reference)
9. [Frequently asked questions](#9-frequently-asked-questions)
10. [Troubleshooting](#10-troubleshooting)
11. [Glossary (SAP terms)](#11-glossary-sap-terms)

---

## 1. Introduction

The HR Module is a web application for managing employee master data,
organizational structures and time/leave, following the concepts of the **SAP
HCM** module. It has three functional areas:

| Area | SAP term | What you do here |
|------|----------|------------------|
| **Personnel Administration** | PA | Hire employees, maintain personal, organizational, pay, address and bank data |
| **Organizational Management** | OM | View the org structure, org units, positions and reporting lines |
| **Time Management** | PT | View leave balances and record absences |

**Who this manual is for:** HR administrators and HR business partners who
maintain employee records. No technical knowledge is required.

---

## 2. Key concepts (read this first)

Understanding two SAP concepts will make everything else clear.

### Infotypes
Employee data is grouped into **infotypes** — logical blocks of related fields,
each with a number. You'll see references such as *IT0002* throughout the app.

| Infotype | Name | Contains |
|----------|------|----------|
| IT0000 | Actions | Hiring, leaving and other personnel actions |
| IT0001 | Organizational Assignment | Company, personnel area, org unit, position |
| IT0002 | Personal Data | Name, date of birth, gender, marital status |
| IT0006 | Addresses | Home / mailing address |
| IT0007 | Planned Working Time | Work schedule, weekly hours, employment % |
| IT0008 | Basic Pay | Pay scale, salary, wage types |
| IT0009 | Bank Details | Bank account for payment |
| IT0105 | Communication | E-mail, phone |
| IT2001 | Absences | Recorded absences (leave, sickness…) |
| IT2006 | Absence Quotas | Leave entitlement and balance |

### Time-dependency (validity periods)
**Every record is valid for a period**, defined by a **start date (Valid From)**
and an **end date (Valid To)**. An open-ended record ends on **31.12.9999**,
shown in the app as **"unlimited"**.

This means the system always knows what was true *on any given date*. When you
change data, you don't overwrite history — you create a **new time slice** that
starts on your chosen date, and the previous record is automatically closed the
day before (this is SAP **time constraint 1**). Example:

```
Before change:   Personal Data   01.03.2020 ──────────────▶ 31.12.9999 (unlimited)

After a change effective 01.08.2026:
                 Personal Data   01.03.2020 ──▶ 31.07.2026   (old, now closed)
                 Personal Data   01.08.2026 ──────────────▶ 31.12.9999 (new)
```

> **Why it matters:** always set the **Valid From** date to the date the change
> actually takes effect — not necessarily today.

---

## 3. Getting started

### Opening the application
1. Make sure the backend API and the database are running (see the project
   `README.md` if you are also responsible for setup).
2. Open your browser at **`http://localhost:8080`**.
3. The **Employees** screen appears — this is the home screen.

> This version has no login step; it is intended for an internal, trusted
> environment. Access control would be added before productive use.

### Demo data
A fresh installation contains a small example organization so you can explore
immediately:

| Pers.No. | Name | Position | Org unit |
|----------|------|----------|----------|
| 1000 | Andreas Schmidt | Head of Human Resources | Human Resources |
| 1001 | Linda Nguyen | HR Specialist | Human Resources |

Org structure: **Executive Board** → **Human Resources**, **Finance**.

---

## 4. Navigation overview

The app has three main screens, reachable from the buttons in the top-right of
the header:

| Button | Screen | Purpose |
|--------|--------|---------|
| *(home)* | **Employees** | List, search and open employees; hire |
| **Org Chart** | **Organizational Structure** | Browse the org hierarchy |
| **Positions** | **Positions** | List positions and who holds them |

Use the **back arrow** (top-left) to return to the previous screen. The
**Refresh** icon reloads the current data.

---

## 5. Working with employees

### 5.1 Employee list & search
The **Employees** screen lists everyone with, per row: personnel number, name,
organizational unit, position, e-mail and employment status (colour-coded —
green = *Active*).

- **Search:** type in the search box to filter by **name** or **personnel
  number**. Filtering happens as you type.
- **Open an employee:** click a row to open the detail screen.
- **Count:** the header shows how many employees match.

### 5.2 Using the key date (time travel)
The **Key date** picker in the sub-header controls *the date on which you view
the data*. Because records are time-dependent (see §2), changing the key date
shows you the organization **as it was / will be** on that date.

- Leave it empty to view **today's** situation (the default).
- Set it to a past date to see historical assignments; a future date to see
  planned ones.

The key date you pick on the list also drives the data shown when you open an
employee.

### 5.3 Hiring an employee
Hiring is a **personnel action** that creates the new employee and the mandatory
infotypes (IT0000/0001/0002) in one step. The system assigns the next free
**personnel number** automatically.

1. On the **Employees** screen, click **Hire** (top-right).
2. Complete the form. Required fields are marked with a red asterisk:
   - **Personal Data:** First name\*, Last name\*, Gender, Date of birth, E-mail
   - **Organizational Assignment:** Hiring date\*, Company code, Personnel area,
     Employee group, Employee subgroup, Organizational unit
3. Click **Hire**.
4. A confirmation appears with the new personnel number, and the new employee's
   detail screen opens automatically.

> **Tip:** the **Hiring date** becomes the *Valid From* date of the first
> records. Set it to the employee's actual start date.

### 5.4 Viewing employee master data
The detail screen shows a header (name, personnel number, org unit, position)
and a set of tabs — one per group of infotypes:

| Tab | Shows | Infotypes |
|-----|-------|-----------|
| **Personal Data** | Name, birth, gender, marital status, validity | IT0002 |
| **Organizational Assignment** | Company, area, group, org unit, position, cost centre | IT0001 |
| **Address & Communication** | Home address; e-mail / phone list | IT0006, IT0105 |
| **Working Time & Pay** | Work schedule; basic pay, wage types, bank details | IT0007, IT0008, IT0009 |
| **Time & Leave** | Leave balances | IT2006 |

All data reflects the **key date** shown in the header status.

### 5.5 Changing personal data
Use this to correct or update name, gender, marital status, etc. A **new time
slice** is created (see §2); prior history is preserved.

1. On the detail screen, click **Edit Personal Data**.
2. Set **Valid From** to the date the change takes effect.
3. Change the fields as needed.
4. Click **Save**. A confirmation appears and the screen refreshes.

### 5.6 Organizational reassignment
Use this when an employee moves to a different org unit, position or cost
centre. The current assignment is **delimited** and a new one starts on your
chosen date. Fields you leave blank are **carried forward** from the current
record.

1. On the detail screen, click **Reassign**.
2. Set **Valid From** (the move's effective date).
3. Choose a new **Organizational unit** and/or **Position**, and/or enter a new
   **Cost center**. Leave anything unchanged that should stay the same.
4. Click **Save**.

---

## 6. Time & leave

### 6.1 Viewing leave balances
Open an employee and select the **Time & Leave** tab. For each quota type you
see:

- **Entitlement** — total granted for the period
- **Deducted** — amount already used
- **Remaining** — balance still available (green if positive)
- **Validity** — the period the quota applies to

### 6.2 Recording an absence
Recording an absence (e.g. annual leave) automatically **deducts** it from the
matching leave quota.

1. On the detail screen, click **Record Absence**.
2. Select the **Absence type** (e.g. *Annual leave*, *Sick leave*).
3. Enter **Valid From** and **Valid To**.
4. Optionally enter **Days**. If you leave it empty, the system calculates the
   number of calendar days from the dates.
5. Click **Submit**.

> **Note:** if the remaining quota is smaller than the absence, the system
> **rejects** the entry with a message. Reduce the days or check the balance.

---

## 7. Organizational Management

### 7.1 Organizational structure (org chart)
Click **Org Chart** to see the org hierarchy as an expandable tree. Each node
shows:

- The **org unit name**
- The **Org ID**
- The **Manager** (the chief position of that unit), when defined
- A **head-count** badge (number of employees assigned)

Expand/collapse nodes with the triangle to the left of each row.

### 7.2 Positions
Click **Positions** to list all positions. Each row shows the position, its org
unit, the **job** it is described by, the current **holder**, and an
**Occupancy** status:

- **Occupied** (green) — a person holds the position
- **Vacant** (amber) — the position is unfilled

Use the **Organizational Unit** filter in the sub-header to show only the
positions of one unit.

---

## 8. Field reference

Common fields and their SAP equivalents:

| Field on screen | SAP field | Notes |
|-----------------|-----------|-------|
| Pers.No. | PERNR | Unique personnel number, assigned at hiring |
| Company Code | BUKRS | Legal/accounting entity |
| Personnel Area | WERKS | Location / sub-organization |
| Employee Group | PERSG | e.g. Active employee, Pensioner |
| Employee Subgroup | PERSK | e.g. Salaried, Industrial |
| Organizational Unit | ORGEH | Department in the org structure |
| Position | PLANS | The post the employee occupies |
| Job | STELL | The generic role a position is based on |
| Cost Center | KOSTL | Cost accounting assignment |
| Valid From / To | BEGDA / ENDDA | Record validity period |

---

## 9. Frequently asked questions

**Q: I changed data but the old value still shows.**
Check the **key date**. If the key date is before your change's *Valid From*
date, you are looking at the older time slice. Set the key date to (or after)
the effective date.

**Q: Why can't I just overwrite a value?**
The system keeps history on purpose. Every change creates a new dated record so
you can always see what was true at any point in time (payroll, audits, etc.).

**Q: The employee list is empty.**
Either no employees match your search, or the key date is before anyone was
hired. Clear the search and the key date.

**Q: My absence was rejected.**
The remaining leave quota is insufficient for the number of days. Check the
**Time & Leave** tab and adjust.

**Q: How do I see a future organizational change?**
Set the **key date** to the future date; assignments that start on or before
that date will be shown.

---

## 10. Troubleshooting

| Symptom | Likely cause | What to do |
|---------|--------------|------------|
| Page doesn't load / blank | Backend or database not running | Ask your admin; verify the API at `/health` |
| "Request failed (0)" or data won't load | API not reachable / proxy off | Confirm the backend runs on port 5000 |
| Dropdowns are empty in a dialog | Customizing/value-help data not loaded | Refresh; verify the database seed data was applied |
| A save shows an error banner | Business rule violation (e.g. quota, missing required field) | Read the message; correct the input and retry |
| Dates look wrong | Key date set to an unexpected value | Clear or reset the key date |

If a problem persists, note the **exact message** and the **personnel number**,
and contact your system administrator.

---

## 11. Glossary (SAP terms)

- **Infotype** — a numbered block of related employee fields (e.g. IT0002 =
  Personal Data).
- **Personnel number (PERNR)** — the unique ID of an employee.
- **Time slice** — one dated version of a record (Valid From–Valid To).
- **Delimit** — to close a record by setting its end date, usually when a new
  slice begins.
- **Time constraint 1** — the rule that exactly one record must exist at all
  times, with no gaps or overlaps (used for Personal Data, Org Assignment).
- **Personnel action** — a guided process that maintains several infotypes at
  once (e.g. Hiring).
- **Organizational unit** — a department or team in the org structure.
- **Position** — a specific post held by one person; based on a **job**.
- **Job** — a generic role template (e.g. "HR Specialist").
- **Chief position** — the managing position of an org unit.
- **Quota** — an entitlement balance, e.g. annual leave days.
- **Key date** — the reference date used to read time-dependent data.

---

*For installation, configuration and technical details, see `README.md` and
`docs/architecture.md`.*
