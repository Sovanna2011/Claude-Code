/* ============================================================================
   PO Approval Module - Stored Procedures
   Reference: SAP ECC 6.0 EHP8 - emulates the release-procedure logic that in
   SAP is executed by BAPI_PO_RELEASE / the release transactions ME28/ME29N.

   These procedures document the release logic in T-SQL. The ASP.NET Core
   backend implements the same rules in C# (ReleaseService); either layer can
   drive the workflow.
   ============================================================================ */

USE [POApproval];
GO

/* ----------------------------------------------------------------------------
   usp_GetNextNumber - atomically draw the next number for a range object.
   ---------------------------------------------------------------------------- */
CREATE OR ALTER PROCEDURE PO.usp_GetNextNumber
    @RangeObject VARCHAR(20),
    @NextNumber  BIGINT OUTPUT
AS
BEGIN
    SET NOCOUNT ON;
    BEGIN TRAN;
        UPDATE PO.NumberRange
            SET @NextNumber = CurrentNumber = CurrentNumber + 1
        WHERE RangeObject = @RangeObject;

        IF @@ROWCOUNT = 0
        BEGIN
            ROLLBACK TRAN;
            THROW 50001, 'Unknown number range object.', 1;
        END
    COMMIT TRAN;
END
GO

/* ----------------------------------------------------------------------------
   usp_DetermineReleaseStrategy - recompute the net order value from the items
   and (re)determine the release group / strategy from the value band (the
   classification of CEKKO-GNETW). Resets the release status to blocked.
   ---------------------------------------------------------------------------- */
CREATE OR ALTER PROCEDURE PO.usp_DetermineReleaseStrategy
    @Ebeln VARCHAR(10)
AS
BEGIN
    SET NOCOUNT ON;

    DECLARE @Total DECIMAL(15,2) = (
        SELECT ISNULL(SUM(NETWR),0) FROM PO.EKPO WHERE EBELN=@Ebeln AND LOEKZ<>'X');

    DECLARE @Frggr VARCHAR(2), @Frgsx VARCHAR(2);
    SELECT TOP 1 @Frggr=FRGGR, @Frgsx=FRGSX
    FROM PO.T16FS
    WHERE @Total BETWEEN ValFrom AND ValTo
    ORDER BY ValFrom DESC;

    IF @Frgsx IS NULL
        /* No strategy for this value -> not subject to release. */
        UPDATE PO.EKKO
            SET RLWRT=@Total, FRGGR=NULL, FRGSX=NULL, FRGZU='', FRGKE=' ', FRGRL=' ',
                AEDAT=CAST(SYSDATETIME() AS DATE)
        WHERE EBELN=@Ebeln;
    ELSE
        UPDATE PO.EKKO
            SET RLWRT=@Total, FRGGR=@Frggr, FRGSX=@Frgsx, FRGZU='', FRGKE='B', FRGRL='X',
                AEDAT=CAST(SYSDATETIME() AS DATE)
        WHERE EBELN=@Ebeln;
END
GO

/* ----------------------------------------------------------------------------
   usp_ReleasePO - effect a release code on a PO (SAP "release" ME29N).
   Enforces sequential sign-off: only the next pending step's code is accepted.
   When the last step signs off the PO is released (FRGKE='R', FRGRL=' ').
   The API layer is responsible for checking that the user actually holds the
   code (auth object M_EINK_FRG); this procedure enforces the workflow order.
   ---------------------------------------------------------------------------- */
CREATE OR ALTER PROCEDURE PO.usp_ReleasePO
    @Ebeln VARCHAR(10),
    @Frgco VARCHAR(2),
    @Uname VARCHAR(12),
    @Note  NVARCHAR(250) = NULL
