/* ============================================================================
   HR Module - Seed data for extended infotypes
   Adds customizing (domain values, contract types, attendance types) and demo
   records for employees 1000 (Andreas Schmidt) and 1001 (Linda Nguyen).
   Idempotent.
   ============================================================================ */
USE [HRModule];
GO

/* --- Domain fixed values for the new infotypes --------------------------- */
MERGE HR.DomainValue AS t
USING (VALUES
    /* Family / related person type (IT0021 SUBTY) */
    ('FAMSA','1','Spouse'),('FAMSA','2','Child'),('FAMSA','6','Emergency contact'),
    ('FAMSA','11','Father'),('FAMSA','12','Mother'),
    /* Education establishment type (IT0022 SUBTY) */
    ('SLART','10','University'),('SLART','20','Secondary school'),
    ('SLART','30','Vocational training'),('SLART','40','Doctorate'),
    /* Task type for monitoring of dates (IT0019 SUBTY) */
    ('TMART','01','Expiry of probation'),('TMART','02','Work permit expiry'),
    ('TMART','03','Contract end'),('TMART','04','Next appraisal'),
    /* Qualification group (IT0024 SUBTY) */
    ('QUALG','01','Technical'),('QUALG','02','Language'),('QUALG','03','Leadership'),
    /* Address type (IT0006 SUBTY, ANSSA) */
    ('ANSSA','1','Permanent residence'),('ANSSA','2','Temporary residence'),
    ('ANSSA','3','Home address'),('ANSSA','4','Mailing address'),('ANSSA','5','Emergency address')
) AS s(Domain,ValueKey,ValueTxt)
ON t.Domain=s.Domain AND t.ValueKey=s.ValueKey
WHEN NOT MATCHED THEN INSERT(Domain,ValueKey,ValueTxt) VALUES(s.Domain,s.ValueKey,s.ValueTxt);
GO

/* --- Contract types (T547T) ---------------------------------------------- */
MERGE HR.T547T AS t
USING (VALUES ('01','Permanent'),('02','Fixed-term'),('03','Temporary'),('04','Internship'))
    AS s(CTTYP,CTTXT) ON t.CTTYP=s.CTTYP
WHEN NOT MATCHED THEN INSERT(CTTYP,CTTXT) VALUES(s.CTTYP,s.CTTXT);
GO

/* --- Additional attendance types (reuse T554S, KENNZ='P') ---------------- */
MERGE HR.T554S AS t
USING (VALUES ('01','0500','Training','P'),('01','0600','Conference','P'))
    AS s(MOABW,AWART,ATEXT,KENNZ) ON t.MOABW=s.MOABW AND t.AWART=s.AWART
WHEN NOT MATCHED THEN INSERT(MOABW,AWART,ATEXT,KENNZ) VALUES(s.MOABW,s.AWART,s.ATEXT,s.KENNZ);
GO

