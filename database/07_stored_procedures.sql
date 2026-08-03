/* ============================================================================
   HR Module - Stored Procedures
   Reference: SAP ECC 6.0 EHP8 - emulates HR_INFOTYPE_OPERATION style logic

   Key SAP behaviour reproduced:
     - Personnel numbers drawn from a number range (like RP_PERNR).
     - Infotype "delimiting": inserting a new time slice closes the previous
       open-ended record on the day before the new BEGDA (no overlaps), the
       classic SAP time-constraint 1 behaviour.
   ============================================================================ */

USE [HRModule];
GO

/* ----------------------------------------------------------------------------
   usp_GetNextNumber - atomically draw the next number for a range object.
   ---------------------------------------------------------------------------- */
CREATE OR ALTER PROCEDURE HR.usp_GetNextNumber
    @RangeObject VARCHAR(20),
    @NextNumber  BIGINT OUTPUT
AS
BEGIN
    SET NOCOUNT ON;
    BEGIN TRAN;
        UPDATE HR.NumberRange
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
   usp_HireEmployee - creates a new employee (SAP action "Hiring", MASSN 01)
   and the mandatory infotypes 0000/0001/0002. Returns the new PERNR.
   ---------------------------------------------------------------------------- */
CREATE OR ALTER PROCEDURE HR.usp_HireEmployee
    @HireDate DATE,
    @LastName NVARCHAR(40),
    @FirstName NVARCHAR(40),
    @Gender CHAR(1) = NULL,
    @BirthDate DATE = NULL,
    @CompanyCode VARCHAR(4) = '1000',
    @PersonnelArea VARCHAR(4) = '1000',
    @EmployeeGroup VARCHAR(1) = '1',
    @EmployeeSubgroup VARCHAR(2) = 'DU',
    @OrgUnit INT = NULL,
    @Position INT = NULL,
    @ChangedBy VARCHAR(12) = 'SYSTEM',
    @NewPernr INT OUTPUT
AS
BEGIN
    SET NOCOUNT ON;
    DECLARE @n BIGINT;
    BEGIN TRY
        BEGIN TRAN;
            EXEC HR.usp_GetNextNumber 'PERNR', @n OUTPUT;
            SET @NewPernr = CAST(@n AS INT);

            INSERT INTO HR.EmployeeMaster (PERNR, HireDate, IsActive)
            VALUES (@NewPernr, @HireDate, 1);

            INSERT INTO HR.PA0000 (PERNR,BEGDA,ENDDA,MASSN,MASSG,STAT2,AEDTM,UNAME)
            VALUES (@NewPernr,@HireDate,'9999-12-31','01','01','3',CAST(SYSDATETIME() AS DATE),@ChangedBy);

            INSERT INTO HR.PA0001 (PERNR,BEGDA,ENDDA,BUKRS,WERKS,PERSG,PERSK,ORGEH,PLANS,AEDTM,UNAME)
            VALUES (@NewPernr,@HireDate,'9999-12-31',@CompanyCode,@PersonnelArea,
                    @EmployeeGroup,@EmployeeSubgroup,@OrgUnit,@Position,
                    CAST(SYSDATETIME() AS DATE),@ChangedBy);

            INSERT INTO HR.PA0002 (PERNR,BEGDA,ENDDA,NACHN,VORNA,GBDAT,GESCH,AEDTM,UNAME)
            VALUES (@NewPernr,@HireDate,'9999-12-31',@LastName,@FirstName,@BirthDate,@Gender,
                    CAST(SYSDATETIME() AS DATE),@ChangedBy);
        COMMIT TRAN;
    END TRY
    BEGIN CATCH
        IF @@TRANCOUNT > 0 ROLLBACK TRAN;
        THROW;
    END CATCH
END
GO

/* ----------------------------------------------------------------------------
   usp_DelimitPA0001 - time-constraint 1 update of Organizational Assignment.
   Closes the current open record at @NewBegin-1 and inserts a new slice.
   Demonstrates the SAP infotype "copy/delimit" pattern for one infotype.
   ---------------------------------------------------------------------------- */