AS
BEGIN
    SET NOCOUNT ON;
    BEGIN TRY
        BEGIN TRAN;
            DECLARE @Frggr VARCHAR(2), @Frgsx VARCHAR(2), @Frgzu VARCHAR(16), @Frgke CHAR(1);
            SELECT @Frggr=FRGGR, @Frgsx=FRGSX, @Frgzu=FRGZU, @Frgke=FRGKE
            FROM PO.EKKO WHERE EBELN=@Ebeln;

            IF @Frgsx IS NULL      THROW 50010, 'PO is not subject to release.', 1;
            IF @Frgke = 'R'        THROW 50011, 'PO is already fully released.', 1;

            /* Next pending step = lowest StepNo whose code is not yet in FRGZU. */
            DECLARE @NextCode VARCHAR(2);
            SELECT TOP 1 @NextCode = FRGCO
            FROM PO.T16FS_Code
            WHERE FRGGR=@Frggr AND FRGSX=@Frgsx
              AND CHARINDEX(FRGCO, @Frgzu) = 0
            ORDER BY StepNo;

            IF @NextCode IS NULL   THROW 50012, 'No pending release step.', 1;
            IF @NextCode <> @Frgco THROW 50013, 'This release code is not the next pending step.', 1;

            SET @Frgzu = @Frgzu + @Frgco;

            /* Any remaining pending steps? */
            DECLARE @Remaining INT = (
                SELECT COUNT(*) FROM PO.T16FS_Code
                WHERE FRGGR=@Frggr AND FRGSX=@Frgsx AND CHARINDEX(FRGCO, @Frgzu)=0);

            UPDATE PO.EKKO
                SET FRGZU=@Frgzu,
                    FRGKE=CASE WHEN @Remaining=0 THEN 'R' ELSE 'B' END,
                    FRGRL=CASE WHEN @Remaining=0 THEN ' ' ELSE 'X' END,
                    AEDAT=CAST(SYSDATETIME() AS DATE)
            WHERE EBELN=@Ebeln;

            INSERT INTO PO.ReleaseLog (EBELN,FRGCO,ActionTyp,UNAME,Note)
            VALUES (@Ebeln,@Frgco,'RELEASE',@Uname,@Note);
        COMMIT TRAN;
    END TRY
    BEGIN CATCH
        IF @@TRANCOUNT > 0 ROLLBACK TRAN;
        THROW;
    END CATCH
END
GO

/* ----------------------------------------------------------------------------
   usp_RejectRelease - cancel/reject a release (SAP "cancel release").
   Rejecting at a code removes that code and every later code from FRGZU and
   re-blocks the PO, so the sign-off restarts from the rejected step.
   ---------------------------------------------------------------------------- */
CREATE OR ALTER PROCEDURE PO.usp_RejectRelease
    @Ebeln VARCHAR(10),
    @Frgco VARCHAR(2),
    @Uname VARCHAR(12),
    @Note  NVARCHAR(250) = NULL
AS
BEGIN
    SET NOCOUNT ON;
    BEGIN TRY
        BEGIN TRAN;
            DECLARE @Frggr VARCHAR(2), @Frgsx VARCHAR(2);
            SELECT @Frggr=FRGGR, @Frgsx=FRGSX FROM PO.EKKO WHERE EBELN=@Ebeln;
            IF @Frgsx IS NULL THROW 50020, 'PO is not subject to release.', 1;

            /* Keep only codes whose step precedes the rejected code's step. */
            DECLARE @Step INT = (SELECT StepNo FROM PO.T16FS_Code
                                 WHERE FRGGR=@Frggr AND FRGSX=@Frgsx AND FRGCO=@Frgco);
            IF @Step IS NULL THROW 50021, 'Release code is not part of the strategy.', 1;

            DECLARE @Frgzu VARCHAR(16) = '';
            SELECT @Frgzu = @Frgzu + FRGCO
            FROM PO.T16FS_Code
            WHERE FRGGR=@Frggr AND FRGSX=@Frgsx AND StepNo < @Step
            ORDER BY StepNo;

            UPDATE PO.EKKO
                SET FRGZU=@Frgzu, FRGKE='B', FRGRL='X', AEDAT=CAST(SYSDATETIME() AS DATE)
            WHERE EBELN=@Ebeln;

            INSERT INTO PO.ReleaseLog (EBELN,FRGCO,ActionTyp,UNAME,Note)
            VALUES (@Ebeln,@Frgco,'REJECT',@Uname,@Note);
        COMMIT TRAN;
    END TRY
    BEGIN CATCH
        IF @@TRANCOUNT > 0 ROLLBACK TRAN;
        THROW;
    END CATCH
END
GO

PRINT 'Stored procedures created.';
GO
