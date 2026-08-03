/* ============================================================================
   PO Approval Module - Seed reference & demo data
   Reference: SAP ECC 6.0 EHP8 conventions.

   Populates customizing, the release strategy (group 01, three strategies by
   net value), vendors, four demo purchase orders in different release states,
   and four application users. Idempotent (MERGE / existence checks).

   Demo login password for every seeded user is:  Welcome1
   ============================================================================ */

USE [POApproval];
GO

/* --- Number range: external PO numbers (SAP 45* range) -------------------- */
MERGE PO.NumberRange AS t
USING (VALUES ('EBELN', 4500000000, 4599999999, 4500000010))
    AS s(RangeObject, FromNumber, ToNumber, CurrentNumber)
ON t.RangeObject = s.RangeObject
WHEN NOT MATCHED THEN
    INSERT (RangeObject, FromNumber, ToNumber, CurrentNumber)
    VALUES (s.RangeObject, s.FromNumber, s.ToNumber, s.CurrentNumber);
GO

/* --- Domain fixed values -------------------------------------------------- */
MERGE PO.DomainValue AS t
USING (VALUES
    ('FRGKE',' ','Not subject to release'),
    ('FRGKE','B','Blocked - release pending'),
    ('FRGKE','R','Released'),
    ('BSTYP','F','Purchase order'),
    ('BSTYP','A','Request for quotation'),
    ('BSTYP','K','Contract'),
    ('ACTION','RELEASE','Release effected'),
    ('ACTION','REJECT','Release rejected'),
    ('ACTION','RESET','Release reset')
) AS s(Domain,ValueKey,ValueTxt)
ON t.Domain=s.Domain AND t.ValueKey=s.ValueKey
WHEN NOT MATCHED THEN INSERT(Domain,ValueKey,ValueTxt) VALUES(s.Domain,s.ValueKey,s.ValueTxt);
GO

/* --- Company codes / purchasing organization & groups --------------------- */
MERGE PO.T001 AS t
USING (VALUES ('1000','Global Corp AG','DE','EUR')) AS s(BUKRS,BUTXT,LAND1,WAERS)
ON t.BUKRS=s.BUKRS
WHEN NOT MATCHED THEN INSERT(BUKRS,BUTXT,LAND1,WAERS) VALUES(s.BUKRS,s.BUTXT,s.LAND1,s.WAERS);

MERGE PO.T024E AS t
USING (VALUES ('1000','Central Purchasing','1000')) AS s(EKORG,EKOTX,BUKRS)
ON t.EKORG=s.EKORG
WHEN NOT MATCHED THEN INSERT(EKORG,EKOTX,BUKRS) VALUES(s.EKORG,s.EKOTX,s.BUKRS);

MERGE PO.T024 AS t
USING (VALUES ('001','IT Procurement','+49 30 111'),
              ('002','Facilities','+49 30 222'),
              ('003','Indirect Spend','+49 30 333'))
    AS s(EKGRP,EKNAM,EKTEL) ON t.EKGRP=s.EKGRP
WHEN NOT MATCHED THEN INSERT(EKGRP,EKNAM,EKTEL) VALUES(s.EKGRP,s.EKNAM,s.EKTEL);

MERGE PO.T161 AS t
USING (VALUES ('F','NB','Standard PO'),('F','FO','Framework order'),
              ('F','UB','Stock transport order'))
    AS s(BSTYP,BSART,BATXT) ON t.BSTYP=s.BSTYP AND t.BSART=s.BSART
WHEN NOT MATCHED THEN INSERT(BSTYP,BSART,BATXT) VALUES(s.BSTYP,s.BSART,s.BATXT);
GO

/* ============================================================================
   Release strategy - group 01, sign-off by net order value.
     Codes : 01 Department Manager, 02 Finance Controller, 03 CFO
     S1 :        0 .. 4,999.99  -> [01]
     S2 :    5,000 .. 24,999.99 -> [01, 02]
     S3 :   25,000 .. and above -> [01, 02, 03]
   ============================================================================ */
MERGE PO.T16FG AS t
USING (VALUES ('01','Purchase order approval','FRG_EKKO'))
    AS s(FRGGR,FRGGT,CLASS) ON t.FRGGR=s.FRGGR
