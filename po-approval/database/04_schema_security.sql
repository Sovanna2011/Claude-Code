/* ============================================================================
   PO Approval Module - Application security (user-level authentication)
   Reference: SAP release codes are assigned to users via authorization object
   M_EINK_FRG (fields FRGGR / FRGCO). This module reproduces that assignment in
   an application user store so the UI5 front end can authenticate users and so
   the API can authorise a release only for a code the user actually holds.

   Passwords are stored as PBKDF2-HMAC-SHA256 hashes in the format
       <iterations>.<base64(salt)>.<base64(hash)>
   (see Security/PasswordHasher.cs in the backend). No plaintext is stored.
   ============================================================================ */

USE [POApproval];
GO

/* ----------------------------------------------------------------------------
   AppUser - login accounts.
   ---------------------------------------------------------------------------- */
IF OBJECT_ID(N'PO.AppUser', N'U') IS NULL
BEGIN
    CREATE TABLE PO.AppUser
    (
        UserId      INT           IDENTITY(1,1) NOT NULL,
        UserName    VARCHAR(12)   NOT NULL,      -- SAP-style user id (UNAME), login name
        DisplayName NVARCHAR(80)  NOT NULL,      -- Full name
        Email       NVARCHAR(120) NULL,
        PwdHash     VARCHAR(210)  NOT NULL,      -- PBKDF2 encoded hash
        IsActive    BIT           NOT NULL CONSTRAINT DF_AppUser_Active DEFAULT (1),
        CreatedOn   DATETIME2(0)  NOT NULL CONSTRAINT DF_AppUser_Created DEFAULT (SYSDATETIME()),
        CONSTRAINT PK_AppUser PRIMARY KEY (UserId),
        CONSTRAINT UQ_AppUser_Name UNIQUE (UserName)
    );
END
GO

/* ----------------------------------------------------------------------------
   AppUserReleaseCode - release codes granted to a user (auth object M_EINK_FRG).
   A user with no rows here can view / create POs but cannot release them.
   ---------------------------------------------------------------------------- */
IF OBJECT_ID(N'PO.AppUserReleaseCode', N'U') IS NULL
BEGIN
    CREATE TABLE PO.AppUserReleaseCode
    (
        UserId  INT           NOT NULL,
        FRGGR   VARCHAR(2)    NOT NULL,          -- Release group
        FRGCO   VARCHAR(2)    NOT NULL,          -- Release code
        CONSTRAINT PK_AppUserReleaseCode PRIMARY KEY (UserId, FRGGR, FRGCO),
        CONSTRAINT FK_AURC_User FOREIGN KEY (UserId) REFERENCES PO.AppUser(UserId) ON DELETE CASCADE,
        CONSTRAINT FK_AURC_Code FOREIGN KEY (FRGGR, FRGCO) REFERENCES PO.T16FC(FRGGR, FRGCO)
    );
END
GO

PRINT 'Security schema (AppUser, AppUserReleaseCode) created.';
GO
