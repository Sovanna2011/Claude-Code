/* ============================================================================
   HR Module - Seed Reference & Demo Data
   Reference: SAP ECC 6.0 EHP8 conventions

   Populates customizing tables, number ranges, a small organizational
   structure and two demo employees so the module is runnable end to end.
   Idempotent: uses MERGE / existence checks.
   ============================================================================ */

USE [HRModule];
GO

/* --- Number ranges -------------------------------------------------------- */
MERGE HR.NumberRange AS t
USING (VALUES
    ('PERNR', 1, 99999999, 1000),
    ('OBJID', 50000000, 59999999, 50000000)
) AS s(RangeObject, FromNumber, ToNumber, CurrentNumber)
ON t.RangeObject = s.RangeObject
WHEN NOT MATCHED THEN
    INSERT (RangeObject, FromNumber, ToNumber, CurrentNumber)
    VALUES (s.RangeObject, s.FromNumber, s.ToNumber, s.CurrentNumber);
GO

/* --- Countries ------------------------------------------------------------ */
MERGE HR.T005 AS t
USING (VALUES
    ('DE','Germany','EUR'),('US','United States','USD'),
    ('FR','France','EUR'),('GB','United Kingdom','GBP'),
    ('KH','Cambodia','KHR'),('SG','Singapore','SGD')
) AS s(LAND1,LANDX,WAERS) ON t.LAND1=s.LAND1
WHEN NOT MATCHED THEN INSERT(LAND1,LANDX,WAERS) VALUES(s.LAND1,s.LANDX,s.WAERS);
GO

/* --- Company code / personnel structure ----------------------------------- */
MERGE HR.T001 AS t
USING (VALUES ('1000','Global Corp AG','DE','EUR')) AS s(BUKRS,BUTXT,LAND1,WAERS)
ON t.BUKRS=s.BUKRS
WHEN NOT MATCHED THEN INSERT(BUKRS,BUTXT,LAND1,WAERS) VALUES(s.BUKRS,s.BUTXT,s.LAND1,s.WAERS);

MERGE HR.T500P AS t
USING (VALUES ('1000','Head Office','1000','01'),('2000','Branch Office','1000','01'))
    AS s(WERKS,NAME1,BUKRS,MOLGA) ON t.WERKS=s.WERKS
WHEN NOT MATCHED THEN INSERT(WERKS,NAME1,BUKRS,MOLGA) VALUES(s.WERKS,s.NAME1,s.BUKRS,s.MOLGA);

MERGE HR.T001P AS t
USING (VALUES ('1000','0001','Administration'),('1000','0002','Production'),
              ('2000','0001','Sales')) AS s(WERKS,BTRTL,BTEXT)
ON t.WERKS=s.WERKS AND t.BTRTL=s.BTRTL
WHEN NOT MATCHED THEN INSERT(WERKS,BTRTL,BTEXT) VALUES(s.WERKS,s.BTRTL,s.BTEXT);

MERGE HR.T501 AS t
USING (VALUES ('1','Active employees'),('2','Pensioners'),('9','External staff'))
    AS s(PERSG,PTEXT) ON t.PERSG=s.PERSG
WHEN NOT MATCHED THEN INSERT(PERSG,PTEXT) VALUES(s.PERSG,s.PTEXT);

MERGE HR.T503K AS t
USING (VALUES ('DU','Salaried staff'),('DW','Industrial workers'),
              ('DT','Trainees')) AS s(PERSK,PTEXT) ON t.PERSK=s.PERSK
WHEN NOT MATCHED THEN INSERT(PERSK,PTEXT) VALUES(s.PERSK,s.PTEXT);
GO

/* --- Action types & reasons ----------------------------------------------- */
MERGE HR.T529A AS t
USING (VALUES ('01','Hiring'),('02','Organizational reassignment'),
              ('03','Pay increase'),('10','Leaving'),('11','Retirement'))
    AS s(MASSN,MNTXT) ON t.MASSN=s.MASSN
WHEN NOT MATCHED THEN INSERT(MASSN,MNTXT) VALUES(s.MASSN,s.MNTXT);