/* --- Demo records (only if not already present) -------------------------- */
IF NOT EXISTS (SELECT 1 FROM HR.PA0021 WHERE PERNR IN (1000,1001))
BEGIN
    /* IT0016 Contract Elements */
    INSERT INTO HR.PA0016 (PERNR,BEGDA,ENDDA,CTTYP,PRBEZ,KDGFB,KDGF2,AEDTM,UNAME) VALUES
      (1000,'2020-03-01','9999-12-31','01',6.0,3.0,3.0,'2020-03-01','ADMIN'),
      (1001,'2021-06-15','9999-12-31','02',3.0,1.0,1.0,'2021-06-15','ADMIN');

    /* IT0019 Monitoring of Dates */
    INSERT INTO HR.PA0019 (PERNR,SUBTY,BEGDA,ENDDA,TERMN,MNDAT,AEDTM,UNAME) VALUES
      (1001,'01','2021-06-15','9999-12-31','2021-09-15','2021-09-01','2021-06-15','ADMIN'),
      (1001,'04','2021-06-15','9999-12-31','2026-06-15','2026-06-01','2021-06-15','ADMIN'),
      (1000,'04','2020-03-01','9999-12-31','2026-03-01','2026-02-15','2020-03-01','ADMIN');

    /* IT0021 Family Members */
    INSERT INTO HR.PA0021 (PERNR,SUBTY,OBJPS,BEGDA,ENDDA,FANAM,FAVOR,FGBDT,FASEX,FGBLD,AEDTM,UNAME) VALUES
      (1000,'1','01','2010-06-20','9999-12-31','Schmidt','Julia','1984-02-18','2','DE','2020-03-01','ADMIN'),
      (1000,'2','01','2012-04-11','9999-12-31','Schmidt','Max','2012-04-11','1','DE','2020-03-01','ADMIN'),
      (1000,'2','02','2015-09-30','9999-12-31','Schmidt','Emma','2015-09-30','2','DE','2020-03-01','ADMIN'),
      (1001,'6','01','2021-06-15','9999-12-31','Nguyen','Peter','1988-01-05','1','US','2021-06-15','ADMIN');

    /* IT0022 Education */
    INSERT INTO HR.PA0022 (PERNR,SUBTY,BEGDA,ENDDA,SLABS,INSTI,SLAND,SFACH,SLGRA,AEDTM,UNAME) VALUES
      (1000,'10','2001-10-01','2006-07-31','Diplom (Master)','TU Berlin','DE','Business Administration','1.7','2020-03-01','ADMIN'),
      (1001,'10','2009-09-01','2013-05-31','B.Sc.','NYU','US','Human Resources Mgmt','3.8 GPA','2021-06-15','ADMIN'),
      (1001,'40','2013-09-01','2017-06-30','Ph.D.','Columbia University','US','Organizational Psychology','','2021-06-15','ADMIN');

    /* IT0023 Previous Employers (work experience) */
    INSERT INTO HR.PA0023 (PERNR,BEGDA,ENDDA,ARBGB,ORT01,LAND1,TASK,BRANC,AEDTM,UNAME) VALUES
      (1000,'2006-08-01','2020-02-29','Muster GmbH','Munich','DE','HR Business Partner','Manufacturing','2020-03-01','ADMIN'),
      (1001,'2017-07-01','2021-05-31','Acme Corp','Boston','US','HR Analyst','Technology','2021-06-15','ADMIN');

    /* IT0024 Qualifications / Skills */
    INSERT INTO HR.PA0024 (PERNR,SUBTY,BEGDA,ENDDA,QUALI,AUSPR,AEDTM,UNAME) VALUES
      (1000,'03','2020-03-01','9999-12-31','People leadership',8,'2020-03-01','ADMIN'),
      (1000,'02','2020-03-01','9999-12-31','English (fluent)',7,'2020-03-01','ADMIN'),
      (1000,'01','2020-03-01','9999-12-31','SAP SuccessFactors',6,'2020-03-01','ADMIN'),
      (1001,'01','2021-06-15','9999-12-31','HR Analytics',8,'2021-06-15','ADMIN'),
      (1001,'02','2021-06-15','9999-12-31','Spanish (intermediate)',5,'2021-06-15','ADMIN');

    /* IT2002 Attendances */
    INSERT INTO HR.PA2002 (PERNR,SUBTY,BEGDA,ENDDA,AWART,ABWTG,STDAZ,AEDTM,UNAME) VALUES
      (1000,'0500','2026-05-04','2026-05-06','0500',3.00,24.00,'2026-04-20','ADMIN'),
      (1001,'0400','2026-03-10','2026-03-12','0400',3.00,24.00,'2026-03-01','ADMIN'),
      (1001,'1000','2026-04-15','2026-04-15','1000',NULL,3.50,'2026-04-15','ADMIN');
END
GO

PRINT 'Extended infotype seed data loaded.';
GO
