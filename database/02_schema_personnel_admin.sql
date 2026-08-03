/* ============================================================================
   HR Module - Personnel Administration (PA) Infotypes
   Reference: SAP ECC 6.0 EHP8 - Personnel Administration (PA-PA)

   Every infotype record is time-dependent and carries the standard SAP
   infotype key fields:
     PERNR  Personnel number
     SUBTY  Subtype
     OBJPS  Object identification
     SPRPS  Lock indicator ('X' = locked)
     BEGDA  Start date of validity  (SAP "Begin date")
     ENDDA  End   date of validity  (SAP "End date", 9999-12-31 = open ended)
     SEQNR  Sequence number (used when several records share a validity period)
     AEDTM  Last changed on
     UNAME  Changed by
   ============================================================================ */

USE [HRModule];
GO

/* ----------------------------------------------------------------------------
   Master record - PA0003 is the SAP payroll status; here we keep a lightweight
   employee master anchor so PERNR referential integrity can be enforced.
   The real personal data lives in PA0002.
   ---------------------------------------------------------------------------- */
IF OBJECT_ID(N'HR.EmployeeMaster', N'U') IS NULL
BEGIN
    CREATE TABLE HR.EmployeeMaster
    (
        PERNR       INT          NOT NULL,          -- Personnel number (8 digit in SAP)
        HireDate    DATE         NULL,              -- First hiring date
        IsActive    BIT          NOT NULL CONSTRAINT DF_EmpMaster_Active DEFAULT (1),
        CreatedOn   DATETIME2(0) NOT NULL CONSTRAINT DF_EmpMaster_CreatedOn DEFAULT (SYSDATETIME()),
        CONSTRAINT PK_EmployeeMaster PRIMARY KEY (PERNR)
    );
END
GO

/* ----------------------------------------------------------------------------
   PA0000 - Actions (Infotype 0000, Massnahmen)
   Records every personnel action (hiring, org change, leaving, ...).
   ---------------------------------------------------------------------------- */
IF OBJECT_ID(N'HR.PA0000', N'U') IS NULL
BEGIN
    CREATE TABLE HR.PA0000
    (
        PERNR   INT          NOT NULL,             -- Personnel number
        SUBTY   VARCHAR(4)   NOT NULL CONSTRAINT DF_PA0000_SUBTY DEFAULT (''),
        OBJPS   VARCHAR(2)   NOT NULL CONSTRAINT DF_PA0000_OBJPS DEFAULT (''),
        SPRPS   CHAR(1)      NOT NULL CONSTRAINT DF_PA0000_SPRPS DEFAULT (' '),
        BEGDA   DATE         NOT NULL,
        ENDDA   DATE         NOT NULL CONSTRAINT DF_PA0000_ENDDA DEFAULT ('9999-12-31'),
        SEQNR   INT          NOT NULL CONSTRAINT DF_PA0000_SEQNR DEFAULT (1),
        MASSN   VARCHAR(2)   NOT NULL,             -- Action type   (T529A)
        MASSG   VARCHAR(2)   NULL,                 -- Reason for action (T530)
        STAT2   CHAR(1)      NULL,                 -- Employment status (0=left,1=inactive,2=retiree,3=active)
        AEDTM   DATE         NULL,
        UNAME   VARCHAR(12)  NULL,
        CONSTRAINT PK_PA0000 PRIMARY KEY (PERNR, SUBTY, OBJPS, SPRPS, ENDDA, SEQNR),
        CONSTRAINT FK_PA0000_Emp FOREIGN KEY (PERNR) REFERENCES HR.EmployeeMaster(PERNR)
    );
END
GO

/* ----------------------------------------------------------------------------
   PA0001 - Organizational Assignment (Infotype 0001)
   Links the employee to enterprise & personnel structure and to OM objects.
   ---------------------------------------------------------------------------- */
