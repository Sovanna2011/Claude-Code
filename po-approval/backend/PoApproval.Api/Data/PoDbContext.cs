using Microsoft.EntityFrameworkCore;
using PoApproval.Api.Models;

namespace PoApproval.Api.Data;

/// <summary>
/// EF Core context for the PO Approval module. All objects live in the SQL
/// schema [PO], mirroring the SAP MM-PUR application area (EKKO/EKPO, LFA1,
/// release-strategy customizing T16Fx) plus the application security tables.
/// </summary>
public class PoDbContext : DbContext
{
    public PoDbContext(DbContextOptions<PoDbContext> options) : base(options) { }

    // Purchasing documents
    public DbSet<Ekko> Ekko => Set<Ekko>();
    public DbSet<Ekpo> Ekpo => Set<Ekpo>();
    public DbSet<Lfa1> Lfa1 => Set<Lfa1>();
    public DbSet<ReleaseLog> ReleaseLog => Set<ReleaseLog>();

    // Release strategy customizing
    public DbSet<T16FG> T16FG => Set<T16FG>();
    public DbSet<T16FS> T16FS => Set<T16FS>();
    public DbSet<T16FC> T16FC => Set<T16FC>();
    public DbSet<T16FSCode> T16FSCode => Set<T16FSCode>();

    // Enterprise / purchasing customizing
    public DbSet<T001> T001 => Set<T001>();
    public DbSet<T024> T024 => Set<T024>();
    public DbSet<T024E> T024E => Set<T024E>();
    public DbSet<T161> T161 => Set<T161>();
    public DbSet<T023T> T023T => Set<T023T>();
    public DbSet<T001W> T001W => Set<T001W>();
    public DbSet<DomainValue> DomainValues => Set<DomainValue>();
    public DbSet<NumberRange> NumberRanges => Set<NumberRange>();

    // Security
    public DbSet<AppUser> AppUsers => Set<AppUser>();
    public DbSet<AppUserReleaseCode> AppUserReleaseCodes => Set<AppUserReleaseCode>();

    protected override void OnModelCreating(ModelBuilder mb)
    {
        mb.Entity<Lfa1>(e => { e.ToTable("LFA1", "PO"); e.HasKey(x => x.LIFNR); });

        mb.Entity<Ekko>(e =>
        {
            e.ToTable("EKKO", "PO");
            e.HasKey(x => x.EBELN);
            e.Property(x => x.EBELN).ValueGeneratedNever();
            e.HasOne(x => x.Vendor).WithMany().HasForeignKey(x => x.LIFNR);
            e.HasMany(x => x.Items).WithOne().HasForeignKey(x => x.EBELN);
        });

        mb.Entity<Ekpo>(e =>
        {
            e.ToTable("EKPO", "PO");
            e.HasKey(x => new { x.EBELN, x.EBELP });
        });

        mb.Entity<ReleaseLog>(e =>
        {
            e.ToTable("ReleaseLog", "PO");
            e.HasKey(x => x.LogId);
        });

        mb.Entity<T16FG>(e => { e.ToTable("T16FG", "PO"); e.HasKey(x => x.FRGGR); });
        mb.Entity<T16FS>(e => { e.ToTable("T16FS", "PO"); e.HasKey(x => new { x.FRGGR, x.FRGSX }); });
        mb.Entity<T16FC>(e => { e.ToTable("T16FC", "PO"); e.HasKey(x => new { x.FRGGR, x.FRGCO }); });
        mb.Entity<T16FSCode>(e => { e.ToTable("T16FS_Code", "PO"); e.HasKey(x => new { x.FRGGR, x.FRGSX, x.StepNo }); });

        mb.Entity<T001>(e => { e.ToTable("T001", "PO"); e.HasKey(x => x.BUKRS); });
        mb.Entity<T024>(e => { e.ToTable("T024", "PO"); e.HasKey(x => x.EKGRP); });
        mb.Entity<T024E>(e => { e.ToTable("T024E", "PO"); e.HasKey(x => x.EKORG); });
        mb.Entity<T161>(e => { e.ToTable("T161", "PO"); e.HasKey(x => new { x.BSTYP, x.BSART }); });
        mb.Entity<T023T>(e => { e.ToTable("T023T", "PO"); e.HasKey(x => x.MATKL); });
        mb.Entity<T001W>(e => { e.ToTable("T001W", "PO"); e.HasKey(x => x.WERKS); });
        mb.Entity<DomainValue>(e => { e.ToTable("DomainValue", "PO"); e.HasKey(x => new { x.Domain, x.ValueKey }); });
        mb.Entity<NumberRange>(e => { e.ToTable("NumberRange", "PO"); e.HasKey(x => x.RangeObject); });

        mb.Entity<AppUser>(e =>
        {
            e.ToTable("AppUser", "PO");
            e.HasKey(x => x.UserId);
            e.HasMany(x => x.ReleaseCodes).WithOne().HasForeignKey(x => x.UserId);
        });
        mb.Entity<AppUserReleaseCode>(e =>
        {
            e.ToTable("AppUserReleaseCode", "PO");
            e.HasKey(x => new { x.UserId, x.FRGGR, x.FRGCO });
        });

        base.OnModelCreating(mb);
    }
}
