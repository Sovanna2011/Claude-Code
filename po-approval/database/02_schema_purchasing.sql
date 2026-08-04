/* ============================================================================
   PO Approval Module - Purchasing documents (EKKO / EKPO) and Vendor (LFA1)
   Reference: SAP ECC 6.0 EHP8 - Materials Management, Purchasing (MM-PUR)

   EKKO carries the release-strategy control fields that drive the approval
   workflow:
     FRGGR  Release group      (T16FG)
     FRGSX  Release strategy   (T16FS)
     FRGZU  Release status - the codes that have already been effected
     FRGKE  Release indicator  (' ' not subject, 'B' blocked, 'R' released)
     FRGRL  Release incomplete flag ('X' = not yet fully released)
   ============================================================================ */

USE [POApproval];
GO

/* ----------------------------------------------------------------------------
   LFA1 - Vendor master (general section)
   ---------------------------------------------------------------------------- */
IF OBJECT_ID(N'PO.LFA1', N'U') IS NULL
BEGIN
    CREATE TABLE PO.LFA1
    (
        LIFNR   VARCHAR(10)   NOT NULL,          -- Vendor account number
        NAME1   NVARCHAR(70)  NOT NULL,          -- Name
        ORT01   NVARCHAR(40)  NULL,              -- City
        PSTLZ   VARCHAR(10)   NULL,              -- Postal code
        LAND1   VARCHAR(3)    NULL,              -- Country key
        STCEG   VARCHAR(20)   NULL,              -- VAT registration number
        WAERS   VARCHAR(5)    NULL,              -- Default order currency
        CONSTRAINT PK_LFA1 PRIMARY KEY (LIFNR)
    );
END
GO

/* ----------------------------------------------------------------------------
   EKKO - Purchasing Document Header
   ---------------------------------------------------------------------------- */
IF OBJECT_ID(N'PO.EKKO', N'U') IS NULL
BEGIN
    CREATE TABLE PO.EKKO
    (
        EBELN   VARCHAR(10)   NOT NULL,          -- Purchasing document number
        BSTYP   CHAR(1)       NOT NULL CONSTRAINT DF_EKKO_BSTYP DEFAULT ('F'),   -- Category ('F' = PO)
        BSART   VARCHAR(4)    NOT NULL,          -- Document type (T161)
        LIFNR   VARCHAR(10)   NOT NULL,          -- Vendor (LFA1)
        EKORG   VARCHAR(4)    NOT NULL,          -- Purchasing organization (T024E)
        EKGRP   VARCHAR(3)    NOT NULL,          -- Purchasing group (T024)
        BUKRS   VARCHAR(4)    NOT NULL,          -- Company code (T001)
        WAERS   VARCHAR(5)    NOT NULL,          -- Document currency
        BEDAT   DATE          NOT NULL,          -- Document date
        RLWRT   DECIMAL(15,2) NOT NULL CONSTRAINT DF_EKKO_RLWRT DEFAULT (0),  -- Total net order value (CEKKO-GNETW)
        /* --- Release strategy control fields --- */
        FRGGR   VARCHAR(2)    NULL,              -- Release group   (T16FG)
        FRGSX   VARCHAR(2)    NULL,              -- Release strategy (T16FS)
        FRGZU   VARCHAR(16)   NOT NULL CONSTRAINT DF_EKKO_FRGZU DEFAULT (''),  -- Release status (codes effected)
        FRGKE   CHAR(1)       NOT NULL CONSTRAINT DF_EKKO_FRGKE DEFAULT (' '), -- Release indicator
        FRGRL   CHAR(1)       NOT NULL CONSTRAINT DF_EKKO_FRGRL DEFAULT (' '), -- Release incomplete ('X')
        /* --- Administrative --- */
        ERNAM   VARCHAR(12)   NULL,              -- Created by
        AEDAT   DATE          NULL,              -- Last changed on
        MEMORY  CHAR(1)       NOT NULL CONSTRAINT DF_EKKO_MEMORY DEFAULT (' '), -- Held/incomplete ('X')
        CONSTRAINT PK_EKKO PRIMARY KEY (EBELN),
        CONSTRAINT FK_EKKO_LFA1 FOREIGN KEY (LIFNR) REFERENCES PO.LFA1(LIFNR)
    );
END
GO

/* ----------------------------------------------------------------------------
   EKPO - Purchasing Document Item
   ---------------------------------------------------------------------------- */
IF OBJECT_ID(N'PO.EKPO', N'U') IS NULL
BEGIN
    CREATE TABLE PO.EKPO
    (
        EBELN   VARCHAR(10)   NOT NULL,          -- Purchasing document number
        EBELP   INT           NOT NULL,          -- Item number (10,20,30,...)
        TXZ01   NVARCHAR(40)  NOT NULL,          -- Short text
        MATNR   VARCHAR(18)   NULL,              -- Material number
        MATKL   VARCHAR(9)    NULL,              -- Material group
        WERKS   VARCHAR(4)    NULL,              -- Plant
        MENGE   DECIMAL(13,3) NOT NULL,          -- Quantity
        MEINS   VARCHAR(3)    NOT NULL CONSTRAINT DF_EKPO_MEINS DEFAULT ('EA'), -- Order unit
        NETPR   DECIMAL(15,2) NOT NULL,          -- Net price
        PEINH   DECIMAL(9,0)  NOT NULL CONSTRAINT DF_EKPO_PEINH DEFAULT (1),    -- Price unit
        NETWR   DECIMAL(15,2) NOT NULL,          -- Net value (MENGE/PEINH * NETPR)
        BRTWR   DECIMAL(15,2) NULL,              -- Gross value (incl. tax)
        MWSKZ   VARCHAR(2)    NULL,              -- Tax code (T007A)
        LGORT   VARCHAR(4)    NULL,              -- Storage location
        EINDT   DATE          NULL,              -- Item delivery date
        LOEKZ   CHAR(1)       NOT NULL CONSTRAINT DF_EKPO_LOEKZ DEFAULT (' '),  -- Deletion indicator
        CONSTRAINT PK_EKPO PRIMARY KEY (EBELN, EBELP),
        CONSTRAINT FK_EKPO_EKKO FOREIGN KEY (EBELN) REFERENCES PO.EKKO(EBELN) ON DELETE CASCADE
    );
END
GO

/* Add the extended item-detail columns to a pre-existing EKPO (idempotent). */
IF COL_LENGTH('PO.EKPO','BRTWR') IS NULL ALTER TABLE PO.EKPO ADD BRTWR DECIMAL(15,2) NULL;
IF COL_LENGTH('PO.EKPO','MWSKZ') IS NULL ALTER TABLE PO.EKPO ADD MWSKZ VARCHAR(2)   NULL;
IF COL_LENGTH('PO.EKPO','LGORT') IS NULL ALTER TABLE PO.EKPO ADD LGORT VARCHAR(4)   NULL;
IF COL_LENGTH('PO.EKPO','EINDT') IS NULL ALTER TABLE PO.EKPO ADD EINDT DATE         NULL;
GO

/* Indexes to accelerate worklist queries. */
IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = 'IX_EKKO_Release')
    CREATE INDEX IX_EKKO_Release ON PO.EKKO (FRGKE, FRGGR, FRGSX);
IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = 'IX_EKKO_Vendor')
    CREATE INDEX IX_EKKO_Vendor ON PO.EKKO (LIFNR);
GO

PRINT 'Purchasing schema (LFA1, EKKO, EKPO) created.';
GO