WHEN NOT MATCHED THEN INSERT(FRGGR,FRGGT,CLASS) VALUES(s.FRGGR,s.FRGGT,s.CLASS);
GO

MERGE PO.T16FC AS t
USING (VALUES ('01','01','Department Manager'),
              ('01','02','Finance Controller'),
              ('01','03','Chief Financial Officer'))
    AS s(FRGGR,FRGCO,FRGCT) ON t.FRGGR=s.FRGGR AND t.FRGCO=s.FRGCO
WHEN NOT MATCHED THEN INSERT(FRGGR,FRGCO,FRGCT) VALUES(s.FRGGR,s.FRGCO,s.FRGCT);
GO

MERGE PO.T16FS AS t
USING (VALUES
    ('01','S1','Up to EUR 5,000',        'EUR',0.00,        4999.99),
    ('01','S2','EUR 5,000 - 25,000',     'EUR',5000.00,     24999.99),
    ('01','S3','EUR 25,000 and above',   'EUR',25000.00,    999999999999.99)
) AS s(FRGGR,FRGSX,FRGST,WAERS,ValFrom,ValTo)
ON t.FRGGR=s.FRGGR AND t.FRGSX=s.FRGSX
WHEN NOT MATCHED THEN INSERT(FRGGR,FRGSX,FRGST,WAERS,ValFrom,ValTo)
    VALUES(s.FRGGR,s.FRGSX,s.FRGST,s.WAERS,s.ValFrom,s.ValTo);
GO

MERGE PO.T16FS_Code AS t
USING (VALUES
    ('01','S1',1,'01'),
    ('01','S2',1,'01'),('01','S2',2,'02'),
    ('01','S3',1,'01'),('01','S3',2,'02'),('01','S3',3,'03')
) AS s(FRGGR,FRGSX,StepNo,FRGCO)
ON t.FRGGR=s.FRGGR AND t.FRGSX=s.FRGSX AND t.StepNo=s.StepNo
WHEN NOT MATCHED THEN INSERT(FRGGR,FRGSX,StepNo,FRGCO)
    VALUES(s.FRGGR,s.FRGSX,s.StepNo,s.FRGCO);
GO

/* --- Vendors -------------------------------------------------------------- */
MERGE PO.LFA1 AS t
USING (VALUES
    ('100000','Dell Technologies GmbH','Frankfurt','60313','DE','DE811234567','EUR'),
    ('100001','Siemens AG','Munich','80333','DE','DE129274202','EUR'),
    ('100002','SAP SE','Walldorf','69190','DE','DE143454214','EUR'),
    ('100003','Staples Deutschland GmbH','Hamburg','22453','DE','DE118650123','EUR')
) AS s(LIFNR,NAME1,ORT01,PSTLZ,LAND1,STCEG,WAERS)
ON t.LIFNR=s.LIFNR
WHEN NOT MATCHED THEN INSERT(LIFNR,NAME1,ORT01,PSTLZ,LAND1,STCEG,WAERS)
    VALUES(s.LIFNR,s.NAME1,s.ORT01,s.PSTLZ,s.LAND1,s.STCEG,s.WAERS);
GO

/* ============================================================================
   Demo purchase orders in different release states.
     4500000001  EUR  3,200  S1 [01]        blocked (pending 01)
     4500000002  EUR 12,500  S2 [01,02]     01 released, pending 02
     4500000003  EUR 48,000  S3 [01,02,03]  blocked (pending 01)
     4500000004  EUR    850  S1 [01]        fully released
   ============================================================================ */
IF NOT EXISTS (SELECT 1 FROM PO.EKKO WHERE EBELN IN
    ('4500000001','4500000002','4500000003','4500000004'))
