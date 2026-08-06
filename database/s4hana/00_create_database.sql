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

/* Raise the compatibility level to the highest the server supports, up to 170
   (SQL Server 2025). A restored or newly created database can inherit an older
   level from model, and the level - not the product version - is what decides
   which cardinality estimator and which optimiser features apply. The schema
   itself needs nothing newer than 2012, so this is about how well it runs, not
   whether it runs. */
DECLARE @Level int = (SELECT MIN(level) FROM (VALUES
    (170), (CAST(SERVERPROPERTY('ProductMajorVersion') AS int) * 10)) AS l(level));

IF @Level > (SELECT compatibility_level FROM sys.databases WHERE name = N'ErpS4')
BEGIN
    DECLARE @Sql nvarchar(200) =
        N'ALTER DATABASE [ErpS4] SET COMPATIBILITY_LEVEL = ' + CAST(@Level AS nvarchar(3)) + N';';
    EXEC sp_executesql @Sql;
END
GO

/* Snapshot statistics and the default cardinality estimator: this schema is
   read far more than it is written, and a stale estimate on the universal
   journal is the difference between a seek and a 4 000 000 row scan. */
ALTER DATABASE [ErpS4] SET AUTO_UPDATE_STATISTICS ON;
ALTER DATABASE [ErpS4] SET AUTO_UPDATE_STATISTICS_ASYNC ON;
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
