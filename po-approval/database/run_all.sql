/* ============================================================================
   PO Approval Module - Master install script
   Run with SQLCMD:
       sqlcmd -S localhost -i database/run_all.sql
   or open each file in order in SSMS. Scripts are idempotent.
   ============================================================================ */
:r 01_create_database.sql
:r 02_schema_purchasing.sql
:r 03_schema_release_strategy.sql
:r 04_schema_security.sql
:r 05_customizing_tables.sql
:r 06_seed_reference_data.sql
:r 07_stored_procedures.sql
:r 08_views.sql
GO
PRINT 'PO Approval database installation complete.';
GO
