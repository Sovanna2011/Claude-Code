/* ============================================================================
   HR Module - Master install script
   Run with SQLCMD:
       sqlcmd -S localhost -i database/run_all.sql
   or open each file in order in SSMS. Scripts are idempotent.
   ============================================================================ */
:r 01_create_database.sql
:r 02_schema_personnel_admin.sql
:r 03_schema_org_management.sql
:r 04_schema_time_management.sql
:r 05_customizing_tables.sql
:r 06_seed_reference_data.sql
:r 07_stored_procedures.sql
:r 08_views.sql
:r 09_schema_hr_extended.sql
:r 10_seed_hr_extended.sql
:r 11_schema_security.sql
:r 12_schema_modules.sql
GO
PRINT 'HR Module database installation complete.';
GO
