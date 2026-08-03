using HRModule.Api.Models;
using HRModule.Api.Models.Customizing;
using HRModule.Api.Models.Infotypes;
using HRModule.Api.Models.OrgManagement;
using Microsoft.EntityFrameworkCore;

namespace HRModule.Api.Data;

/// <summary>
/// EF Core context mapping the HR module tables. All objects live in the SQL
/// schema [HR], mirroring the SAP HR application area. Infotypes use the
/// composite SAP key (PERNR, SUBTY, OBJPS, SPRPS, ENDDA, SEQNR).
/// </summary>
public class HRDbContext : DbContext
{
    public HRDbContext(DbContextOptions<HRDbContext> options) : base(options) { }

    // Master & Personnel Administration
    public DbSet<EmployeeMaster> Employees => Set<EmployeeMaster>();
    public DbSet<PA0000> PA0000 => Set<PA0000>();
    public DbSet<PA0001> PA0001 => Set<PA0001>();
    public DbSet<PA0002> PA0002 => Set<PA0002>();
    public DbSet<PA0006> PA0006 => Set<PA0006>();
    public DbSet<PA0007> PA0007 => Set<PA0007>();
    public DbSet<PA0008> PA0008 => Set<PA0008>();
    public DbSet<PA0008WageType> PA0008WageTypes => Set<PA0008WageType>();
    public DbSet<PA0009> PA0009 => Set<PA0009>();
    public DbSet<PA0105> PA0105 => Set<PA0105>();
    public DbSet<PA2001> PA2001 => Set<PA2001>();
    public DbSet<PA2006> PA2006 => Set<PA2006>();
    public DbSet<PA0016> PA0016 => Set<PA0016>();
    public DbSet<PA0019> PA0019 => Set<PA0019>();
    public DbSet<PA0021> PA0021 => Set<PA0021>();
    public DbSet<PA0022> PA0022 => Set<PA0022>();
    public DbSet<PA0023> PA0023 => Set<PA0023>();
    public DbSet<PA0024> PA0024 => Set<PA0024>();
    public DbSet<PA2002> PA2002 => Set<PA2002>();

    // Organizational Management
    public DbSet<HRP1000> HRP1000 => Set<HRP1000>();
    public DbSet<HRP1001> HRP1001 => Set<HRP1001>();

    // Customizing
    public DbSet<T001> T001 => Set<T001>();
    public DbSet<T500P> T500P => Set<T500P>();
    public DbSet<T001P> T001P => Set<T001P>();
    public DbSet<T501> T501 => Set<T501>();
    public DbSet<T503K> T503K => Set<T503K>();
    public DbSet<T528T> T528T => Set<T528T>();
    public DbSet<T529A> T529A => Set<T529A>();
    public DbSet<T530> T530 => Set<T530>();
    public DbSet<T554S> T554S => Set<T554S>();
    public DbSet<T005> T005 => Set<T005>();
    public DbSet<T512T> T512T => Set<T512T>();
    public DbSet<T547T> T547T => Set<T547T>();
    public DbSet<DomainValue> DomainValues => Set<DomainValue>();
    public DbSet<NumberRange> NumberRanges => Set<NumberRange>();

