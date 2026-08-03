/* ============================================================================
   PO Approval Module - Enterprise & Purchasing customizing tables
   Reference: SAP ECC 6.0 EHP8
     T001   Company codes
     T024   Purchasing groups
     T024E  Purchasing organizations
     T161   Purchasing document types
   ============================================================================ */

USE [POApproval];
GO

IF OBJECT_ID(N'PO.T001', N'U') IS NULL
BEGIN
    CREATE TABLE PO.T001
    (
        BUKRS   VARCHAR(4)    NOT NULL,          -- Company code
        BUTXT   NVARCHAR(40)  NULL,              -- Company name
        LAND1   VARCHAR(3)    NULL,              -- Country key
        WAERS   VARCHAR(5)    NULL,              -- Currency
        CONSTRAINT PK_T001 PRIMARY KEY (BUKRS)
    );
END
GO

IF OBJECT_ID(N'PO.T024', N'U') IS NULL
BEGIN
    CREATE TABLE PO.T024
    (
        EKGRP   VARCHAR(3)    NOT NULL,          -- Purchasing group
        EKNAM   NVARCHAR(18)  NULL,              -- Description
        EKTEL   VARCHAR(20)   NULL,              -- Telephone
        CONSTRAINT PK_T024 PRIMARY KEY (EKGRP)
    );
END
GO

IF OBJECT_ID(N'PO.T024E', N'U') IS NULL
BEGIN
    CREATE TABLE PO.T024E
    (
        EKORG   VARCHAR(4)    NOT NULL,          -- Purchasing organization
        EKOTX   NVARCHAR(20)  NULL,              -- Description
        BUKRS   VARCHAR(4)    NULL,              -- Assigned company code
        CONSTRAINT PK_T024E PRIMARY KEY (EKORG)
    );
END
GO

IF OBJECT_ID(N'PO.T161', N'U') IS NULL
BEGIN
    CREATE TABLE PO.T161
    (
        BSTYP   CHAR(1)       NOT NULL,          -- Document category
        BSART   VARCHAR(4)    NOT NULL,          -- Document type
        BATXT   NVARCHAR(20)  NULL,              -- Description
        CONSTRAINT PK_T161 PRIMARY KEY (BSTYP, BSART)
    );
END
GO

PRINT 'Customizing tables (T001, T024, T024E, T161) created.';
GO
