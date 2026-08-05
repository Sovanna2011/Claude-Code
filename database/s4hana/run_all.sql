/* ============================================================================
   S/4HANA-inspired ERP - master install script

   Run with SQLCMD from this directory:
       sqlcmd -S localhost -i run_all.sql
   or open each file in order in SSMS. Every script is idempotent.
   ============================================================================ */
:r 00_create_database.sql
:r 10_schema_org.sql
:r 20_schema_cfg.sql
:r 30_schema_mdm.sql
:r 40_schema_fin.sql
:r 50_schema_co.sql
:r 60_schema_wf.sql
:r 70_schema_sec.sql
:r 75_schema_audit.sql
:r 80_schema_rpt.sql
:r 85_schema_intg.sql
:r 90_foreign_keys.sql
:r 91_indexes.sql
:r 92_seed_dictionary.sql
GO
PRINT 'ErpS4 database installation complete.';
GO
