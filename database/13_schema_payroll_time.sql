/* ============================================================================
   HR Module - Payroll (PY) & Overtime / Public Holidays
   Reference: SAP ECC 6.0 EHP8 - Payroll and Time Management

     HR.PublicHoliday   Holiday calendar (drives OT categorisation)
     HR.OvertimeRecord  Recorded overtime hours per employee/day
     HR.PayrollResult   Payroll run results (RT-style: gross/net per period)
     HR.PayrollConfig   Rates & OT multipliers (customizing)

   Overtime is categorised by date: public holiday > Sunday > normal, each with
   its own pay multiplier - reproduced by the app/report logic.
   ============================================================================ */

USE [HRModule];
GO

IF OBJECT_ID(N'HR.PublicHoliday', N'U') IS NULL
BEGIN
    CREATE TABLE HR.PublicHoliday
    (
        HolDate  DATE          NOT NULL,
        HolName  NVARCHAR(60)  NULL,
        MOLGA    VARCHAR(2)    NOT NULL CONSTRAINT DF_Hol_Molga DEFAULT ('01'), -- country grouping
        CONSTRAINT PK_PublicHoliday PRIMARY KEY (MOLGA, HolDate)
    );
END
GO

IF OBJECT_ID(N'HR.OvertimeRecord', N'U') IS NULL
BEGIN
    CREATE TABLE HR.OvertimeRecord
    (
        OtId     INT IDENTITY(1,1) NOT NULL,
        PERNR    INT           NOT NULL,
        OtDate   DATE          NOT NULL,
        Hours    DECIMAL(6,2)  NOT NULL,
        Category AS (           -- computed: Holiday / Sunday / Normal
            CASE WHEN DATENAME(WEEKDAY, OtDate) = 'Sunday' THEN 'Sunday' ELSE 'Normal' END) PERSISTED,
        CreatedOn DATETIME2(0) NOT NULL CONSTRAINT DF_Ot_On DEFAULT (SYSDATETIME()),
        CONSTRAINT PK_OvertimeRecord PRIMARY KEY (OtId),
        CONSTRAINT FK_Ot_Emp FOREIGN KEY (PERNR) REFERENCES HR.EmployeeMaster(PERNR)
    );
    /* Note: public-holiday overrides Sunday/Normal; the app resolves the final
       category against HR.PublicHoliday (a computed column can't join a table). */
END
GO

IF OBJECT_ID(N'HR.PayrollConfig', N'U') IS NULL
BEGIN
    CREATE TABLE HR.PayrollConfig
    (
        ConfigKey VARCHAR(30)  NOT NULL,
        NumValue  DECIMAL(9,4) NOT NULL,
        CONSTRAINT PK_PayrollConfig PRIMARY KEY (ConfigKey)
    );
END
GO

IF OBJECT_ID(N'HR.PayrollResult', N'U') IS NULL
BEGIN
    CREATE TABLE HR.PayrollResult
    (
        ResultId   INT IDENTITY(1,1) NOT NULL,
        PERNR      INT           NOT NULL,
        PeriodFrom DATE          NOT NULL,
        PeriodTo   DATE          NOT NULL,
        BaseAmount DECIMAL(15,2) NOT NULL,
        OtAmount   DECIMAL(15,2) NOT NULL CONSTRAINT DF_PR_Ot DEFAULT (0),
        Gross      DECIMAL(15,2) NOT NULL,
        Tax        DECIMAL(15,2) NOT NULL CONSTRAINT DF_PR_Tax DEFAULT (0),
        Social     DECIMAL(15,2) NOT NULL CONSTRAINT DF_PR_Soc DEFAULT (0),
        Net        DECIMAL(15,2) NOT NULL,
        Currency   VARCHAR(5)    NOT NULL CONSTRAINT DF_PR_Cur DEFAULT ('EUR'),
        RunOn      DATETIME2(0)  NOT NULL CONSTRAINT DF_PR_On DEFAULT (SYSDATETIME()),
        CONSTRAINT PK_PayrollResult PRIMARY KEY (ResultId),
        CONSTRAINT FK_PR_Emp FOREIGN KEY (PERNR) REFERENCES HR.EmployeeMaster(PERNR)
    );
END
GO

/* --- Seed ---------------------------------------------------------------- */
MERGE HR.PayrollConfig AS t
USING (VALUES
    ('TAX_RATE',0.15),('SOCIAL_RATE',0.09),
    ('OT_MULT_NORMAL',1.25),('OT_MULT_SUNDAY',1.5),('OT_MULT_HOLIDAY',2.0),
    ('MONTHLY_FACTOR',4.33)
) AS s(ConfigKey,NumValue) ON t.ConfigKey=s.ConfigKey
WHEN NOT MATCHED THEN INSERT(ConfigKey,NumValue) VALUES(s.ConfigKey,s.NumValue);
GO

MERGE HR.PublicHoliday AS t
USING (VALUES
    ('2026-01-01','New Year'),('2026-04-03','Good Friday'),('2026-05-01','Labour Day'),
    ('2026-10-03','Day of German Unity'),('2026-12-25','Christmas Day'),('2026-12-26','Boxing Day')
) AS s(HolDate,HolName) ON t.MOLGA='01' AND t.HolDate=s.HolDate
WHEN NOT MATCHED THEN INSERT(HolDate,HolName) VALUES(s.HolDate,s.HolName);
GO

IF NOT EXISTS (SELECT 1 FROM HR.OvertimeRecord)
BEGIN
    INSERT INTO HR.OvertimeRecord (PERNR,OtDate,Hours) VALUES
      (1000,'2026-06-02',2.0),(1000,'2026-06-07',4.0),(1000,'2026-05-01',3.0),
      (1001,'2026-06-03',1.5),(1001,'2026-06-14',3.0),(1001,'2026-12-25',2.0);
END
GO

PRINT 'Payroll & overtime schema created.';
GO