CREATE OR ALTER PROCEDURE HR.usp_DelimitPA0001
    @Pernr INT,
    @NewBegin DATE,
    @OrgUnit INT = NULL,
    @Position INT = NULL,
    @CostCenter VARCHAR(10) = NULL,
    @ChangedBy VARCHAR(12) = 'SYSTEM'
AS
BEGIN
    SET NOCOUNT ON;
    BEGIN TRY
        BEGIN TRAN;
            /* carry forward unspecified fields from the currently valid record */
            DECLARE @Bukrs VARCHAR(4), @Werks VARCHAR(4), @Persg VARCHAR(1),
                    @Persk VARCHAR(2), @OldOrg INT, @OldPlans INT, @OldKostl VARCHAR(10);
            SELECT TOP 1 @Bukrs=BUKRS,@Werks=WERKS,@Persg=PERSG,@Persk=PERSK,
                         @OldOrg=ORGEH,@OldPlans=PLANS,@OldKostl=KOSTL
            FROM HR.PA0001
            WHERE PERNR=@Pernr AND @NewBegin BETWEEN BEGDA AND ENDDA
            ORDER BY BEGDA DESC;

            /* delimit the record that is open on/after the new begin date */
            UPDATE HR.PA0001
                SET ENDDA = DATEADD(DAY,-1,@NewBegin),
                    AEDTM = CAST(SYSDATETIME() AS DATE), UNAME=@ChangedBy
            WHERE PERNR=@Pernr AND ENDDA >= @NewBegin AND BEGDA < @NewBegin;

            INSERT INTO HR.PA0001 (PERNR,BEGDA,ENDDA,BUKRS,WERKS,PERSG,PERSK,ORGEH,PLANS,KOSTL,AEDTM,UNAME)
            VALUES (@Pernr,@NewBegin,'9999-12-31',@Bukrs,@Werks,@Persg,@Persk,
                    ISNULL(@OrgUnit,@OldOrg), ISNULL(@Position,@OldPlans),
                    ISNULL(@CostCenter,@OldKostl), CAST(SYSDATETIME() AS DATE), @ChangedBy);
        COMMIT TRAN;
    END TRY
    BEGIN CATCH
        IF @@TRANCOUNT > 0 ROLLBACK TRAN;
        THROW;
    END CATCH
END
GO

/* ----------------------------------------------------------------------------
   usp_GetOrgStructure - returns the org units below a root org unit by walking
   HRP1001 A/002 relationships (org unit reports to org unit). Emulates SAP
   evaluation path O-O-S.
   ---------------------------------------------------------------------------- */
CREATE OR ALTER PROCEDURE HR.usp_GetOrgStructure
    @RootOrgId INT,
    @KeyDate DATE = NULL
AS
BEGIN
    SET NOCOUNT ON;
    IF @KeyDate IS NULL SET @KeyDate = CAST(SYSDATETIME() AS DATE);

    ;WITH OrgTree AS (
        SELECT o.OBJID, o.STEXT, o.SHORT, 0 AS Depth,
               CAST(NULL AS INT) AS ParentOrgId
        FROM HR.HRP1000 o
        WHERE o.OTYPE='O' AND o.OBJID=@RootOrgId AND o.PLVAR='01'
          AND @KeyDate BETWEEN o.BEGDA AND o.ENDDA
        UNION ALL
        SELECT c.OBJID, c.STEXT, c.SHORT, p.Depth+1, p.OBJID
        FROM OrgTree p
        JOIN HR.HRP1001 r ON r.SCLAS='O' AND r.SOBID=CAST(p.OBJID AS VARCHAR(45))
             AND r.OTYPE='O' AND r.RSIGN='A' AND r.RELAT='002' AND r.PLVAR='01'
             AND @KeyDate BETWEEN r.BEGDA AND r.ENDDA
        JOIN HR.HRP1000 c ON c.OTYPE='O' AND c.OBJID=r.OBJID AND c.PLVAR='01'
             AND @KeyDate BETWEEN c.BEGDA AND c.ENDDA
    )
    SELECT OBJID AS OrgUnitId, STEXT AS OrgUnitName, SHORT AS ShortText,
           Depth, ParentOrgId
    FROM OrgTree
    ORDER BY Depth, OrgUnitId
    OPTION (MAXRECURSION 100);
END
GO

PRINT 'Stored procedures created.';
GO
