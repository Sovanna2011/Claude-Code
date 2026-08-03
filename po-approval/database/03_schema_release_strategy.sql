/* ============================================================================
   PO Approval Module - Release Strategy customizing + audit log
   Reference: SAP ECC 6.0 EHP8 - Purchasing, Release Procedure (T16Fx tables)

   SAP determines the release strategy of a PO through classification of the
   communication structure CEKKO (net order value, doc type, purchasing group,
   ...). Here the determination is simplified to a net-value band per strategy
   (ValFrom/ValTo on T16FS), which is the most common single-characteristic
   release procedure. Each strategy lists the release codes that must sign off,
   in order, in PO.T16FS_Code.
   ============================================================================ */

USE [POApproval];
GO

/* ----------------------------------------------------------------------------
   T16FG - Release Groups
   ---------------------------------------------------------------------------- */
IF OBJECT_ID(N'PO.T16FG', N'U') IS NULL
BEGIN
    CREATE TABLE PO.T16FG
    (
        FRGGR   VARCHAR(2)    NOT NULL,          -- Release group
        FRGGT   NVARCHAR(40)  NULL,              -- Description
        CLASS   VARCHAR(18)   NULL,              -- Class (classification) name
        CONSTRAINT PK_T16FG PRIMARY KEY (FRGGR)
    );
END
GO

/* ----------------------------------------------------------------------------
   T16FS - Release Strategies
   Determination band (ValFrom..ValTo) emulates the classification condition on
   the net order value CEKKO-GNETW.
   ---------------------------------------------------------------------------- */
IF OBJECT_ID(N'PO.T16FS', N'U') IS NULL
BEGIN
    CREATE TABLE PO.T16FS
    (
        FRGGR   VARCHAR(2)    NOT NULL,          -- Release group
        FRGSX   VARCHAR(2)    NOT NULL,          -- Release strategy
        FRGST   NVARCHAR(40)  NULL,              -- Strategy description
        WAERS   VARCHAR(5)    NOT NULL CONSTRAINT DF_T16FS_WAERS DEFAULT ('EUR'), -- Condition currency
        ValFrom DECIMAL(15,2) NOT NULL CONSTRAINT DF_T16FS_ValFrom DEFAULT (0),   -- Min net value (inclusive)
        ValTo   DECIMAL(15,2) NOT NULL CONSTRAINT DF_T16FS_ValTo DEFAULT (999999999999.99), -- Max net value (inclusive)
        CONSTRAINT PK_T16FS PRIMARY KEY (FRGGR, FRGSX),
        CONSTRAINT FK_T16FS_Grp FOREIGN KEY (FRGGR) REFERENCES PO.T16FG(FRGGR)
    );
END
GO

/* ----------------------------------------------------------------------------
   T16FC - Release Codes (per release group)
   ---------------------------------------------------------------------------- */
IF OBJECT_ID(N'PO.T16FC', N'U') IS NULL
BEGIN
    CREATE TABLE PO.T16FC
    (
        FRGGR   VARCHAR(2)    NOT NULL,          -- Release group
        FRGCO   VARCHAR(2)    NOT NULL,          -- Release code
        FRGCT   NVARCHAR(40)  NULL,              -- Release code description
        CONSTRAINT PK_T16FC PRIMARY KEY (FRGGR, FRGCO),
        CONSTRAINT FK_T16FC_Grp FOREIGN KEY (FRGGR) REFERENCES PO.T16FG(FRGGR)
    );
END
GO

/* ----------------------------------------------------------------------------
   T16FS_Code - Ordered release codes that make up a strategy.
   In SAP the codes and their prerequisites are held in T16FS (FRGC1..FRGC8)
   and T16FW; here they are normalised into ordered steps. StepNo defines the
   sign-off sequence (prerequisite = the previous step must be released first).
   ---------------------------------------------------------------------------- */
IF OBJECT_ID(N'PO.T16FS_Code', N'U') IS NULL
BEGIN
    CREATE TABLE PO.T16FS_Code
    (
        FRGGR   VARCHAR(2)    NOT NULL,          -- Release group
        FRGSX   VARCHAR(2)    NOT NULL,          -- Release strategy
        StepNo  INT           NOT NULL,          -- Sign-off sequence (1..n)
        FRGCO   VARCHAR(2)    NOT NULL,          -- Release code required at this step
        CONSTRAINT PK_T16FS_Code PRIMARY KEY (FRGGR, FRGSX, StepNo),
        CONSTRAINT FK_T16FSC_Strat FOREIGN KEY (FRGGR, FRGSX) REFERENCES PO.T16FS(FRGGR, FRGSX),
        CONSTRAINT FK_T16FSC_Code  FOREIGN KEY (FRGGR, FRGCO) REFERENCES PO.T16FC(FRGGR, FRGCO)
    );
END
GO

/* ----------------------------------------------------------------------------
   ReleaseLog - application audit trail of every release / reject / reset.
   (Custom Z-table; not part of standard SAP but mirrors change-document intent.)
   ---------------------------------------------------------------------------- */
IF OBJECT_ID(N'PO.ReleaseLog', N'U') IS NULL
BEGIN
    CREATE TABLE PO.ReleaseLog
    (
        LogId     BIGINT        IDENTITY(1,1) NOT NULL,
        EBELN     VARCHAR(10)   NOT NULL,        -- Purchasing document
        FRGCO     VARCHAR(2)    NULL,            -- Release code acted on
        ActionTyp VARCHAR(10)   NOT NULL,        -- RELEASE | REJECT | RESET
        UNAME     VARCHAR(12)   NOT NULL,        -- Acting user
        ActedOn   DATETIME2(0)  NOT NULL CONSTRAINT DF_RelLog_ActedOn DEFAULT (SYSDATETIME()),
        Note      NVARCHAR(250) NULL,            -- Optional reason / comment
        CONSTRAINT PK_ReleaseLog PRIMARY KEY (LogId),
        CONSTRAINT FK_RelLog_EKKO FOREIGN KEY (EBELN) REFERENCES PO.EKKO(EBELN) ON DELETE CASCADE
    );
END
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = 'IX_ReleaseLog_Ebeln')
    CREATE INDEX IX_ReleaseLog_Ebeln ON PO.ReleaseLog (EBELN, ActedOn);
GO

PRINT 'Release strategy customizing and audit log created.';
GO