IF OBJECT_ID(N'HR.PA0001', N'U') IS NULL
BEGIN
    CREATE TABLE HR.PA0001
    (
        PERNR   INT          NOT NULL,
        SUBTY   VARCHAR(4)   NOT NULL CONSTRAINT DF_PA0001_SUBTY DEFAULT (''),
        OBJPS   VARCHAR(2)   NOT NULL CONSTRAINT DF_PA0001_OBJPS DEFAULT (''),
        SPRPS   CHAR(1)      NOT NULL CONSTRAINT DF_PA0001_SPRPS DEFAULT (' '),
        BEGDA   DATE         NOT NULL,
        ENDDA   DATE         NOT NULL CONSTRAINT DF_PA0001_ENDDA DEFAULT ('9999-12-31'),
        SEQNR   INT          NOT NULL CONSTRAINT DF_PA0001_SEQNR DEFAULT (1),
        BUKRS   VARCHAR(4)   NULL,                 -- Company code   (T001)
        WERKS   VARCHAR(4)   NULL,                 -- Personnel area (T500P)
        BTRTL   VARCHAR(4)   NULL,                 -- Personnel subarea (T001P)
        PERSG   VARCHAR(1)   NULL,                 -- Employee group    (T501)
        PERSK   VARCHAR(2)   NULL,                 -- Employee subgroup (T503K)
        ORGEH   INT          NULL,                 -- Organizational unit (HRP1000, O)
        PLANS   INT          NULL,                 -- Position            (HRP1000, S)
        STELL   INT          NULL,                 -- Job                 (HRP1000, C)
        KOSTL   VARCHAR(10)  NULL,                 -- Cost center
        ABKRS   VARCHAR(2)   NULL,                 -- Payroll area
        SACHZ   VARCHAR(3)   NULL,                 -- Administrator
        AEDTM   DATE         NULL,
        UNAME   VARCHAR(12)  NULL,
        CONSTRAINT PK_PA0001 PRIMARY KEY (PERNR, SUBTY, OBJPS, SPRPS, ENDDA, SEQNR),
        CONSTRAINT FK_PA0001_Emp FOREIGN KEY (PERNR) REFERENCES HR.EmployeeMaster(PERNR)
    );
END
GO

/* ----------------------------------------------------------------------------
   PA0002 - Personal Data (Infotype 0002)
   ---------------------------------------------------------------------------- */
IF OBJECT_ID(N'HR.PA0002', N'U') IS NULL
BEGIN
    CREATE TABLE HR.PA0002
    (
        PERNR   INT          NOT NULL,
        SUBTY   VARCHAR(4)   NOT NULL CONSTRAINT DF_PA0002_SUBTY DEFAULT (''),
        OBJPS   VARCHAR(2)   NOT NULL CONSTRAINT DF_PA0002_OBJPS DEFAULT (''),
        SPRPS   CHAR(1)      NOT NULL CONSTRAINT DF_PA0002_SPRPS DEFAULT (' '),
        BEGDA   DATE         NOT NULL,
        ENDDA   DATE         NOT NULL CONSTRAINT DF_PA0002_ENDDA DEFAULT ('9999-12-31'),
        SEQNR   INT          NOT NULL CONSTRAINT DF_PA0002_SEQNR DEFAULT (1),
        ANRED   VARCHAR(1)   NULL,                 -- Form of address key
        NACHN   NVARCHAR(40) NOT NULL,             -- Last name
        VORNA   NVARCHAR(40) NOT NULL,             -- First name
        MIDNM   NVARCHAR(40) NULL,                 -- Middle name
        RUFNM   NVARCHAR(40) NULL,                 -- Nickname / known-as
        TITEL   VARCHAR(15)  NULL,                 -- Title
        GBDAT   DATE         NULL,                 -- Date of birth
        GBORT   NVARCHAR(40) NULL,                 -- Place of birth
        GESCH   CHAR(1)      NULL,                 -- Gender key (1=male,2=female,3=other/undefined)
        NATIO   VARCHAR(3)   NULL,                 -- Nationality (T005)
        FAMST   VARCHAR(1)   NULL,                 -- Marital status key
        SPRSL   VARCHAR(1)   NULL,                 -- Language key
        AEDTM   DATE         NULL,
        UNAME   VARCHAR(12)  NULL,
        CONSTRAINT PK_PA0002 PRIMARY KEY (PERNR, SUBTY, OBJPS, SPRPS, ENDDA, SEQNR),
        CONSTRAINT FK_PA0002_Emp FOREIGN KEY (PERNR) REFERENCES HR.EmployeeMaster(PERNR)
    );
