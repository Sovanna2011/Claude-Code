/* ============================================================================
   HR Module - Self-contained DEMO server
   ----------------------------------------------------------------------------
   Runs the WHOLE system with a single command and zero dependencies:

       node demo/server.js
       → open http://localhost:8080

   It serves, on one origin:
     • the real SAPUI5 web app  (../frontend/webapp, unmodified)
     • a stand-in of the REST API under /api/*

   This lets you try the complete front end + API contract without installing
   .NET or SQL Server. It is NOT the production backend — that is the C# /
   ASP.NET Core + EF Core + SQL Server project under backend/HRModule.Api,
   which implements the identical REST contract against a real database.

   The stand-in reproduces:
     • the seed data from database/06_seed_reference_data.sql
     • the core business logic from the C# services:
         - key-date reads (return the record valid on a date)
         - SAP time-constraint-1 updates (delimit open record + insert new slice)
         - organizational-path evaluation (org tree, chief position, head count)
         - absence quota deduction with insufficient-balance rejection
     • the same JSON/DTO shapes (camelCase) the SAPUI5 app expects

   Data lives in memory and resets every time you restart the server, so you
   can experiment freely.

   NOTE: the web app bootstraps SAPUI5 from the public CDN (ui5.sap.com), so an
   internet connection is required to render the UI. The /api endpoints work
   offline. See demo/README.md for an offline (local OpenUI5) alternative.
   ============================================================================ */
"use strict";
const http = require("http");
const url = require("url");
const fs = require("fs");
const path = require("path");

const WEBAPP = path.join(__dirname, "..", "frontend", "webapp");
const PORT = process.env.PORT || 8080;
const HIGH = "9999-12-31";
const iso = (d) => (d instanceof Date ? d.toISOString().slice(0, 10) : d);
const today = () => iso(new Date());

