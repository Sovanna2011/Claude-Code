/* ============================================================================
   HR Module - Extended Personnel Administration Infotypes
   Reference: SAP ECC 6.0 EHP8 - PA (Personal Development / Time)

   Adds the infotypes needed for a fuller HR master record:
     PA0016  Contract Elements
     PA0019  Monitoring of Dates (task deadlines)
     PA0021  Family Members / Dependents
     PA0022  Education
     PA0023  Other/Previous Employers (work experience)
     PA0024  Qualifications (skills)
     PA2002  Attendances

   All tables carry the standard SAP infotype key
   (PERNR, SUBTY, OBJPS, SPRPS, ENDDA, SEQNR) and administrative fields.
   ============================================================================ */

USE [HRModule];
GO

/* ----------------------------------------------------------------------------
   PA0016 - Contract Elements (Infotype 0016). Time constraint 1 (one valid).
   ---------------------------------------------------------------------------- */
IF OBJECT_ID(N'HR.PA0016', N'U') IS NULL
BEGIN
    CREATE TABLE HR.PA0016
    (
        PERNR   INT          NOT NULL,
        SUBTY   VARCHAR(4)   NOT NULL CONSTRAINT DF_PA0016_SUBTY DEFAULT (''),
        OBJPS   VARCHAR(2)   NOT NULL CONSTRAINT DF_PA0016_OBJPS DEFAULT (''),
        SPRPS   CHAR(1)      NOT NULL CONSTRAINT DF_PA0016_SPRPS DEFAULT (' '),
        BEGDA   DATE         NOT NULL,
        ENDDA   DATE         NOT NULL CONSTRAINT DF_PA0016_ENDDA DEFAULT ('9999-12-31'),
        SEQNR   INT          NOT NULL CONSTRAINT DF_PA0016_SEQNR DEFAULT (1),
        CTTYP   VARCHAR(2)   NULL,               -- Contract type (T547T)
        PRBEZ   DECIMAL(4,1) NULL,               -- Probation period (months)
        KDGFB   DECIMAL(4,1) NULL,               -- Notice period, employer (months)
        KDGF2   DECIMAL(4,1) NULL,               -- Notice period, employee (months)
        EGZuo   VARCHAR(2)   NULL,               -- (spare) grouping
        AEDTM   DATE         NULL,
        UNAME   VARCHAR(12)  NULL,
        CONSTRAINT PK_PA0016 PRIMARY KEY (PERNR, SUBTY, OBJPS, SPRPS, ENDDA, SEQNR),
        CONSTRAINT FK_PA0016_Emp FOREIGN KEY (PERNR) REFERENCES HR.EmployeeMaster(PERNR)
    );
END
GO

/* ----------------------------------------------------------------------------
   PA0019 - Monitoring of Dates (Infotype 0019). SUBTY = task type (TMART).
   ---------------------------------------------------------------------------- */
IF OBJECT_ID(N'HR.PA0019', N'U') IS NULL
BEGIN
    CREATE TABLE HR.PA0019
    (
        PERNR   INT          NOT NULL,
        SUBTY   VARCHAR(4)   NOT NULL,           -- Task type (TMART)
        OBJPS   VARCHAR(2)   NOT NULL CONSTRAINT DF_PA0019_OBJPS DEFAULT (''),
        SPRPS   CHAR(1)      NOT NULL CONSTRAINT DF_PA0019_SPRPS DEFAULT (' '),
        BEGDA   DATE         NOT NULL,
        ENDDA   DATE         NOT NULL CONSTRAINT DF_PA0019_ENDDA DEFAULT ('9999-12-31'),
        SEQNR   INT          NOT NULL CONSTRAINT DF_PA0019_SEQNR DEFAULT (1),
        TERMN   DATE         NOT NULL,           -- Date of task / deadline
        MNDAT   DATE         NULL,               -- Reminder date
        REMINDED BIT         NOT NULL CONSTRAINT DF_PA0019_Rem DEFAULT (0),
        AEDTM   DATE         NULL,
        UNAME   VARCHAR(12)  NULL,
        CONSTRAINT PK_PA0019 PRIMARY KEY (PERNR, SUBTY, OBJPS, SPRPS, ENDDA, SEQNR),
        CONSTRAINT FK_PA0019_Emp FOREIGN KEY (PERNR) REFERENCES HR.EmployeeMaster(PERNR)
    );
END
GO

/* ----------------------------------------------------------------------------
   PA0021 - Family Members / Dependents (Infotype 0021). SUBTY = family type.
   ---------------------------------------------------------------------------- */