END
GO

/* ----------------------------------------------------------------------------
   PA0006 - Addresses (Infotype 0006). SUBTY = address type (T591A).
   ---------------------------------------------------------------------------- */
IF OBJECT_ID(N'HR.PA0006', N'U') IS NULL
BEGIN
    CREATE TABLE HR.PA0006
    (
        PERNR   INT          NOT NULL,
        SUBTY   VARCHAR(4)   NOT NULL CONSTRAINT DF_PA0006_SUBTY DEFAULT ('1'),  -- 1 = permanent residence
        OBJPS   VARCHAR(2)   NOT NULL CONSTRAINT DF_PA0006_OBJPS DEFAULT (''),
        SPRPS   CHAR(1)      NOT NULL CONSTRAINT DF_PA0006_SPRPS DEFAULT (' '),
        BEGDA   DATE         NOT NULL,
        ENDDA   DATE         NOT NULL CONSTRAINT DF_PA0006_ENDDA DEFAULT ('9999-12-31'),
        SEQNR   INT          NOT NULL CONSTRAINT DF_PA0006_SEQNR DEFAULT (1),
        STRAS   NVARCHAR(60) NULL,                 -- Street and house number
        ORT01   NVARCHAR(40) NULL,                 -- City
        ORT02   NVARCHAR(40) NULL,                 -- District
        PSTLZ   VARCHAR(10)  NULL,                 -- Postal code
        LAND1   VARCHAR(3)   NULL,                 -- Country key (T005)
        STATE   VARCHAR(3)   NULL,                 -- Region / state
        TELNR   VARCHAR(20)  NULL,                 -- Telephone number
        AEDTM   DATE         NULL,
        UNAME   VARCHAR(12)  NULL,
        CONSTRAINT PK_PA0006 PRIMARY KEY (PERNR, SUBTY, OBJPS, SPRPS, ENDDA, SEQNR),
        CONSTRAINT FK_PA0006_Emp FOREIGN KEY (PERNR) REFERENCES HR.EmployeeMaster(PERNR)
    );
END
GO

/* ----------------------------------------------------------------------------
   PA0007 - Planned Working Time (Infotype 0007)
   ---------------------------------------------------------------------------- */
IF OBJECT_ID(N'HR.PA0007', N'U') IS NULL
BEGIN
    CREATE TABLE HR.PA0007
    (
        PERNR   INT           NOT NULL,
        SUBTY   VARCHAR(4)    NOT NULL CONSTRAINT DF_PA0007_SUBTY DEFAULT (''),
        OBJPS   VARCHAR(2)    NOT NULL CONSTRAINT DF_PA0007_OBJPS DEFAULT (''),
        SPRPS   CHAR(1)       NOT NULL CONSTRAINT DF_PA0007_SPRPS DEFAULT (' '),
        BEGDA   DATE          NOT NULL,
        ENDDA   DATE          NOT NULL CONSTRAINT DF_PA0007_ENDDA DEFAULT ('9999-12-31'),
        SEQNR   INT           NOT NULL CONSTRAINT DF_PA0007_SEQNR DEFAULT (1),
        SCHKZ   VARCHAR(8)    NULL,                -- Work schedule rule (T508A)
        ZTERF   CHAR(1)       NULL,                -- Time management status
        EMPCT   DECIMAL(5,2)  NULL,                -- Employment percentage
        WOSTD   DECIMAL(6,2)  NULL,                -- Weekly working hours
        MOSTD   DECIMAL(7,2)  NULL,                -- Monthly working hours
        JRSTD   DECIMAL(8,2)  NULL,                -- Annual working hours
        AEDTM   DATE          NULL,
        UNAME   VARCHAR(12)   NULL,
        CONSTRAINT PK_PA0007 PRIMARY KEY (PERNR, SUBTY, OBJPS, SPRPS, ENDDA, SEQNR),
        CONSTRAINT FK_PA0007_Emp FOREIGN KEY (PERNR) REFERENCES HR.EmployeeMaster(PERNR)
    );