BEGIN
    INSERT INTO PO.EKKO (EBELN,BSTYP,BSART,LIFNR,EKORG,EKGRP,BUKRS,WAERS,BEDAT,RLWRT,FRGGR,FRGSX,FRGZU,FRGKE,FRGRL,ERNAM,AEDAT)
    VALUES
      ('4500000001','F','NB','100000','1000','001','1000','EUR','2026-07-20', 3200.00,'01','S1','',  'B','X','rbuyer','2026-07-20'),
      ('4500000002','F','NB','100001','1000','002','1000','EUR','2026-07-22',12500.00,'01','S2','01','B','X','rbuyer','2026-07-22'),
      ('4500000003','F','NB','100002','1000','001','1000','EUR','2026-07-25',48000.00,'01','S3','',  'B','X','rbuyer','2026-07-25'),
      ('4500000004','F','NB','100003','1000','003','1000','EUR','2026-07-28',  850.00,'01','S1','01','R',' ','rbuyer','2026-07-28');

    INSERT INTO PO.EKPO (EBELN,EBELP,TXZ01,MATNR,MATKL,WERKS,MENGE,MEINS,NETPR,PEINH,NETWR,LOEKZ)
    VALUES
      ('4500000001',10,'Notebook Latitude 5540','MAT-NB-5540','0010','1000', 8.000,'EA', 400.00,1, 3200.00,' '),
      ('4500000002',10,'Rack server RX2540',    'MAT-SRV-2540','0011','1000', 5.000,'EA',2500.00,1,12500.00,' '),
      ('4500000003',10,'SAP S/4HANA user license','MAT-LIC-S4','0090','1000', 1.000,'EA',48000.00,1,48000.00,' '),
      ('4500000004',10,'Copy paper A4 (box)',    'MAT-OFF-A4','0080','1000',10.000,'EA',  85.00,1,  850.00,' ');

    /* Audit log for releases that have already occurred */
    INSERT INTO PO.ReleaseLog (EBELN,FRGCO,ActionTyp,UNAME,ActedOn,Note)
    VALUES
      ('4500000002','01','RELEASE','jdoe','2026-07-22T10:15:00','Approved by department'),
      ('4500000004','01','RELEASE','jdoe','2026-07-28T09:05:00','Approved - routine supplies');
END
GO

/* ============================================================================
   Application users (password = "Welcome1" for all; PBKDF2-HMAC-SHA256).
     jdoe   Department Manager  -> release code 01
     msmith Finance Controller  -> release code 02
     klee   Chief Financial Off -> release code 03
     rbuyer Purchaser           -> no release code (view / create only)
   ============================================================================ */
MERGE PO.AppUser AS t
USING (VALUES
    ('jdoe','John Doe','john.doe@globalcorp.com',
     '120000.c69yMo9qpJu13zl8lCS52A==.IsJTNZXXktzBIDWzLbZzRQKBLul2olyJfJidPWGTOy4='),
    ('msmith','Mary Smith','mary.smith@globalcorp.com',
     '120000.J2c6jU9YykS5rfdBO0eIPw==.NZ5f5lJwMTNq5UY3a8DCurRwvUUlB8ZJ3d2KQc/fjck='),
    ('klee','Karen Lee','karen.lee@globalcorp.com',
     '120000.iP7SdqZiuQoYvO66JylQkQ==.d98z7taM9CkyYTYcPLwhd2e9kId1VFz4zw6sFPNaU+8='),
    ('rbuyer','Robert Buyer','robert.buyer@globalcorp.com',
     '120000.IgpccQPWXd4d0Vpx7mx+dA==.f1cQhkOb3TJQdEJ2Bhg+oYm1yOR/PjM6FqayMpa1f+8=')
) AS s(UserName,DisplayName,Email,PwdHash)
ON t.UserName=s.UserName
WHEN NOT MATCHED THEN INSERT(UserName,DisplayName,Email,PwdHash)
    VALUES(s.UserName,s.DisplayName,s.Email,s.PwdHash);
GO

/* Release-code assignments (auth object M_EINK_FRG). */
MERGE PO.AppUserReleaseCode AS t
USING (
    SELECT u.UserId, x.FRGGR, x.FRGCO
    FROM (VALUES
        ('jdoe','01','01'),
        ('msmith','01','02'),
        ('klee','01','03')
    ) AS x(UserName,FRGGR,FRGCO)
    JOIN PO.AppUser u ON u.UserName = x.UserName
) AS s(UserId,FRGGR,FRGCO)
ON t.UserId=s.UserId AND t.FRGGR=s.FRGGR AND t.FRGCO=s.FRGCO
WHEN NOT MATCHED THEN INSERT(UserId,FRGGR,FRGCO) VALUES(s.UserId,s.FRGGR,s.FRGCO);
GO

PRINT 'Seed reference and demo data loaded.';
GO