IF OBJECT_ID(N'HR.PA0021', N'U') IS NULL
BEGIN
    CREATE TABLE HR.PA0021
    (
        PERNR   INT          NOT NULL,
        SUBTY   VARCHAR(4)   NOT NULL,           -- Family/related person type (FAMSA)
        OBJPS   VARCHAR(2)   NOT NULL CONSTRAINT DF_PA0021_OBJPS DEFAULT ('01'),
        SPRPS   CHAR(1)      NOT NULL CONSTRAINT DF_PA0021_SPRPS DEFAULT (' '),
        BEGDA   DATE         NOT NULL,
        ENDDA   DATE         NOT NULL CONSTRAINT DF_PA0021_ENDDA DEFAULT ('9999-12-31'),
        SEQNR   INT          NOT NULL CONSTRAINT DF_PA0021_SEQNR DEFAULT (1),
        FANAM   NVARCHAR(40) NULL,               -- Last name of family member
        FAVOR   NVARCHAR(40) NULL,               -- First name of family member
        FGBDT   DATE         NULL,               -- Date of birth
        FASEX   CHAR(1)      NULL,               -- Gender (1/2)
        FGBLD   VARCHAR(3)   NULL,               -- Country of birth
        FGBOT   NVARCHAR(40) NULL,               -- Place of birth
        AEDTM   DATE         NULL,
        UNAME   VARCHAR(12)  NULL,
        CONSTRAINT PK_PA0021 PRIMARY KEY (PERNR, SUBTY, OBJPS, SPRPS, ENDDA, SEQNR),
        CONSTRAINT FK_PA0021_Emp FOREIGN KEY (PERNR) REFERENCES HR.EmployeeMaster(PERNR)
    );
END
GO

/* ----------------------------------------------------------------------------
   PA0022 - Education (Infotype 0022). SUBTY = education establishment type.
   ---------------------------------------------------------------------------- */
IF OBJECT_ID(N'HR.PA0022', N'U') IS NULL
BEGIN
    CREATE TABLE HR.PA0022
    (
        PERNR   INT          NOT NULL,
        SUBTY   VARCHAR(4)   NOT NULL,           -- Education establishment type (SLART)
        OBJPS   VARCHAR(2)   NOT NULL CONSTRAINT DF_PA0022_OBJPS DEFAULT (''),
        SPRPS   CHAR(1)      NOT NULL CONSTRAINT DF_PA0022_SPRPS DEFAULT (' '),
        BEGDA   DATE         NOT NULL,
        ENDDA   DATE         NOT NULL CONSTRAINT DF_PA0022_ENDDA DEFAULT ('9999-12-31'),
        SEQNR   INT          NOT NULL CONSTRAINT DF_PA0022_SEQNR DEFAULT (1),
        SLABS   NVARCHAR(40) NULL,               -- Certificate / degree
        INSTI   NVARCHAR(60) NULL,               -- Institute / school name
        SLAND   VARCHAR(3)   NULL,               -- Country of establishment
        SFACH   NVARCHAR(40) NULL,               -- Branch of study / major
        SLGRA   NVARCHAR(20) NULL,               -- Final grade
        AEDTM   DATE         NULL,
        UNAME   VARCHAR(12)  NULL,
        CONSTRAINT PK_PA0022 PRIMARY KEY (PERNR, SUBTY, OBJPS, SPRPS, ENDDA, SEQNR),
        CONSTRAINT FK_PA0022_Emp FOREIGN KEY (PERNR) REFERENCES HR.EmployeeMaster(PERNR)
    );
END
GO

/* ----------------------------------------------------------------------------
   PA0023 - Other / Previous Employers (Infotype 0023) - work experience.
   ---------------------------------------------------------------------------- */
IF OBJECT_ID(N'HR.PA0023', N'U') IS NULL
BEGIN
    CREATE TABLE HR.PA0023
    (
        PERNR   INT          NOT NULL,
        SUBTY   VARCHAR(4)   NOT NULL CONSTRAINT DF_PA0023_SUBTY DEFAULT (''),
        OBJPS   VARCHAR(2)   NOT NULL CONSTRAINT DF_PA0023_OBJPS DEFAULT (''),
        SPRPS   CHAR(1)      NOT NULL CONSTRAINT DF_PA0023_SPRPS DEFAULT (' '),
        BEGDA   DATE         NOT NULL,
        ENDDA   DATE         NOT NULL,
        SEQNR   INT          NOT NULL CONSTRAINT DF_PA0023_SEQNR DEFAULT (1),
        ARBGB   NVARCHAR(60) NULL,               -- Previous employer
        ORT01   NVARCHAR(40) NULL,               -- Place
        LAND1   VARCHAR(3)   NULL,               -- Country
        TASK    NVARCHAR(60) NULL,               -- Activity / job title
        BRANC   NVARCHAR(40) NULL,               -- Industry
        AEDTM   DATE         NULL,
        UNAME   VARCHAR(12)  NULL,
        CONSTRAINT PK_PA0023 PRIMARY KEY (PERNR, SUBTY, OBJPS, SPRPS, ENDDA, SEQNR),
        CONSTRAINT FK_PA0023_Emp FOREIGN KEY (PERNR) REFERENCES HR.EmployeeMaster(PERNR)
    );