END
GO

/* ----------------------------------------------------------------------------
   PA0008 - Basic Pay (Infotype 0008). Wage types are stored in PA0008_WageType.
   ---------------------------------------------------------------------------- */
IF OBJECT_ID(N'HR.PA0008', N'U') IS NULL
BEGIN
    CREATE TABLE HR.PA0008
    (
        PERNR   INT           NOT NULL,
        SUBTY   VARCHAR(4)    NOT NULL CONSTRAINT DF_PA0008_SUBTY DEFAULT (''),
        OBJPS   VARCHAR(2)    NOT NULL CONSTRAINT DF_PA0008_OBJPS DEFAULT (''),
        SPRPS   CHAR(1)       NOT NULL CONSTRAINT DF_PA0008_SPRPS DEFAULT (' '),
        BEGDA   DATE          NOT NULL,
        ENDDA   DATE          NOT NULL CONSTRAINT DF_PA0008_ENDDA DEFAULT ('9999-12-31'),
        SEQNR   INT           NOT NULL CONSTRAINT DF_PA0008_SEQNR DEFAULT (1),
        TRFAR   VARCHAR(2)    NULL,                -- Pay scale type   (T510A)
        TRFGB   VARCHAR(2)    NULL,                -- Pay scale area   (T510G)
        TRFGR   VARCHAR(8)    NULL,                -- Pay scale group
        TRFST   VARCHAR(2)    NULL,                -- Pay scale level
        BSGRD   DECIMAL(5,2)  NULL,                -- Capacity utilization level (%)
        DIVGV   DECIMAL(6,2)  NULL,                -- Working hours per pay period
        WAERS   VARCHAR(5)    NULL,                -- Currency key (T500C)
        ANSAL   DECIMAL(15,2) NULL,                -- Annual salary
        AEDTM   DATE          NULL,
        UNAME   VARCHAR(12)   NULL,
        CONSTRAINT PK_PA0008 PRIMARY KEY (PERNR, SUBTY, OBJPS, SPRPS, ENDDA, SEQNR),
        CONSTRAINT FK_PA0008_Emp FOREIGN KEY (PERNR) REFERENCES HR.EmployeeMaster(PERNR),
        /* Alternate key so the wage-type sub-records can link on the natural
           (PERNR, ENDDA, SEQNR) tuple (SUBTY/OBJPS/SPRPS are constant here). */
        CONSTRAINT UQ_PA0008_Natural UNIQUE (PERNR, ENDDA, SEQNR)
    );
END
GO

/* Wage type sub-records of Basic Pay (SAP fields LGART/BETRG/ANZHL). */
IF OBJECT_ID(N'HR.PA0008_WageType', N'U') IS NULL
BEGIN
    CREATE TABLE HR.PA0008_WageType
    (
        PERNR   INT           NOT NULL,
        ENDDA   DATE          NOT NULL,
        SEQNR   INT           NOT NULL,
        LineNo  INT           NOT NULL,            -- 1..40 wage type lines
        LGART   VARCHAR(4)    NOT NULL,            -- Wage type   (T512W)
        BETRG   DECIMAL(15,2) NULL,                -- Amount
        WAERS   VARCHAR(5)    NULL,                -- Currency
        ANZHL   DECIMAL(9,2)  NULL,                -- Number / quantity
        CONSTRAINT PK_PA0008_WT PRIMARY KEY (PERNR, ENDDA, SEQNR, LineNo),
        CONSTRAINT FK_PA0008_WT FOREIGN KEY (PERNR, ENDDA, SEQNR)
            REFERENCES HR.PA0008(PERNR, ENDDA, SEQNR)   -- links on the natural key (UQ_PA0008_Natural)
    );
END
GO

/* ----------------------------------------------------------------------------
   PA0009 - Bank Details (Infotype 0009)
   ---------------------------------------------------------------------------- */