// ===========================================================================
// Seed data (mirrors database/06_seed_reference_data.sql). Reset on restart.
// ===========================================================================
function seed() {
    return {
        numberRange: { PERNR: 1001 }, // demo employees 1000/1001 exist; next hire -> 1002
        employees: [
            { pernr: 1000, hireDate: "2020-03-01", isActive: true },
            { pernr: 1001, hireDate: "2021-06-15", isActive: true }
        ],
        PA0000: [
            { pernr: 1000, subty: "", begda: "2020-03-01", endda: HIGH, seqnr: 1, massn: "01", massg: "01", stat2: "3" },
            { pernr: 1001, subty: "", begda: "2021-06-15", endda: HIGH, seqnr: 1, massn: "01", massg: "01", stat2: "3" }
        ],
        PA0001: [
            { pernr: 1000, subty: "", begda: "2020-03-01", endda: HIGH, seqnr: 1, bukrs: "1000", werks: "1000", btrtl: "0001", persg: "1", persk: "DU", orgeh: 50000010, plans: 50000100, stell: 50000900, kostl: "HR-1000" },
            { pernr: 1001, subty: "", begda: "2021-06-15", endda: HIGH, seqnr: 1, bukrs: "1000", werks: "1000", btrtl: "0001", persg: "1", persk: "DU", orgeh: 50000010, plans: 50000101, stell: 50000900, kostl: "HR-1000" }
        ],
        PA0002: [
            { pernr: 1000, subty: "", begda: "2020-03-01", endda: HIGH, seqnr: 1, anred: "2", nachn: "Schmidt", vorna: "Andreas", gbdat: "1982-07-12", gesch: "1", natio: "DE", famst: "1" },
            { pernr: 1001, subty: "", begda: "2021-06-15", endda: HIGH, seqnr: 1, anred: "1", nachn: "Nguyen", vorna: "Linda", gbdat: "1990-11-03", gesch: "2", natio: "US", famst: "0" }
        ],
        PA0006: [
            { pernr: 1000, subty: "1", begda: "2020-03-01", endda: HIGH, seqnr: 1, stras: "Hauptstrasse 12", ort01: "Berlin", pstlz: "10115", land1: "DE" },
            { pernr: 1001, subty: "1", begda: "2021-06-15", endda: HIGH, seqnr: 1, stras: "5th Avenue 200", ort01: "New York", pstlz: "10001", land1: "US" }
        ],
        PA0007: [
            { pernr: 1000, subty: "", begda: "2020-03-01", endda: HIGH, seqnr: 1, schkz: "FLEX", empct: 100.0, wostd: 40.0 },
            { pernr: 1001, subty: "", begda: "2021-06-15", endda: HIGH, seqnr: 1, schkz: "FLEX", empct: 100.0, wostd: 40.0 }
        ],
        PA0008: [
            { pernr: 1000, subty: "", begda: "2020-03-01", endda: HIGH, seqnr: 1, trfar: "01", trfgb: "01", trfgr: "E4", bsgrd: 100.0, waers: "EUR", ansal: 96000.0, wageTypes: [{ lgart: "1010", betrg: 8000.0, waers: "EUR" }] },
            { pernr: 1001, subty: "", begda: "2021-06-15", endda: HIGH, seqnr: 1, trfar: "01", trfgb: "01", trfgr: "E2", bsgrd: 100.0, waers: "EUR", ansal: 60000.0, wageTypes: [{ lgart: "1010", betrg: 5000.0, waers: "EUR" }] }
        ],
        PA0009: [
            { pernr: 1000, subty: "0", begda: "2020-03-01", endda: HIGH, seqnr: 1, banks: "DE", bankl: "10070000", bankn: "DE89370400440532013000", zlsch: "U", waers: "EUR" },
            { pernr: 1001, subty: "0", begda: "2021-06-15", endda: HIGH, seqnr: 1, banks: "US", bankl: "021000021", bankn: "US64SVBKUS6S3300958879", zlsch: "U", waers: "EUR" }
        ],
        PA0105: [
            { pernr: 1000, subty: "0010", begda: "2020-03-01", endda: HIGH, seqnr: 1, usrid: "a.schmidt", usrid_long: "a.schmidt@globalcorp.com" },
            { pernr: 1001, subty: "0010", begda: "2021-06-15", endda: HIGH, seqnr: 1, usrid: "l.nguyen", usrid_long: "l.nguyen@globalcorp.com" }
        ],
        PA2001: [
            { pernr: 1000, subty: "0100", begda: "2026-07-01", endda: "2026-07-05", seqnr: 1, awart: "0100", abwtg: 5.0, approved: true }
        ],
        PA2006: [
            { pernr: 1000, subty: "0100", begda: "2026-01-01", endda: "2026-12-31", seqnr: 1, ktart: "0100", anzhl: 30.0, kverb: 5.0 },
            { pernr: 1001, subty: "0100", begda: "2026-01-01", endda: "2026-12-31", seqnr: 1, ktart: "0100", anzhl: 25.0, kverb: 0.0 }
        ],
        HRP1000: [
            { otype: "O", objid: 50000001, begda: "2020-01-01", endda: HIGH, short: "EXEC", stext: "Executive Board" },
            { otype: "O", objid: 50000010, begda: "2020-01-01", endda: HIGH, short: "HR", stext: "Human Resources" },
            { otype: "O", objid: 50000020, begda: "2020-01-01", endda: HIGH, short: "FIN", stext: "Finance" },
            { otype: "C", objid: 50000900, begda: "2020-01-01", endda: HIGH, short: "HRSPEC", stext: "HR Specialist (Job)" },
            { otype: "S", objid: 50000100, begda: "2020-01-01", endda: HIGH, short: "HEADHR", stext: "Head of Human Resources" },
            { otype: "S", objid: 50000101, begda: "2020-01-01", endda: HIGH, short: "HRSPEC1", stext: "HR Specialist" }
        ],
        HRP1001: [
            { otype: "O", objid: 50000010, begda: "2020-01-01", endda: HIGH, rsign: "A", relat: "002", sclas: "O", sobid: "50000001" },
            { otype: "O", objid: 50000020, begda: "2020-01-01", endda: HIGH, rsign: "A", relat: "002", sclas: "O", sobid: "50000001" },
            { otype: "S", objid: 50000100, begda: "2020-01-01", endda: HIGH, rsign: "A", relat: "003", sclas: "O", sobid: "50000010" },
            { otype: "O", objid: 50000010, begda: "2020-01-01", endda: HIGH, rsign: "B", relat: "012", sclas: "S", sobid: "50000100" },
            { otype: "S", objid: 50000101, begda: "2020-01-01", endda: HIGH, rsign: "A", relat: "003", sclas: "O", sobid: "50000010" },
            { otype: "S", objid: 50000101, begda: "2020-01-01", endda: HIGH, rsign: "A", relat: "007", sclas: "C", sobid: "50000900" },
            { otype: "S", objid: 50000101, begda: "2020-01-01", endda: HIGH, rsign: "A", relat: "002", sclas: "S", sobid: "50000100" }
        ],
        T001: [{ bukrs: "1000", butxt: "Global Corp AG", land1: "DE", waers: "EUR" }],
        T500P: [{ werks: "1000", name1: "Head Office" }, { werks: "2000", name1: "Branch Office" }],
        T501: [{ persg: "1", ptext: "Active employees" }, { persg: "2", ptext: "Pensioners" }, { persg: "9", ptext: "External staff" }],
        T503K: [{ persk: "DU", ptext: "Salaried staff" }, { persk: "DW", ptext: "Industrial workers" }, { persk: "DT", ptext: "Trainees" }],
        T512T: { "1010": "Base pay", "1000": "Standard salary", "2000": "Overtime pay", "3000": "Bonus", "5000": "Allowance" },
        T554S: [{ awart: "0100", atext: "Annual leave" }, { awart: "0200", atext: "Sick leave" }, { awart: "0300", atext: "Unpaid leave" }, { awart: "1000", atext: "Overtime" }, { awart: "0400", atext: "Business trip" }],
        DomainValue: {
            GESCH: [{ key: "1", text: "Male" }, { key: "2", text: "Female" }, { key: "3", text: "Undefined" }],
            FAMST: [{ key: "0", text: "Single" }, { key: "1", text: "Married" }, { key: "2", text: "Widowed" }, { key: "3", text: "Divorced" }, { key: "4", text: "Separated" }],
            ANRED: [{ key: "1", text: "Mrs." }, { key: "2", text: "Mr." }, { key: "3", text: "Company" }],
            STAT2: [{ key: "0", text: "Withdrawn" }, { key: "1", text: "Inactive" }, { key: "2", text: "Retiree" }, { key: "3", text: "Active" }],
            USRTY: [{ key: "0010", text: "E-Mail" }, { key: "0020", text: "Telephone" }, { key: "CELL", text: "Mobile phone" }, { key: "MAIL", text: "System user" }]
        }
    };
}
let db = seed();

