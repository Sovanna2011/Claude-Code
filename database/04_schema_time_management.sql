/* ============================================================================
   HR Module - Time Management (PT) - core objects
   Reference: SAP ECC 6.0 EHP8 - Time Management (PT)

   Absence / attendance records mirror infotype 2001 (Absences) and 2002
   (Attendances). Quota records mirror infotype 2006 (Absence Quotas).
   ============================================================================ */

USE [HRModule];
GO

/* ----------------------------------------------------------------------------
   PA2001 - Absences (Infotype 2001). SUBTY = absence type (AWART, T554S).
   ---------------------------------------------------------------------------- */
IF OBJECT_ID(N'HR.PA2001', N'U') IS NULL
BEGIN
    CREATE TABLE HR.PA2001
    (
        PERNR   INT           NOT NULL,
        SUBTY   VARCHAR(4)    NOT NULL,            -- Absence type (AWART)
        OBJPS   VARCHAR(2)    NOT NULL CONSTRAINT DF_PA2001_OBJPS DEFAULT (''),
        SPRPS   CHAR(1)       NOT NULL CONSTRAINT DF_PA2001_SPRPS DEFAULT (' '),
        BEGDA   DATE          NOT NULL,
        ENDDA   DATE          NOT NULL,
        SEQNR   INT           NOT NULL CONSTRAINT DF_PA2001_SEQNR DEFAULT (1),
        AWART   VARCHAR(4)     NOT NULL,           -- Attendance/Absence type
        ABWTG   DECIMAL(7,2)  NULL,                -- Absence days
        STDAZ   DECIMAL(7,2)  NULL,                -- Absence hours
        BEGUZ   TIME(0)       NULL,                -- Start time
        ENDUZ   TIME(0)       NULL,                -- End time
        APPROVED BIT          NOT NULL CONSTRAINT DF_PA2001_Appr DEFAULT (0),
        AEDTM   DATE          NULL,
        UNAME   VARCHAR(12)   NULL,
        CONSTRAINT PK_PA2001 PRIMARY KEY (PERNR, SUBTY, OBJPS, SPRPS, ENDDA, BEGDA, SEQNR),
        CONSTRAINT FK_PA2001_Emp FOREIGN KEY (PERNR) REFERENCES HR.EmployeeMaster(PERNR)
    );
END
GO

/* ----------------------------------------------------------------------------
   PA2006 - Absence Quotas (Infotype 2006). E.g. annual leave entitlement.
   ---------------------------------------------------------------------------- */
IF OBJECT_ID(N'HR.PA2006', N'U') IS NULL
BEGIN
    CREATE TABLE HR.PA2006
    (
        PERNR   INT           NOT NULL,
        SUBTY   VARCHAR(4)    NOT NULL,            -- Quota type (KTART, T556A)
        OBJPS   VARCHAR(2)    NOT NULL CONSTRAINT DF_PA2006_OBJPS DEFAULT (''),
        SPRPS   CHAR(1)       NOT NULL CONSTRAINT DF_PA2006_SPRPS DEFAULT (' '),
        BEGDA   DATE          NOT NULL,
        ENDDA   DATE          NOT NULL,
        SEQNR   INT           NOT NULL CONSTRAINT DF_PA2006_SEQNR DEFAULT (1),
        KTART   VARCHAR(4)    NOT NULL,            -- Absence quota type
        ANZHL   DECIMAL(9,2)  NOT NULL,            -- Quota entitlement number
        KVERB   DECIMAL(9,2)  NOT NULL CONSTRAINT DF_PA2006_KVERB DEFAULT (0), -- Deducted amount
        DESTA   DATE          NULL,                -- Deduction from
        DEEND   DATE          NULL,                -- Deduction to
        AEDTM   DATE          NULL,
        UNAME   VARCHAR(12)   NULL,
        CONSTRAINT PK_PA2006 PRIMARY KEY (PERNR, SUBTY, OBJPS, SPRPS, ENDDA, BEGDA, SEQNR),
        CONSTRAINT FK_PA2006_Emp FOREIGN KEY (PERNR) REFERENCES HR.EmployeeMaster(PERNR)
    );
END
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = 'IX_PA2001_Period')
    CREATE INDEX IX_PA2001_Period ON HR.PA2001 (PERNR, BEGDA, ENDDA);
GO