END
GO

/* ----------------------------------------------------------------------------
   PA0024 - Qualifications / Skills (Infotype 0024).
   ---------------------------------------------------------------------------- */
IF OBJECT_ID(N'HR.PA0024', N'U') IS NULL
BEGIN
    CREATE TABLE HR.PA0024
    (
        PERNR   INT          NOT NULL,
        SUBTY   VARCHAR(4)   NOT NULL CONSTRAINT DF_PA0024_SUBTY DEFAULT (''),
        OBJPS   VARCHAR(2)   NOT NULL CONSTRAINT DF_PA0024_OBJPS DEFAULT (''),
        SPRPS   CHAR(1)      NOT NULL CONSTRAINT DF_PA0024_SPRPS DEFAULT (' '),
        BEGDA   DATE         NOT NULL,
        ENDDA   DATE         NOT NULL CONSTRAINT DF_PA0024_ENDDA DEFAULT ('9999-12-31'),
        SEQNR   INT          NOT NULL CONSTRAINT DF_PA0024_SEQNR DEFAULT (1),
        QUALI   NVARCHAR(60) NOT NULL,           -- Qualification / skill
        AUSPR   INT          NULL,               -- Proficiency (0-9)
        AEDTM   DATE         NULL,
        UNAME   VARCHAR(12)  NULL,
        CONSTRAINT PK_PA0024 PRIMARY KEY (PERNR, SUBTY, OBJPS, SPRPS, ENDDA, SEQNR),
        CONSTRAINT FK_PA0024_Emp FOREIGN KEY (PERNR) REFERENCES HR.EmployeeMaster(PERNR)
    );
END
GO

/* ----------------------------------------------------------------------------
   PA2002 - Attendances (Infotype 2002). SUBTY = attendance type (AWART).
   ---------------------------------------------------------------------------- */
IF OBJECT_ID(N'HR.PA2002', N'U') IS NULL
BEGIN
    CREATE TABLE HR.PA2002
    (
        PERNR   INT           NOT NULL,
        SUBTY   VARCHAR(4)    NOT NULL,          -- Attendance type (AWART)
        OBJPS   VARCHAR(2)    NOT NULL CONSTRAINT DF_PA2002_OBJPS DEFAULT (''),
        SPRPS   CHAR(1)       NOT NULL CONSTRAINT DF_PA2002_SPRPS DEFAULT (' '),
        BEGDA   DATE          NOT NULL,
        ENDDA   DATE          NOT NULL,
        SEQNR   INT           NOT NULL CONSTRAINT DF_PA2002_SEQNR DEFAULT (1),
        AWART   VARCHAR(4)    NOT NULL,          -- Attendance type
        ABWTG   DECIMAL(7,2)  NULL,              -- Attendance days
        STDAZ   DECIMAL(7,2)  NULL,              -- Attendance hours
        BEGUZ   TIME(0)       NULL,              -- Start time
        ENDUZ   TIME(0)       NULL,              -- End time
        AEDTM   DATE          NULL,
        UNAME   VARCHAR(12)   NULL,
        CONSTRAINT PK_PA2002 PRIMARY KEY (PERNR, SUBTY, OBJPS, SPRPS, ENDDA, BEGDA, SEQNR),
        CONSTRAINT FK_PA2002_Emp FOREIGN KEY (PERNR) REFERENCES HR.EmployeeMaster(PERNR)
    );
END
GO

/* ----------------------------------------------------------------------------
   T547T - Contract type texts (customizing for IT0016).
   ---------------------------------------------------------------------------- */
IF OBJECT_ID(N'HR.T547T', N'U') IS NULL
CREATE TABLE HR.T547T
(
    CTTYP VARCHAR(2)   NOT NULL,
    CTTXT NVARCHAR(40) NULL,
    CONSTRAINT PK_T547T PRIMARY KEY (CTTYP)
);
GO

/* Validity indexes for list-type infotypes. */
IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name='IX_PA0021_Emp') CREATE INDEX IX_PA0021_Emp ON HR.PA0021(PERNR);
IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name='IX_PA0022_Emp') CREATE INDEX IX_PA0022_Emp ON HR.PA0022(PERNR);
IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name='IX_PA0023_Emp') CREATE INDEX IX_PA0023_Emp ON HR.PA0023(PERNR);
IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name='IX_PA2002_Emp') CREATE INDEX IX_PA2002_Emp ON HR.PA2002(PERNR, BEGDA, ENDDA);
GO

PRINT 'Extended infotype schema created.';
GO