// ===========================================================================
// Helpers (mirror EmployeeService/OrgService/TimeService)
// ===========================================================================
const validOn = (rows, pernr, key) =>
    rows.filter(r => r.pernr === pernr && r.begda <= key && r.endda >= key)
        .sort((a, b) => (a.begda < b.begda ? 1 : -1))[0] || null;
const orgText = (otype, objid, key) => {
    const o = db.HRP1000.find(x => x.otype === otype && x.objid === objid && x.begda <= key && x.endda >= key);
    return o ? o.stext : null;
};
const domText = (domain, key) => {
    const d = (db.DomainValue[domain] || []).find(x => x.key === key);
    return d ? d.text : null;
};
const dayBefore = (d) => { const dt = new Date(d); dt.setDate(dt.getDate() - 1); return iso(dt); };
const daysBetween = (a, b) => Math.round((new Date(b) - new Date(a)) / 86400000) + 1;

// ===========================================================================
// API handlers
// ===========================================================================
function getEmployees(q) {
    const key = q.keyDate || today();
    let list = db.employees.map(em => {
        const p2 = validOn(db.PA0002, em.pernr, key); if (!p2) return null;
        const p1 = validOn(db.PA0001, em.pernr, key);
        const p0 = validOn(db.PA0000, em.pernr, key);
        const mail = db.PA0105.filter(x => x.pernr === em.pernr && x.subty === "0010" && x.begda <= key && x.endda >= key)[0];
        return {
            pernr: em.pernr, fullName: `${p2.vorna} ${p2.nachn}`,
            orgUnitName: p1 && p1.orgeh ? orgText("O", p1.orgeh, key) : null,
            positionName: p1 && p1.plans ? orgText("S", p1.plans, key) : null,
            email: mail ? (mail.usrid_long || mail.usrid) : null,
            employmentStatus: p0 && p0.stat2 ? domText("STAT2", p0.stat2) : null,
            hireDate: em.hireDate
        };
    }).filter(Boolean);
    if (q.search) {
        const s = q.search.toLowerCase();
        list = list.filter(e => e.fullName.toLowerCase().includes(s) || String(e.pernr).includes(s));
    }
    return list.sort((a, b) => a.pernr - b.pernr);
}

