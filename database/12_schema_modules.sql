/* ============================================================================
   HR Module - Additional functional modules
   Reference: SAP ECC 6.0 EHP8
     - Leave Management (ESS/MSS leave request & approval workflow over IT2001/2006)
     - Recruitment (E-Recruiting: requisitions & candidates)
     - Training & Event Management (course catalog & bookings; D = course type)
   ============================================================================ */

USE [HRModule];
GO

/* ----------------------------------------------------------------------------
   Leave Management - request/approval workflow. On approval the app writes the
   absence (IT2001) and deducts the quota (IT2006), mirroring the demo logic.
   ---------------------------------------------------------------------------- */
IF OBJECT_ID(N'HR.LeaveRequest', N'U') IS NULL
BEGIN
    CREATE TABLE HR.LeaveRequest
    (
        RequestId   INT IDENTITY(1,1) NOT NULL,
        PERNR       INT           NOT NULL,
        AWART       VARCHAR(4)    NOT NULL,        -- leave type (T554S)
        BEGDA       DATE          NOT NULL,
        ENDDA       DATE          NOT NULL,
        Days        DECIMAL(7,2)  NOT NULL,
        Status      VARCHAR(10)   NOT NULL CONSTRAINT DF_LReq_Status DEFAULT ('Pending'), -- Pending/Approved/Rejected
        Note        NVARCHAR(200) NULL,
        RequestedOn DATETIME2(0)  NOT NULL CONSTRAINT DF_LReq_On DEFAULT (SYSDATETIME()),
        DecidedBy   VARCHAR(60)   NULL,
        DecidedOn   DATETIME2(0)  NULL,
        CONSTRAINT PK_LeaveRequest PRIMARY KEY (RequestId),
        CONSTRAINT FK_LReq_Emp FOREIGN KEY (PERNR) REFERENCES HR.EmployeeMaster(PERNR),
        CONSTRAINT CK_LReq_Status CHECK (Status IN ('Pending','Approved','Rejected'))
    );
END
GO

/* ----------------------------------------------------------------------------
   Recruitment - requisitions and applicants.
   ---------------------------------------------------------------------------- */
IF OBJECT_ID(N'HR.JobRequisition', N'U') IS NULL
BEGIN
    CREATE TABLE HR.JobRequisition
    (
        ReqId     INT           NOT NULL,
        Title     NVARCHAR(80)  NOT NULL,
        ORGEH     INT           NULL,             -- hiring org unit (HRP1000 O)
        Openings  INT           NOT NULL CONSTRAINT DF_Req_Open DEFAULT (1),
        Status    VARCHAR(10)   NOT NULL CONSTRAINT DF_Req_Status DEFAULT ('Open'),
        PostedOn  DATE          NULL,
        CONSTRAINT PK_JobRequisition PRIMARY KEY (ReqId)
    );
END
GO

IF OBJECT_ID(N'HR.Applicant', N'U') IS NULL
BEGIN
    CREATE TABLE HR.Applicant
    (
        ApplicantId INT IDENTITY(1,1) NOT NULL,
        ReqId       INT           NOT NULL,
        Name        NVARCHAR(80)  NOT NULL,
        Email       NVARCHAR(120) NULL,
        Stage       VARCHAR(12)   NOT NULL CONSTRAINT DF_App_Stage DEFAULT ('Screening'), -- Screening/Interview/Offer/Hired/Rejected
        AppliedOn   DATE          NULL,
        CONSTRAINT PK_Applicant PRIMARY KEY (ApplicantId),
        CONSTRAINT FK_App_Req FOREIGN KEY (ReqId) REFERENCES HR.JobRequisition(ReqId)
    );
END
GO

/* ----------------------------------------------------------------------------
   Training & Event Management - catalog and bookings.
   ---------------------------------------------------------------------------- */
IF OBJECT_ID(N'HR.TrainingCourse', N'U') IS NULL
BEGIN
    CREATE TABLE HR.TrainingCourse
    (
        CourseId   VARCHAR(8)    NOT NULL,        -- e.g. D100 (SAP course type D)
        Title      NVARCHAR(80)  NOT NULL,
        Category   NVARCHAR(30)  NULL,
        Hours      DECIMAL(6,1)  NULL,
        CourseDate DATE          NULL,
        Seats      INT           NULL,
        CONSTRAINT PK_TrainingCourse PRIMARY KEY (CourseId)
    );
END
GO

IF OBJECT_ID(N'HR.TrainingBooking', N'U') IS NULL
BEGIN
    CREATE TABLE HR.TrainingBooking
    (
        BookingId INT IDENTITY(1,1) NOT NULL,
        PERNR     INT           NOT NULL,
        CourseId  VARCHAR(8)    NOT NULL,
        Status    VARCHAR(12)   NOT NULL CONSTRAINT DF_Book_Status DEFAULT ('Confirmed'),
        BookedOn  DATETIME2(0)  NOT NULL CONSTRAINT DF_Book_On DEFAULT (SYSDATETIME()),
        CONSTRAINT PK_TrainingBooking PRIMARY KEY (BookingId),
        CONSTRAINT UQ_Booking UNIQUE (PERNR, CourseId),
        CONSTRAINT FK_Book_Emp FOREIGN KEY (PERNR) REFERENCES HR.EmployeeMaster(PERNR),
        CONSTRAINT FK_Book_Course FOREIGN KEY (CourseId) REFERENCES HR.TrainingCourse(CourseId)
    );
END
GO

/* --- Seed ---------------------------------------------------------------- */
IF NOT EXISTS (SELECT 1 FROM HR.JobRequisition)
BEGIN
    INSERT INTO HR.JobRequisition (ReqId,Title,ORGEH,Openings,Status,PostedOn) VALUES
      (50001,'HR Business Partner',50000010,1,'Open','2026-06-01'),
      (50002,'Financial Analyst',50000020,2,'Open','2026-07-01');
    INSERT INTO HR.Applicant (ReqId,Name,Email,Stage,AppliedOn) VALUES
      (50001,'Sophie Turner','s.turner@mail.com','Interview','2026-06-10'),
      (50001,'Mark Lee','m.lee@mail.com','Screening','2026-06-15'),
      (50002,'Ana Silva','a.silva@mail.com','Offer','2026-07-08');
END
GO

IF NOT EXISTS (SELECT 1 FROM HR.TrainingCourse)
BEGIN
    INSERT INTO HR.TrainingCourse (CourseId,Title,Category,Hours,CourseDate,Seats) VALUES
      ('D100','Leadership Essentials','Leadership',16,'2026-09-15',12),
      ('D200','SAP HCM Fundamentals','Technical',24,'2026-10-06',20),
      ('D300','Business English (B2)','Language',40,'2026-11-03',15),
      ('D400','Data Privacy & GDPR','Compliance',4,'2026-09-01',50);
    INSERT INTO HR.TrainingBooking (PERNR,CourseId,Status) VALUES
      (1000,'D100','Confirmed'),(1001,'D200','Confirmed');
END
GO

IF NOT EXISTS (SELECT 1 FROM HR.LeaveRequest)
BEGIN
    INSERT INTO HR.LeaveRequest (PERNR,AWART,BEGDA,ENDDA,Days,Status,Note) VALUES
      (1001,'0100','2026-08-10','2026-08-14',5,'Pending','Summer holiday'),
      (1001,'0200','2026-05-04','2026-05-04',1,'Approved','Doctor');
END
GO

PRINT 'Additional modules (leave, recruitment, training) created.';
GO
