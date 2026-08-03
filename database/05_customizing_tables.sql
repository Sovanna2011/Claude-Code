/* ============================================================================
   HR Module - Customizing / Control Tables (T-tables)
   Reference: SAP ECC 6.0 EHP8 - Enterprise & Personnel structure customizing

   These are the SAP configuration tables that drive the enterprise structure
   (company code, personnel area/subarea) and personnel structure (employee
   group/subgroup), plus the various value-help / check tables.
   ============================================================================ */

USE [HRModule];
GO

/* T001  - Company Codes */
IF OBJECT_ID(N'HR.T001', N'U') IS NULL
CREATE TABLE HR.T001
(
    BUKRS VARCHAR(4)   NOT NULL,     -- Company code
    BUTXT NVARCHAR(50) NULL,         -- Name
    LAND1 VARCHAR(3)   NULL,         -- Country
    WAERS VARCHAR(5)   NULL,         -- Currency
    CONSTRAINT PK_T001 PRIMARY KEY (BUKRS)
);
GO

/* T500P - Personnel Areas */
IF OBJECT_ID(N'HR.T500P', N'U') IS NULL
CREATE TABLE HR.T500P
(
    WERKS VARCHAR(4)   NOT NULL,     -- Personnel area
    NAME1 NVARCHAR(60) NULL,         -- Name
    BUKRS VARCHAR(4)   NULL,         -- Company code
    MOLGA VARCHAR(2)   NULL,         -- Country grouping
    CONSTRAINT PK_T500P PRIMARY KEY (WERKS)
);
GO

/* T001P - Personnel Subareas */
IF OBJECT_ID(N'HR.T001P', N'U') IS NULL
CREATE TABLE HR.T001P
(
    WERKS VARCHAR(4)   NOT NULL,     -- Personnel area
    BTRTL VARCHAR(4)   NOT NULL,     -- Personnel subarea
    BTEXT NVARCHAR(30) NULL,         -- Text
    CONSTRAINT PK_T001P PRIMARY KEY (WERKS, BTRTL)
);
GO

/* T501  - Employee Group */
IF OBJECT_ID(N'HR.T501', N'U') IS NULL
CREATE TABLE HR.T501
(
    PERSG VARCHAR(1)   NOT NULL,     -- Employee group
    PTEXT NVARCHAR(30) NULL,
    CONSTRAINT PK_T501 PRIMARY KEY (PERSG)
);
GO

/* T503K - Employee Subgroup */
IF OBJECT_ID(N'HR.T503K', N'U') IS NULL
CREATE TABLE HR.T503K
(
    PERSK VARCHAR(2)   NOT NULL,     -- Employee subgroup
    PTEXT NVARCHAR(30) NULL,
    CONSTRAINT PK_T503K PRIMARY KEY (PERSK)
);
GO

/* T528T - Position texts (job/position short & long text) - simplified */
IF OBJECT_ID(N'HR.T528T', N'U') IS NULL
CREATE TABLE HR.T528T
(
    PLANS INT          NOT NULL,     -- Position
    PLSTX NVARCHAR(40) NULL,
    CONSTRAINT PK_T528T PRIMARY KEY (PLANS)
);
GO

/* T529A - Personnel action types */
IF OBJECT_ID(N'HR.T529A', N'U') IS NULL
CREATE TABLE HR.T529A
(
    MASSN VARCHAR(2)   NOT NULL,     -- Action type
    MNTXT NVARCHAR(40) NULL,         -- Text
    CONSTRAINT PK_T529A PRIMARY KEY (MASSN)
);
GO

/* T530  - Reasons for action */
IF OBJECT_ID(N'HR.T530', N'U') IS NULL
CREATE TABLE HR.T530
(
    MASSN VARCHAR(2)   NOT NULL,
    MASSG VARCHAR(2)   NOT NULL,
    MGTXT NVARCHAR(40) NULL,
    CONSTRAINT PK_T530 PRIMARY KEY (MASSN, MASSG)
);
GO

/* T554S - Absence / Attendance types */
IF OBJECT_ID(N'HR.T554S', N'U') IS NULL
CREATE TABLE HR.T554S
(
    MOABW VARCHAR(2)   NOT NULL CONSTRAINT DF_T554S_MOABW DEFAULT ('01'), -- Grouping
    AWART VARCHAR(4)   NOT NULL,     -- Absence/attendance type
    ATEXT NVARCHAR(30) NULL,
    KENNZ CHAR(1)      NULL,         -- 'A'=absence 'P'=attendance
    CONSTRAINT PK_T554S PRIMARY KEY (MOABW, AWART)
);
GO

/* T005  - Countries */
IF OBJECT_ID(N'HR.T005', N'U') IS NULL
CREATE TABLE HR.T005
(
    LAND1 VARCHAR(3)   NOT NULL,     -- Country key
    LANDX NVARCHAR(50) NULL,         -- Country name
    WAERS VARCHAR(5)   NULL,
    CONSTRAINT PK_T005 PRIMARY KEY (LAND1)
);
GO

/* T512T - Wage type texts */
IF OBJECT_ID(N'HR.T512T', N'U') IS NULL
CREATE TABLE HR.T512T
(
    LGART VARCHAR(4)   NOT NULL,     -- Wage type
    LGTXT NVARCHAR(30) NULL,
    CONSTRAINT PK_T512T PRIMARY KEY (LGART)
);
GO

/* Generic domain fixed-value table for simple keys (gender, marital status,
   form of address, communication type) - emulates SAP domain value ranges. */
IF OBJECT_ID(N'HR.DomainValue', N'U') IS NULL
CREATE TABLE HR.DomainValue
(
    Domain   VARCHAR(20)  NOT NULL,  -- GESCH, FAMST, ANRED, USRTY, STAT2
    ValueKey VARCHAR(10)  NOT NULL,
    ValueTxt NVARCHAR(40) NULL,
    CONSTRAINT PK_DomainValue PRIMARY KEY (Domain, ValueKey)
);
GO