function getEmployee(pernr, q) {
    const key = q.keyDate || today();
    if (!db.employees.find(e => e.pernr === pernr)) return { _status: 404, message: `Employee ${pernr} not found.` };
    const dto = { pernr, keyDate: key };
    const p2 = validOn(db.PA0002, pernr, key);
    if (p2) dto.personalData = {
        formOfAddress: domText("ANRED", p2.anred), lastName: p2.nachn, firstName: p2.vorna, middleName: p2.midnm,
        birthDate: p2.gbdat, genderKey: p2.gesch, gender: domText("GESCH", p2.gesch), nationality: p2.natio,
        maritalStatusKey: p2.famst, maritalStatus: domText("FAMST", p2.famst), begda: p2.begda, endda: p2.endda
    };
    const p1 = validOn(db.PA0001, pernr, key);
    if (p1) {
        const cc = db.T001.find(t => t.bukrs === p1.bukrs); const pa = db.T500P.find(t => t.werks === p1.werks);
        dto.orgAssignment = {
            companyCode: p1.bukrs, companyName: cc ? cc.butxt : null,
            personnelArea: p1.werks, personnelAreaName: pa ? pa.name1 : null, personnelSubarea: p1.btrtl,
            employeeGroup: p1.persg, employeeGroupName: (db.T501.find(t => t.persg === p1.persg) || {}).ptext,
            employeeSubgroup: p1.persk, employeeSubgroupName: (db.T503K.find(t => t.persk === p1.persk) || {}).ptext,
            orgUnitId: p1.orgeh, orgUnitName: p1.orgeh ? orgText("O", p1.orgeh, key) : null,
            positionId: p1.plans, positionName: p1.plans ? orgText("S", p1.plans, key) : null,
            jobId: p1.stell, costCenter: p1.kostl, begda: p1.begda, endda: p1.endda
        };
    }
    const p6 = validOn(db.PA0006, pernr, key);
    if (p6) dto.address = { street: p6.stras, city: p6.ort01, postalCode: p6.pstlz, country: p6.land1, state: p6.state, telephone: p6.telnr };
    const p7 = validOn(db.PA0007, pernr, key);
    if (p7) dto.workingTime = { workScheduleRule: p7.schkz, employmentPercent: p7.empct, weeklyHours: p7.wostd };
    const p8 = validOn(db.PA0008, pernr, key);
    if (p8) dto.basicPay = {
        payScaleType: p8.trfar, payScaleArea: p8.trfgb, payScaleGroup: p8.trfgr, capacityUtilization: p8.bsgrd,
        currency: p8.waers, annualSalary: p8.ansal,
        wageTypes: (p8.wageTypes || []).map(w => ({ wageType: w.lgart, wageTypeText: db.T512T[w.lgart], amount: w.betrg, currency: w.waers }))
    };
    dto.communications = db.PA0105.filter(x => x.pernr === pernr && x.begda <= key && x.endda >= key)
        .map(c => ({ typeKey: c.subty, typeText: domText("USRTY", c.subty), id: c.usrid, longId: c.usrid_long }));
    dto.bankDetails = db.PA0009.filter(x => x.pernr === pernr && x.begda <= key && x.endda >= key)
        .map(b => ({ bankDetailsType: b.subty, bankCountry: b.banks, bankKey: b.bankl, accountNumber: b.bankn, currency: b.waers }));
    return dto;
}