MERGE HR.T530 AS t
USING (VALUES ('01','01','New hire'),('10','01','Resignation'),
              ('10','02','Dismissal'),('03','01','Annual review'))
    AS s(MASSN,MASSG,MGTXT) ON t.MASSN=s.MASSN AND t.MASSG=s.MASSG
WHEN NOT MATCHED THEN INSERT(MASSN,MASSG,MGTXT) VALUES(s.MASSN,s.MASSG,s.MGTXT);
GO

/* --- Absence / attendance types ------------------------------------------- */
MERGE HR.T554S AS t
USING (VALUES ('01','0100','Annual leave','A'),('01','0200','Sick leave','A'),
              ('01','0300','Unpaid leave','A'),('01','1000','Overtime','P'),
              ('01','0400','Business trip','P'))
    AS s(MOABW,AWART,ATEXT,KENNZ) ON t.MOABW=s.MOABW AND t.AWART=s.AWART
WHEN NOT MATCHED THEN INSERT(MOABW,AWART,ATEXT,KENNZ)
    VALUES(s.MOABW,s.AWART,s.ATEXT,s.KENNZ);
GO

/* --- Wage type texts ------------------------------------------------------ */
MERGE HR.T512T AS t
USING (VALUES ('1000','Standard salary'),('1010','Base pay'),
              ('2000','Overtime pay'),('3000','Bonus'),('5000','Allowance'))
    AS s(LGART,LGTXT) ON t.LGART=s.LGART
WHEN NOT MATCHED THEN INSERT(LGART,LGTXT) VALUES(s.LGART,s.LGTXT);
GO

/* --- Domain fixed values -------------------------------------------------- */
MERGE HR.DomainValue AS t
USING (VALUES
    ('GESCH','1','Male'),('GESCH','2','Female'),('GESCH','3','Undefined'),
    ('FAMST','0','Single'),('FAMST','1','Married'),('FAMST','2','Widowed'),
    ('FAMST','3','Divorced'),('FAMST','4','Separated'),
    ('ANRED','1','Mrs.'),('ANRED','2','Mr.'),('ANRED','3','Company'),
    ('STAT2','0','Withdrawn'),('STAT2','1','Inactive'),
    ('STAT2','2','Retiree'),('STAT2','3','Active'),
    ('USRTY','0010','E-Mail'),('USRTY','0020','Telephone'),
    ('USRTY','CELL','Mobile phone'),('USRTY','MAIL','System user')
) AS s(Domain,ValueKey,ValueTxt)
ON t.Domain=s.Domain AND t.ValueKey=s.ValueKey
WHEN NOT MATCHED THEN INSERT(Domain,ValueKey,ValueTxt)
    VALUES(s.Domain,s.ValueKey,s.ValueTxt);
GO

/* ============================================================================
   Organizational structure (OM)
   Org units: 50000001 Executive Board -> 50000010 Human Resources,
                                          50000020 Finance
   Job:       50000900 HR Specialist
   Positions: 50000100 Head of HR (chief), 50000101 HR Specialist
   ============================================================================ */
MERGE HR.HRP1000 AS t
USING (VALUES
    ('O',50000001,'EXEC','Executive Board'),
    ('O',50000010,'HR','Human Resources'),
    ('O',50000020,'FIN','Finance'),
    ('C',50000900,'HRSPEC','HR Specialist (Job)'),
    ('S',50000100,'HEADHR','Head of Human Resources'),
    ('S',50000101,'HRSPEC1','HR Specialist')
) AS s(OTYPE,OBJID,SHORT,STEXT)
ON t.PLVAR='01' AND t.OTYPE=s.OTYPE AND t.OBJID=s.OBJID AND t.ENDDA='9999-12-31'
WHEN NOT MATCHED THEN
    INSERT (OTYPE,OBJID,BEGDA,ENDDA,SHORT,STEXT)
    VALUES (s.OTYPE,s.OBJID,'2020-01-01','9999-12-31',s.SHORT,s.STEXT);
