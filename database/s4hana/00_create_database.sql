/* ============================================================================
   S/4HANA-inspired ERP - database and schemas
   228 tables across 10 schemas

   GENERATED FILE - do not edit by hand.
   Source: docs/s4hana/table_catalogue.csv
   Regenerate: python3 tools/generate_sql_ddl.py
   ============================================================================ */

IF DB_ID(N'ErpS4') IS NULL
BEGIN
    CREATE DATABASE [ErpS4];
END
GO

ALTER DATABASE [ErpS4] SET READ_COMMITTED_SNAPSHOT ON WITH ROLLBACK IMMEDIATE;
GO

USE [ErpS4];
GO

IF SCHEMA_ID(N'org') IS NULL EXEC(N'CREATE SCHEMA [org];');
GO
IF SCHEMA_ID(N'cfg') IS NULL EXEC(N'CREATE SCHEMA [cfg];');
GO
IF SCHEMA_ID(N'mdm') IS NULL EXEC(N'CREATE SCHEMA [mdm];');
GO
IF SCHEMA_ID(N'fin') IS NULL EXEC(N'CREATE SCHEMA [fin];');
GO
IF SCHEMA_ID(N'co') IS NULL EXEC(N'CREATE SCHEMA [co];');
GO
IF SCHEMA_ID(N'wf') IS NULL EXEC(N'CREATE SCHEMA [wf];');
GO
IF SCHEMA_ID(N'sec') IS NULL EXEC(N'CREATE SCHEMA [sec];');
GO
IF SCHEMA_ID(N'audit') IS NULL EXEC(N'CREATE SCHEMA [audit];');
GO
IF SCHEMA_ID(N'rpt') IS NULL EXEC(N'CREATE SCHEMA [rpt];');
GO
IF SCHEMA_ID(N'intg') IS NULL EXEC(N'CREATE SCHEMA [intg];');
GO