function hire(body) {
    const pernr = ++db.numberRange.PERNR;
    db.employees.push({ pernr, hireDate: body.hireDate, isActive: true });
    db.PA0000.push({ pernr, subty: "", begda: body.hireDate, endda: HIGH, seqnr: 1, massn: "01", massg: "01", stat2: "3" });
    db.PA0001.push({ pernr, subty: "", begda: body.hireDate, endda: HIGH, seqnr: 1, bukrs: body.companyCode, werks: body.personnelArea, persg: body.employeeGroup, persk: body.employeeSubgroup, orgeh: body.orgUnit || null, plans: body.position || null });
    db.PA0002.push({ pernr, subty: "", begda: body.hireDate, endda: HIGH, seqnr: 1, nachn: body.lastName, vorna: body.firstName, gbdat: body.birthDate || null, gesch: body.gender });
    if (body.email) db.PA0105.push({ pernr, subty: "0010", begda: body.hireDate, endda: HIGH, seqnr: 1, usrid: body.email, usrid_long: body.email });
    return { pernr, message: `Employee ${pernr} hired successfully.` };
}

function updatePersonal(pernr, body) {
    if (!db.employees.find(e => e.pernr === pernr)) return { _status: 404, message: `Employee ${pernr} not found.` };
    const cur = db.PA0002.filter(x => x.pernr === pernr && x.endda >= body.begda && x.begda < body.begda).sort((a, b) => a.begda < b.begda ? 1 : -1)[0];
    if (cur) cur.endda = dayBefore(body.begda);
    db.PA0002.push({ pernr, subty: "", begda: body.begda, endda: HIGH, seqnr: 1, anred: body.formOfAddress, nachn: body.lastName, vorna: body.firstName, midnm: body.middleName, gbdat: body.birthDate || null, gesch: body.gender, natio: body.nationality, famst: body.maritalStatus });
    return { _status: 204 };
}

function reassign(pernr, body) {
    if (!db.employees.find(e => e.pernr === pernr)) return { _status: 404, message: `Employee ${pernr} not found.` };
    const cur = db.PA0001.filter(x => x.pernr === pernr && x.endda >= body.begda && x.begda < body.begda).sort((a, b) => a.begda < b.begda ? 1 : -1)[0];
    const rec = {
        pernr, subty: "", begda: body.begda, endda: HIGH, seqnr: 1,
        bukrs: cur && cur.bukrs, werks: cur && cur.werks, btrtl: cur && cur.btrtl, persg: cur && cur.persg, persk: cur && cur.persk,
        orgeh: body.orgUnit || (cur && cur.orgeh), plans: body.position || (cur && cur.plans),
        stell: cur && cur.stell, kostl: body.costCenter || (cur && cur.kostl)
    };
    if (cur) cur.endda = dayBefore(body.begda);
    db.PA0001.push(rec);
    return { _status: 204 };
}

function leaveBalances(pernr) {
    return db.PA2006.filter(q => q.pernr === pernr).map(q => ({
        pernr, quotaType: q.ktart, quotaText: (db.T554S.find(t => t.awart === q.ktart) || {}).atext,
        begda: q.begda, endda: q.endda, entitlement: q.anzhl, deducted: q.kverb, remaining: q.anzhl - q.kverb
    }));
}

function recordAbsence(pernr, body) {
    if (!db.employees.find(e => e.pernr === pernr)) return { _status: 404, message: `Employee ${pernr} not found.` };
    if (body.endda < body.begda) return { _status: 400, message: "End date must not be before start date." };
    const days = body.days != null ? body.days : daysBetween(body.begda, body.endda);
    const quota = db.PA2006.filter(q => q.pernr === pernr && q.ktart === body.absenceType && q.begda <= body.begda && q.endda >= body.begda).sort((a, b) => a.begda < b.begda ? 1 : -1)[0];
    if (quota) {
        if (quota.anzhl - quota.kverb < days) return { _status: 400, message: "Insufficient leave quota for this absence." };
        quota.kverb += days;
    }
    db.PA2001.push({ pernr, subty: body.absenceType, awart: body.absenceType, begda: body.begda, endda: body.endda, seqnr: 1, abwtg: days, approved: false });
    return { _status: 204 };
}