    protected override void OnModelCreating(ModelBuilder mb)
    {
        // ---- Master ----------------------------------------------------
        mb.Entity<EmployeeMaster>(e =>
        {
            e.ToTable("EmployeeMaster", "HR");
            e.HasKey(x => x.PERNR);
            e.Property(x => x.PERNR).ValueGeneratedNever();
        });

        // ---- Infotypes (shared composite key configuration) ------------
        ConfigureInfotype<PA0000>(mb, "PA0000");
        ConfigureInfotype<PA0001>(mb, "PA0001");
        ConfigureInfotype<PA0002>(mb, "PA0002");
        ConfigureInfotype<PA0006>(mb, "PA0006");
        ConfigureInfotype<PA0007>(mb, "PA0007");
        ConfigureInfotype<PA0008>(mb, "PA0008");
        ConfigureInfotype<PA0009>(mb, "PA0009");
        ConfigureInfotype<PA0105>(mb, "PA0105");
        ConfigureInfotype<PA2001>(mb, "PA2001");
        ConfigureInfotype<PA2006>(mb, "PA2006");
        ConfigureInfotype<PA0016>(mb, "PA0016");
        ConfigureInfotype<PA0019>(mb, "PA0019");
        ConfigureInfotype<PA0021>(mb, "PA0021");
        ConfigureInfotype<PA0022>(mb, "PA0022");
        ConfigureInfotype<PA0023>(mb, "PA0023");
        ConfigureInfotype<PA0024>(mb, "PA0024");

        // PA2002 (Attendances): BEGDA is part of the key (time segments).
        mb.Entity<PA2002>(e =>
        {
            e.ToTable("PA2002", "HR");
            e.HasKey(x => new { x.PERNR, x.SUBTY, x.OBJPS, x.SPRPS, x.ENDDA, x.BEGDA, x.SEQNR });
            e.Property(x => x.PERNR).ValueGeneratedNever();
        });

        mb.Entity<PA0008WageType>(e =>
        {
            e.ToTable("PA0008_WageType", "HR");
            e.HasKey(x => new { x.PERNR, x.ENDDA, x.SEQNR, x.LineNo });
        });

        // Basic Pay -> wage type lines relationship (natural key)
        mb.Entity<PA0008>()
          .HasMany(x => x.WageTypes)
          .WithOne()
          .HasForeignKey(w => new { w.PERNR, w.ENDDA, w.SEQNR })
          .HasPrincipalKey(p => new { p.PERNR, p.ENDDA, p.SEQNR });

        // ---- Organizational Management ---------------------------------
        mb.Entity<HRP1000>(e =>
        {
            e.ToTable("HRP1000", "HR");
            e.HasKey(x => new { x.PLVAR, x.OTYPE, x.OBJID, x.ISTAT, x.ENDDA, x.BEGDA, x.SEQNR });
        });
        mb.Entity<HRP1001>(e =>
        {
            e.ToTable("HRP1001", "HR");
            e.HasKey(x => new { x.PLVAR, x.OTYPE, x.OBJID, x.ISTAT, x.ENDDA, x.BEGDA,
                                x.RSIGN, x.RELAT, x.SCLAS, x.SOBID, x.SEQNR });
        });

        // ---- Customizing -----------------------------------------------
        mb.Entity<T001>(e => { e.ToTable("T001", "HR"); e.HasKey(x => x.BUKRS); });
        mb.Entity<T500P>(e => { e.ToTable("T500P", "HR"); e.HasKey(x => x.WERKS); });
        mb.Entity<T001P>(e => { e.ToTable("T001P", "HR"); e.HasKey(x => new { x.WERKS, x.BTRTL }); });
        mb.Entity<T501>(e => { e.ToTable("T501", "HR"); e.HasKey(x => x.PERSG); });
        mb.Entity<T503K>(e => { e.ToTable("T503K", "HR"); e.HasKey(x => x.PERSK); });
        mb.Entity<T528T>(e => { e.ToTable("T528T", "HR"); e.HasKey(x => x.PLANS); e.Property(x => x.PLANS).ValueGeneratedNever(); });
        mb.Entity<T529A>(e => { e.ToTable("T529A", "HR"); e.HasKey(x => x.MASSN); });
        mb.Entity<T530>(e => { e.ToTable("T530", "HR"); e.HasKey(x => new { x.MASSN, x.MASSG }); });
        mb.Entity<T554S>(e => { e.ToTable("T554S", "HR"); e.HasKey(x => new { x.MOABW, x.AWART }); });
        mb.Entity<T005>(e => { e.ToTable("T005", "HR"); e.HasKey(x => x.LAND1); });
        mb.Entity<T512T>(e => { e.ToTable("T512T", "HR"); e.HasKey(x => x.LGART); });
        mb.Entity<T547T>(e => { e.ToTable("T547T", "HR"); e.HasKey(x => x.CTTYP); });
        mb.Entity<DomainValue>(e => { e.ToTable("DomainValue", "HR"); e.HasKey(x => new { x.Domain, x.ValueKey }); });
        mb.Entity<NumberRange>(e => { e.ToTable("NumberRange", "HR"); e.HasKey(x => x.RangeObject); });

        base.OnModelCreating(mb);
    }

    /// <summary>Applies the shared SAP infotype key + table mapping.</summary>
    private static void ConfigureInfotype<T>(ModelBuilder mb, string table) where T : InfotypeBase
    {
        mb.Entity<T>(e =>
        {
            e.ToTable(table, "HR");
            e.HasKey(x => new { x.PERNR, x.SUBTY, x.OBJPS, x.SPRPS, x.ENDDA, x.SEQNR });
            e.Property(x => x.PERNR).ValueGeneratedNever();
        });
    }
}