GO

/* Relationships:
   HR (O) belongs to Executive Board (A 002 org->org "reports to")
   Head of HR position belongs to HR org unit (A 003 S->O)
   HR org unit is managed by Head of HR (B 012 O->S)
   HR Specialist position belongs to HR (A 003), described by job (A 007),
   and reports to Head of HR (A 002 S->S). */
MERGE HR.HRP1001 AS t
USING (VALUES
    ('O',50000010,'A','002','O','50000001'),   -- HR reports to Exec Board
    ('O',50000020,'A','002','O','50000001'),   -- Finance reports to Exec Board
    ('S',50000100,'A','003','O','50000010'),   -- Head of HR -> HR org unit
    ('O',50000010,'B','012','S','50000100'),   -- HR managed by Head of HR
    ('S',50000101,'A','003','O','50000010'),   -- HR Specialist -> HR org unit
    ('S',50000101,'A','007','C','50000900'),   -- described by job HR Specialist
    ('S',50000101,'A','002','S','50000100')    -- reports to Head of HR
) AS s(OTYPE,OBJID,RSIGN,RELAT,SCLAS,SOBID)
ON t.PLVAR='01' AND t.OTYPE=s.OTYPE AND t.OBJID=s.OBJID
   AND t.RSIGN=s.RSIGN AND t.RELAT=s.RELAT AND t.SCLAS=s.SCLAS
   AND t.SOBID=s.SOBID AND t.ENDDA='9999-12-31'
WHEN NOT MATCHED THEN
    INSERT (OTYPE,OBJID,BEGDA,ENDDA,RSIGN,RELAT,SCLAS,SOBID)
    VALUES (s.OTYPE,s.OBJID,'2020-01-01','9999-12-31',
            s.RSIGN,s.RELAT,s.SCLAS,s.SOBID);
GO

MERGE HR.T528T AS t
USING (VALUES (50000100,'Head of Human Resources'),(50000101,'HR Specialist'))
    AS s(PLANS,PLSTX) ON t.PLANS=s.PLANS
WHEN NOT MATCHED THEN INSERT(PLANS,PLSTX) VALUES(s.PLANS,s.PLSTX);
GO

/* ============================================================================
   Demo employees 1000 (Head of HR) and 1001 (HR Specialist)
   ============================================================================ */