function orgUnitsFlat(key) {
    return db.HRP1000.filter(o => o.otype === "O" && o.begda <= key && o.endda >= key).map(o => {
        const rel = db.HRP1001.find(r => r.otype === "O" && r.sclas === "O" && r.rsign === "A" && r.relat === "002" && r.objid === o.objid && r.begda <= key && r.endda >= key);
        return { orgUnitId: o.objid, orgUnitName: o.stext, shortText: o.short, parentOrgId: rel ? parseInt(rel.sobid, 10) : null, depth: 0, children: [] };
    }).sort((a, b) => a.orgUnitId - b.orgUnitId);
}

function orgStructure(root, key) {
    const flat = orgUnitsFlat(key); const byId = {}; flat.forEach(u => byId[u.orgUnitId] = u);
    if (!byId[root]) return { _status: 404, message: `Org unit ${root} not found.` };
    db.PA0001.filter(a => a.orgeh && a.begda <= key && a.endda >= key).forEach(a => { if (byId[a.orgeh]) byId[a.orgeh].headCount = (byId[a.orgeh].headCount || 0) + 1; });
    db.HRP1001.filter(r => r.otype === "O" && r.rsign === "B" && r.relat === "012" && r.sclas === "S" && r.begda <= key && r.endda >= key).forEach(r => { if (byId[r.objid]) byId[r.objid].managerPositionName = orgText("S", parseInt(r.sobid, 10), key); });
    flat.forEach(u => { if (u.parentOrgId != null && byId[u.parentOrgId] && u.parentOrgId !== u.orgUnitId) byId[u.parentOrgId].children.push(u); });
    return byId[root];
}

function positions(orgUnitId, key) {
    let list = db.HRP1000.filter(p => p.otype === "S" && p.begda <= key && p.endda >= key).map(p => {
        const toOrg = db.HRP1001.find(r => r.otype === "S" && r.rsign === "A" && r.relat === "003" && r.sclas === "O" && r.objid === p.objid && r.begda <= key && r.endda >= key);
        const toJob = db.HRP1001.find(r => r.otype === "S" && r.rsign === "A" && r.relat === "007" && r.sclas === "C" && r.objid === p.objid && r.begda <= key && r.endda >= key);
        const holder = db.PA0001.filter(a => a.plans === p.objid && a.begda <= key && a.endda >= key).map(a => { const pd = validOn(db.PA0002, a.pernr, key); return pd ? { pernr: a.pernr, name: `${pd.vorna} ${pd.nachn}` } : null; }).filter(Boolean)[0];
        const orgId = toOrg ? parseInt(toOrg.sobid, 10) : null; const jobId = toJob ? parseInt(toJob.sobid, 10) : null;
        return {
            positionId: p.objid, positionName: p.stext,
            orgUnitId: orgId, orgUnitName: orgId ? orgText("O", orgId, key) : null,
            jobId, jobName: jobId ? orgText("C", jobId, key) : null,
            holderPernr: holder ? holder.pernr : null, holderName: holder ? holder.name : null, isVacant: !holder
        };
    });
    if (orgUnitId) list = list.filter(p => p.orgUnitId === orgUnitId);
    return list.sort((a, b) => a.positionId - b.positionId);
}

const vh = {
    "company-codes": () => db.T001.map(x => ({ key: x.bukrs, text: x.butxt })),
    "personnel-areas": () => db.T500P.map(x => ({ key: x.werks, text: x.name1 })),
    "employee-groups": () => db.T501.map(x => ({ key: x.persg, text: x.ptext })),
    "employee-subgroups": () => db.T503K.map(x => ({ key: x.persk, text: x.ptext })),
    "absence-types": () => db.T554S.map(x => ({ key: x.awart, text: x.atext }))
};

// ===========================================================================
// Routing
// ===========================================================================
function sendJson(res, status, obj) {
    res.writeHead(status, { "Content-Type": "application/json", "Access-Control-Allow-Origin": "*" });
    res.end(obj === undefined ? "" : JSON.stringify(obj));
}
function result(res, r) {
    if (r && r._status) { const { _status, ...rest } = r; return sendJson(res, _status, _status === 204 ? undefined : rest); }
    return sendJson(res, 200, r);
}

