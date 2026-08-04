/* ============================================================================
   PO Approval Module - Reporting views
   Reference: SAP ECC 6.0 EHP8 - mirrors the ME2N / ME28 release worklist and
   the header display of a purchasing document.
   ============================================================================ */

USE [POApproval];
GO

/* ----------------------------------------------------------------------------
   vw_PoWorklist - one row per PO with vendor, value, release indicator text
   and the next release code still pending. Drives the approval worklist.
   ---------------------------------------------------------------------------- */
CREATE OR ALTER VIEW PO.vw_PoWorklist
AS
    SELECT
        k.EBELN,
        k.BSART,
        k.LIFNR,
        v.NAME1                         AS VendorName,
        k.EKGRP,
        g.EKNAM                         AS PurchasingGroupName,
        k.WAERS,
        k.RLWRT                         AS NetValue,
        k.BEDAT,
        k.FRGGR,
        k.FRGSX,
        s.FRGST                         AS StrategyText,
        k.FRGZU,
        k.FRGKE,
        d.ValueTxt                      AS ReleaseIndicatorText,
        k.FRGRL,
        np.FRGCO                        AS NextPendingCode,
        nc.FRGCT                        AS NextPendingCodeText,
        k.ERNAM
    FROM PO.EKKO k
    JOIN PO.LFA1 v ON v.LIFNR = k.LIFNR
    LEFT JOIN PO.T024  g ON g.EKGRP = k.EKGRP
    LEFT JOIN PO.T16FS s ON s.FRGGR = k.FRGGR AND s.FRGSX = k.FRGSX
    LEFT JOIN PO.DomainValue d ON d.Domain='FRGKE' AND d.ValueKey = k.FRGKE
    OUTER APPLY (
        SELECT TOP 1 c.FRGCO
        FROM PO.T16FS_Code c
        WHERE c.FRGGR=k.FRGGR AND c.FRGSX=k.FRGSX
          AND CHARINDEX(c.FRGCO, k.FRGZU)=0
        ORDER BY c.StepNo) np
    LEFT JOIN PO.T16FC nc ON nc.FRGGR=k.FRGGR AND nc.FRGCO=np.FRGCO;
GO

/* ----------------------------------------------------------------------------
   vw_PoReleaseSteps - the release strategy steps of each PO, flagged as
   released or pending, with the audit user/timestamp of each effected release.
   ---------------------------------------------------------------------------- */
CREATE OR ALTER VIEW PO.vw_PoReleaseSteps
AS
    SELECT
        k.EBELN,
        c.StepNo,
        c.FRGCO,
        t.FRGCT                         AS ReleaseCodeText,
        CASE WHEN CHARINDEX(c.FRGCO, k.FRGZU) > 0 THEN CAST(1 AS BIT)
             ELSE CAST(0 AS BIT) END    AS IsReleased,
        lg.UNAME                        AS ReleasedBy,
        lg.ActedOn                      AS ReleasedOn
    FROM PO.EKKO k
    JOIN PO.T16FS_Code c ON c.FRGGR=k.FRGGR AND c.FRGSX=k.FRGSX
    LEFT JOIN PO.T16FC t ON t.FRGGR=c.FRGGR AND t.FRGCO=c.FRGCO
    OUTER APPLY (
        SELECT TOP 1 l.UNAME, l.ActedOn
        FROM PO.ReleaseLog l
        WHERE l.EBELN=k.EBELN AND l.FRGCO=c.FRGCO AND l.ActionTyp='RELEASE'
        ORDER BY l.ActedOn DESC) lg;
GO

PRINT 'Views created.';
GO
