/* ============================================================================
   HR Module - Database Creation Script
   Platform : Microsoft SQL Server
   Reference: SAP ECC 6.0 EHP8 - Human Capital Management (HCM)

   This module reproduces the core HCM data model. Tables follow SAP naming
   conventions (PAnnnn = Personnel Administration infotypes, HRPnnnn =
   Organizational Management, Tnnn = Customizing / control tables).

   Run order:
     01_create_database.sql        <- this file
     02_schema_personnel_admin.sql
     03_schema_org_management.sql
     04_schema_time_management.sql
     05_customizing_tables.sql
     06_seed_reference_data.sql
     07_stored_procedures.sql
     08_views.sql
   ============================================================================ */

IF DB_ID(N'HRModule') IS NULL
BEGIN
    CREATE DATABASE [HRModule];
END
GO

USE [HRModule];
GO

/* All HCM objects live in the HR schema, mirroring SAP's HR application area. */
IF SCHEMA_ID(N'HR') IS NULL
    EXEC(N'CREATE SCHEMA [HR] AUTHORIZATION [dbo];');
GO

/* ----------------------------------------------------------------------------
   Number range table - emulates SAP number range object RP_PERNR (personnel
   numbers) and the OM object id ranges. In SAP these are maintained via SNRO.
   ---------------------------------------------------------------------------- */
IF OBJECT_ID(N'HR.NumberRange', N'U') IS NULL
BEGIN
    CREATE TABLE HR.NumberRange
    (
        RangeObject   VARCHAR(20)  NOT NULL,   -- e.g. PERNR, OBJID
        FromNumber    BIGINT       NOT NULL,
        ToNumber      BIGINT       NOT NULL,
        CurrentNumber BIGINT       NOT NULL,
        CONSTRAINT PK_NumberRange PRIMARY KEY (RangeObject)
    );
END
GO