function handleApi(req, res, path, q, json) {
    const m = req.method; let mm;
    try {
        if (path === "/api/employees" && m === "GET") return result(res, getEmployees(q));
        if (path === "/api/employees/hire" && m === "POST") return result(res, hire(json));
        if ((mm = path.match(/^\/api\/employees\/(\d+)$/)) && m === "GET") return result(res, getEmployee(+mm[1], q));
        if ((mm = path.match(/^\/api\/employees\/(\d+)\/personaldata$/)) && m === "PUT") return result(res, updatePersonal(+mm[1], json));
        if ((mm = path.match(/^\/api\/employees\/(\d+)\/reassign$/)) && m === "PUT") return result(res, reassign(+mm[1], json));
        if ((mm = path.match(/^\/api\/employees\/(\d+)\/leave-balances$/)) && m === "GET") return result(res, leaveBalances(+mm[1]));
        if ((mm = path.match(/^\/api\/employees\/(\d+)\/absences$/)) && m === "POST") return result(res, recordAbsence(+mm[1], json));
        if (path === "/api/orgunits" && m === "GET") return result(res, orgUnitsFlat(q.keyDate || today()));
        if ((mm = path.match(/^\/api\/orgunits\/(\d+)\/structure$/)) && m === "GET") return result(res, orgStructure(+mm[1], q.keyDate || today()));
        if (path === "/api/orgunits/positions" && m === "GET") return result(res, positions(q.orgUnitId ? +q.orgUnitId : null, q.keyDate || today()));
        if ((mm = path.match(/^\/api\/valuehelp\/([a-z-]+)$/)) && m === "GET" && vh[mm[1]]) return result(res, vh[mm[1]]());
        if ((mm = path.match(/^\/api\/valuehelp\/domain\/(\w+)$/)) && m === "GET") return result(res, db.DomainValue[mm[1]] || []);
        if (path === "/api/reset" && m === "POST") { db = seed(); return sendJson(res, 200, { message: "Demo data reset." }); }
        return sendJson(res, 404, { message: "Unknown API route: " + path });
    } catch (e) {
        return sendJson(res, 400, { message: e.message });
    }
}

const MIME = {
    ".html": "text/html", ".js": "application/javascript", ".json": "application/json",
    ".css": "text/css", ".properties": "text/plain; charset=utf-8", ".png": "image/png",
    ".svg": "image/svg+xml", ".ico": "image/x-icon", ".map": "application/json"
};
function serveStatic(res, pathname) {
    let rel = pathname === "/" ? "/index.html" : pathname;
    // Prevent path traversal.
    const target = path.normalize(path.join(WEBAPP, rel));
    if (!target.startsWith(WEBAPP)) { res.writeHead(403); return res.end("Forbidden"); }
    fs.readFile(target, (err, data) => {
        if (err) { res.writeHead(404, { "Content-Type": "text/plain" }); return res.end("Not found: " + rel); }
        res.writeHead(200, { "Content-Type": MIME[path.extname(target)] || "application/octet-stream" });
        res.end(data);
    });
}

const server = http.createServer((req, res) => {
    const u = url.parse(req.url, true);
    const pathname = u.pathname;
    if (req.method === "OPTIONS") {
        res.writeHead(204, { "Access-Control-Allow-Origin": "*", "Access-Control-Allow-Methods": "GET,POST,PUT,OPTIONS", "Access-Control-Allow-Headers": "Content-Type" });
        return res.end();
    }
    if (pathname === "/health") return sendJson(res, 200, { status: "UP", module: "HCM (demo)" });
    if (pathname.startsWith("/api/")) {
        let body = "";
        req.on("data", c => body += c);
        req.on("end", () => { let json = {}; try { json = body ? JSON.parse(body) : {}; } catch (e) { return sendJson(res, 400, { message: "Invalid JSON body." }); } handleApi(req, res, pathname.replace(/\/$/, ""), u.query, json); });
        return;
    }
    serveStatic(res, pathname);
});

server.listen(PORT, () => {
    console.log("========================================================");
    console.log("  HR Module - full demo system");
    console.log("  Web app + API:  http://localhost:" + PORT);
    console.log("  API health:     http://localhost:" + PORT + "/health");
    console.log("  Reset data:     POST http://localhost:" + PORT + "/api/reset");
    console.log("  (UI needs internet for the SAPUI5 CDN; API works offline)");
    console.log("========================================================");
});
