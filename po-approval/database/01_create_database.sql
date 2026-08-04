/* ============================================================================
   PO Approval Module - Database Creation Script
   Platform : Microsoft SQL Server
   Reference: SAP ECC 6.0 EHP8 - Materials Management / Purchasing (MM-PUR)

   This module reproduces the SAP Purchase Order (PO) release/approval data
   model. Tables follow SAP naming conventions:
     EKKO/EKPO  = Purchasing document header / item
     LFA1       = Vendor master (general)
     T16Fx      = Release strategy customizing (groups, strategies, codes)
     T0xx/T1xx  = Customizing / control tables
   Application-specific security & audit tables live in the same [PO] schema.

   Run order:
     01_create_database.sql        <- this file
     02_schema_purchasing.sql
     03_schema_release_strategy.sql
     04_schema_security.sql
     05_customizing_tables.sql
     06_seed_reference_data.sql
     07_stored_procedures.sql
     08_views.sql
   ============================================================================ */

IF DB_ID(N'POApproval') IS NULL
BEGIN
    CREATE DATABASE [POApproval];
END
GO

USE [POApproval];
GO

/* All purchasing objects live in the PO schema, mirroring SAP's MM-PUR area. */
IF SCHEMA_ID(N'PO') IS NULL
    EXEC(N'CREATE SCHEMA [PO] AUTHORIZATION [dbo];');
GO

/* ----------------------------------------------------------------------------
   Number range table - emulates SAP number range object EINKBELEG (purchasing
   documents, transaction OMH6/SNRO). Standard external PO numbers use the 45*
   range in ECC.
   ---------------------------------------------------------------------------- */
IF OBJECT_ID(N'PO.NumberRange', N'U') IS NULL
BEGIN
    CREATE TABLE PO.NumberRange
    (
        RangeObject   VARCHAR(20)  NOT NULL,   -- e.g. EBELN
        FromNumber    BIGINT       NOT NULL,
        ToNumber      BIGINT       NOT NULL,
        CurrentNumber BIGINT       NOT NULL,
        CONSTRAINT PK_NumberRange PRIMARY KEY (RangeObject)
    );
END
GO

/* ----------------------------------------------------------------------------
   Generic fixed-value / domain text table (SAP data-element domain values),
   used for language-dependent short texts such as release indicators and
   document categories.
   ---------------------------------------------------------------------------- */
IF OBJECT_ID(N'PO.DomainValue', N'U') IS NULL
BEGIN
    CREATE TABLE PO.DomainValue
    (
        Domain    VARCHAR(20)   NOT NULL,      -- e.g. FRGKE, BSTYP
        ValueKey  VARCHAR(20)   NOT NULL,
        ValueTxt  NVARCHAR(100) NULL,
        CONSTRAINT PK_DomainValue PRIMARY KEY (Domain, ValueKey)
    );
END
GO

PRINT 'PO Approval database and PO schema created.';
GO