IF NOT EXISTS (SELECT 1 FROM HR.EmployeeMaster WHERE PERNR IN (1000,1001))
BEGIN
    INSERT INTO HR.EmployeeMaster (PERNR, HireDate, IsActive)
    VALUES (1000,'2020-03-01',1),(1001,'2021-06-15',1);

    /* PA0000 Actions */
    INSERT INTO HR.PA0000 (PERNR,BEGDA,ENDDA,MASSN,MASSG,STAT2,AEDTM,UNAME)
    VALUES (1000,'2020-03-01','9999-12-31','01','01','3','2020-03-01','ADMIN'),
           (1001,'2021-06-15','9999-12-31','01','01','3','2021-06-15','ADMIN');

    /* PA0001 Organizational Assignment */
    INSERT INTO HR.PA0001 (PERNR,BEGDA,ENDDA,BUKRS,WERKS,BTRTL,PERSG,PERSK,ORGEH,PLANS,STELL,KOSTL,AEDTM,UNAME)
    VALUES (1000,'2020-03-01','9999-12-31','1000','1000','0001','1','DU',50000010,50000100,50000900,'HR-1000','2020-03-01','ADMIN'),
           (1001,'2021-06-15','9999-12-31','1000','1000','0001','1','DU',50000010,50000101,50000900,'HR-1000','2021-06-15','ADMIN');

    /* PA0002 Personal Data */
    INSERT INTO HR.PA0002 (PERNR,BEGDA,ENDDA,ANRED,NACHN,VORNA,GBDAT,GESCH,NATIO,FAMST,SPRSL,AEDTM,UNAME)
    VALUES (1000,'2020-03-01','9999-12-31','2','Schmidt','Andreas','1982-07-12','1','DE','1','E','2020-03-01','ADMIN'),
           (1001,'2021-06-15','9999-12-31','1','Nguyen','Linda','1990-11-03','2','US','0','E','2021-06-15','ADMIN');

    /* PA0006 Addresses */
    INSERT INTO HR.PA0006 (PERNR,SUBTY,BEGDA,ENDDA,STRAS,ORT01,PSTLZ,LAND1,AEDTM,UNAME)
    VALUES (1000,'1','2020-03-01','9999-12-31','Hauptstrasse 12','Berlin','10115','DE','2020-03-01','ADMIN'),
           (1001,'1','2021-06-15','9999-12-31','5th Avenue 200','New York','10001','US','2021-06-15','ADMIN');

    /* PA0007 Planned Working Time */
    INSERT INTO HR.PA0007 (PERNR,BEGDA,ENDDA,SCHKZ,EMPCT,WOSTD,AEDTM,UNAME)
    VALUES (1000,'2020-03-01','9999-12-31','FLEX',100.00,40.00,'2020-03-01','ADMIN'),
           (1001,'2021-06-15','9999-12-31','FLEX',100.00,40.00,'2021-06-15','ADMIN');

    /* PA0008 Basic Pay + wage types */
    INSERT INTO HR.PA0008 (PERNR,BEGDA,ENDDA,TRFAR,TRFGB,TRFGR,BSGRD,WAERS,ANSAL,AEDTM,UNAME)
    VALUES (1000,'2020-03-01','9999-12-31','01','01','E4',100.00,'EUR',96000.00,'2020-03-01','ADMIN'),
           (1001,'2021-06-15','9999-12-31','01','01','E2',100.00,'EUR',60000.00,'2021-06-15','ADMIN');

    INSERT INTO HR.PA0008_WageType (PERNR,ENDDA,SEQNR,LineNo,LGART,BETRG,WAERS)
    VALUES (1000,'9999-12-31',1,1,'1010',8000.00,'EUR'),
           (1001,'9999-12-31',1,1,'1010',5000.00,'EUR');

    /* PA0009 Bank Details */
    INSERT INTO HR.PA0009 (PERNR,SUBTY,BEGDA,ENDDA,BANKS,BANKL,BANKN,ZLSCH,WAERS,AEDTM,UNAME)
    VALUES (1000,'0','2020-03-01','9999-12-31','DE','10070000','DE89370400440532013000','U','EUR','2020-03-01','ADMIN'),
           (1001,'0','2021-06-15','9999-12-31','US','021000021','US64SVBKUS6S3300958879','U','EUR','2021-06-15','ADMIN');

    /* PA0105 Communication (email) */
    INSERT INTO HR.PA0105 (PERNR,SUBTY,BEGDA,ENDDA,USRID,USRID_LONG,AEDTM,UNAME)
    VALUES (1000,'0010','2020-03-01','9999-12-31','a.schmidt','a.schmidt@globalcorp.com','2020-03-01','ADMIN'),
           (1001,'0010','2021-06-15','9999-12-31','l.nguyen','l.nguyen@globalcorp.com','2021-06-15','ADMIN');

    /* PA2006 Absence quota - 30 days annual leave 2026 */
    INSERT INTO HR.PA2006 (PERNR,SUBTY,BEGDA,ENDDA,KTART,ANZHL,KVERB,AEDTM,UNAME)
    VALUES (1000,'0100','2026-01-01','2026-12-31','0100',30.00,5.00,'2026-01-01','ADMIN'),
           (1001,'0100','2026-01-01','2026-12-31','0100',25.00,0.00,'2026-01-01','ADMIN');

    /* PA2001 Absence - Linda annual leave */
    INSERT INTO HR.PA2001 (PERNR,SUBTY,BEGDA,ENDDA,AWART,ABWTG,APPROVED,AEDTM,UNAME)
    VALUES (1000,'0100','2026-07-01','2026-07-05','0100',5.00,1,'2026-06-01','ADMIN');
END
GO

PRINT 'Seed reference and demo data loaded.';
GO
