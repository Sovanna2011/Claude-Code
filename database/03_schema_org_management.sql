/* ============================================================================
   HR Module - Organizational Management (OM)
   Reference: SAP ECC 6.0 EHP8 - Organizational Management (PA-OS)

   OM in SAP is built from objects and the relationships between them:
     HRP1000  Objects   (OTYPE O=Org unit, S=Position, C=Job, P=Person)
     HRP1001  Relationships between objects (evaluation paths)
     HRP1002  Description (infotype 1002, verbal text) - simplified here

   Standard relationship (RELAT) / sign (RSIGN) combinations used:
     A 002 "reports to"        S->S (position reports to position)
     B 002 "is line supervisor of"
     A 003 "belongs to"        S->O (position belongs to org unit)
     B 003 "incorporates"      O->S
     A 007 "is described by"   S->C (position described by job)
     B 008 "holder"            S->P (position is held by person)
     B 012 "is managed by"     O->S (org unit managed by chief position)
   ============================================================================ */

USE [HRModule];
GO

/* ----------------------------------------------------------------------------
   HRP1000 - Object (Org unit / Position / Job)
   ---------------------------------------------------------------------------- */
IF OBJECT_ID(N'HR.HRP1000', N'U') IS NULL
BEGIN
    CREATE TABLE HR.HRP1000
    (
        MANDT   VARCHAR(3)   NOT NULL CONSTRAINT DF_HRP1000_MANDT DEFAULT ('100'), -- Client
        PLVAR   VARCHAR(2)   NOT NULL CONSTRAINT DF_HRP1000_PLVAR DEFAULT ('01'),  -- Plan version (active)
        OTYPE   VARCHAR(2)   NOT NULL,             -- Object type O/S/C/P
        OBJID   INT          NOT NULL,             -- Object ID
        ISTAT   VARCHAR(1)   NOT NULL CONSTRAINT DF_HRP1000_ISTAT DEFAULT ('1'),   -- Planning status (1=active)
        BEGDA   DATE         NOT NULL,
        ENDDA   DATE         NOT NULL CONSTRAINT DF_HRP1000_ENDDA DEFAULT ('9999-12-31'),
        SEQNR   VARCHAR(3)   NOT NULL CONSTRAINT DF_HRP1000_SEQNR DEFAULT ('000'),
        LANGU   VARCHAR(1)   NOT NULL CONSTRAINT DF_HRP1000_LANGU DEFAULT ('E'),
        SHORT   NVARCHAR(12) NULL,                 -- Object abbreviation
        STEXT   NVARCHAR(40) NULL,                 -- Object name / description
        AEDTM   DATE         NULL,
        UNAME   VARCHAR(12)  NULL,
        CONSTRAINT PK_HRP1000 PRIMARY KEY (PLVAR, OTYPE, OBJID, ISTAT, ENDDA, BEGDA, SEQNR)
    );
END
GO

/* ----------------------------------------------------------------------------
   HRP1001 - Relationships
   ---------------------------------------------------------------------------- */
IF OBJECT_ID(N'HR.HRP1001', N'U') IS NULL
BEGIN
    CREATE TABLE HR.HRP1001
    (
        MANDT   VARCHAR(3)   NOT NULL CONSTRAINT DF_HRP1001_MANDT DEFAULT ('100'),
        PLVAR   VARCHAR(2)   NOT NULL CONSTRAINT DF_HRP1001_PLVAR DEFAULT ('01'),
        OTYPE   VARCHAR(2)   NOT NULL,             -- Source object type
        OBJID   INT          NOT NULL,             -- Source object ID
        ISTAT   VARCHAR(1)   NOT NULL CONSTRAINT DF_HRP1001_ISTAT DEFAULT ('1'),
        BEGDA   DATE         NOT NULL,
        ENDDA   DATE         NOT NULL CONSTRAINT DF_HRP1001_ENDDA DEFAULT ('9999-12-31'),
        SEQNR   VARCHAR(3)   NOT NULL CONSTRAINT DF_HRP1001_SEQNR DEFAULT ('000'),
        RSIGN   VARCHAR(1)   NOT NULL,             -- Relationship specification A/B
        RELAT   VARCHAR(3)   NOT NULL,             -- Relationship (002,003,007,008,012...)
        SCLAS   VARCHAR(2)   NOT NULL,             -- Type of related object
        SOBID   VARCHAR(45)  NOT NULL,             -- ID of related object
        PRIOX   VARCHAR(4)   NULL,                 -- Priority
        AEDTM   DATE         NULL,
        UNAME   VARCHAR(12)  NULL,
        CONSTRAINT PK_HRP1001 PRIMARY KEY (PLVAR, OTYPE, OBJID, ISTAT, ENDDA, BEGDA, RSIGN, RELAT, SCLAS, SOBID, SEQNR)
    );
END
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = 'IX_HRP1001_Source')
    CREATE INDEX IX_HRP1001_Source ON HR.HRP1001 (OTYPE, OBJID, RELAT, RSIGN, BEGDA, ENDDA);
IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = 'IX_HRP1001_Target')
    CREATE INDEX IX_HRP1001_Target ON HR.HRP1001 (SCLAS, SOBID, RELAT, RSIGN, BEGDA, ENDDA);
GO