IF OBJECT_ID(N'HR.PA0009', N'U') IS NULL
BEGIN
    CREATE TABLE HR.PA0009
    (
        PERNR   INT           NOT NULL,
        SUBTY   VARCHAR(4)    NOT NULL CONSTRAINT DF_PA0009_SUBTY DEFAULT ('0'),  -- 0 = main bank
        OBJPS   VARCHAR(2)    NOT NULL CONSTRAINT DF_PA0009_OBJPS DEFAULT (''),
        SPRPS   CHAR(1)       NOT NULL CONSTRAINT DF_PA0009_SPRPS DEFAULT (' '),
        BEGDA   DATE          NOT NULL,
        ENDDA   DATE          NOT NULL CONSTRAINT DF_PA0009_ENDDA DEFAULT ('9999-12-31'),
        SEQNR   INT           NOT NULL CONSTRAINT DF_PA0009_SEQNR DEFAULT (1),
        BNKSA   VARCHAR(2)    NULL,                -- Bank details type
        EMFTX   NVARCHAR(40)  NULL,                -- Payee name
        BANKS   VARCHAR(3)    NULL,                -- Bank country key
        BANKL   VARCHAR(15)   NULL,                -- Bank key / routing
        BANKN   VARCHAR(34)   NULL,                -- Bank account number (IBAN capable)
        ZLSCH   CHAR(1)       NULL,                -- Payment method
        WAERS   VARCHAR(5)    NULL,                -- Currency
        AEDTM   DATE          NULL,
        UNAME   VARCHAR(12)   NULL,
        CONSTRAINT PK_PA0009 PRIMARY KEY (PERNR, SUBTY, OBJPS, SPRPS, ENDDA, SEQNR),
        CONSTRAINT FK_PA0009_Emp FOREIGN KEY (PERNR) REFERENCES HR.EmployeeMaster(PERNR)
    );
END
GO

/* ----------------------------------------------------------------------------
   PA0105 - Communication (Infotype 0105). SUBTY = communication type (T591A):
   0010 email, 0020 phone, MAIL system user, CELL mobile, ...
   ---------------------------------------------------------------------------- */
IF OBJECT_ID(N'HR.PA0105', N'U') IS NULL
BEGIN
    CREATE TABLE HR.PA0105
    (
        PERNR   INT           NOT NULL,
        SUBTY   VARCHAR(4)    NOT NULL,            -- Communication type (USRTY)
        OBJPS   VARCHAR(2)    NOT NULL CONSTRAINT DF_PA0105_OBJPS DEFAULT (''),
        SPRPS   CHAR(1)       NOT NULL CONSTRAINT DF_PA0105_SPRPS DEFAULT (' '),
        BEGDA   DATE          NOT NULL,
        ENDDA   DATE          NOT NULL CONSTRAINT DF_PA0105_ENDDA DEFAULT ('9999-12-31'),
        SEQNR   INT           NOT NULL CONSTRAINT DF_PA0105_SEQNR DEFAULT (1),
        USRID   NVARCHAR(100) NULL,                -- Communication ID / value (short)
        USRID_LONG NVARCHAR(241) NULL,             -- Long form (e.g. email)
        AEDTM   DATE          NULL,
        UNAME   VARCHAR(12)   NULL,
        CONSTRAINT PK_PA0105 PRIMARY KEY (PERNR, SUBTY, OBJPS, SPRPS, ENDDA, SEQNR),
        CONSTRAINT FK_PA0105_Emp FOREIGN KEY (PERNR) REFERENCES HR.EmployeeMaster(PERNR)
    );
END
GO

/* Indexes on validity for fast "record valid on key date" lookups. */
IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = 'IX_PA0001_Valid')
    CREATE INDEX IX_PA0001_Valid ON HR.PA0001 (PERNR, BEGDA, ENDDA);
IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = 'IX_PA0002_Valid')
    CREATE INDEX IX_PA0002_Valid ON HR.PA0002 (PERNR, BEGDA, ENDDA);
IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = 'IX_PA0001_Org')
    CREATE INDEX IX_PA0001_Org ON HR.PA0001 (ORGEH, BEGDA, ENDDA);
GO
