using Microsoft.EntityFrameworkCore;
using Microsoft.EntityFrameworkCore.Metadata.Builders;
using SugarcanePlanning.Domain.Entities;

namespace SugarcanePlanning.Infrastructure.Persistence.Configurations;

/// <summary>Column types, keys, unique constraints, check constraints and indexes for the land hierarchy.</summary>
public class CompanyConfiguration : IEntityTypeConfiguration<Company>
{
    public void Configure(EntityTypeBuilder<Company> b)
    {
        b.ToTable("Companies");
        b.Property(c => c.Code).HasMaxLength(20).IsRequired();
        b.Property(c => c.Name).HasMaxLength(150).IsRequired();
        b.Property(c => c.Address).HasMaxLength(400);
        b.Property(c => c.BaseCurrency).HasMaxLength(3).IsRequired();
        b.HasIndex(c => c.Code).IsUnique().HasFilter(null);
    }
}

public class EstateConfiguration : IEntityTypeConfiguration<Estate>
{
    public void Configure(EntityTypeBuilder<Estate> b)
    {
        b.ToTable("Estates", t => t.HasCheckConstraint("CK_Estate_Area", "[TotalAreaHa] >= 0"));
        b.Property(e => e.Code).HasMaxLength(20).IsRequired();
        b.Property(e => e.Name).HasMaxLength(150).IsRequired();
        b.Property(e => e.Location).HasMaxLength(200);
        b.Property(e => e.TotalAreaHa).HasPrecision(18, 4);
        b.HasIndex(e => new { e.CompanyId, e.Code }).IsUnique();
        b.HasOne(e => e.Company).WithMany(c => c.Estates)
            .HasForeignKey(e => e.CompanyId).OnDelete(DeleteBehavior.Restrict);
    }
}

public class FarmConfiguration : IEntityTypeConfiguration<Farm>
{
    public void Configure(EntityTypeBuilder<Farm> b)
    {
        b.ToTable("Farms", t => t.HasCheckConstraint("CK_Farm_Area", "[TotalAreaHa] >= 0"));
        b.Property(f => f.Code).HasMaxLength(20).IsRequired();
        b.Property(f => f.Name).HasMaxLength(150).IsRequired();
        b.Property(f => f.ManagerName).HasMaxLength(150);
        b.Property(f => f.TotalAreaHa).HasPrecision(18, 4);
        b.HasIndex(f => new { f.EstateId, f.Code }).IsUnique();
        b.HasIndex(f => f.CompanyId);
        b.HasOne(f => f.Estate).WithMany(e => e.Farms)
            .HasForeignKey(f => f.EstateId).OnDelete(DeleteBehavior.Restrict);
    }
}

public class ZoneConfiguration : IEntityTypeConfiguration<Zone>
{
    public void Configure(EntityTypeBuilder<Zone> b)
    {
        b.ToTable("Zones", t => t.HasCheckConstraint("CK_Zone_Area", "[TotalAreaHa] >= 0"));
        b.Property(z => z.Code).HasMaxLength(20).IsRequired();
        b.Property(z => z.Name).HasMaxLength(150).IsRequired();
        b.Property(z => z.SupervisorName).HasMaxLength(150);
        b.Property(z => z.TotalAreaHa).HasPrecision(18, 4);
        b.HasIndex(z => new { z.FarmId, z.Code }).IsUnique();
        b.HasIndex(z => z.CompanyId);
        b.HasOne(z => z.Farm).WithMany(f => f.Zones)
            .HasForeignKey(z => z.FarmId).OnDelete(DeleteBehavior.Restrict);
    }
}

public class PlantationBlockConfiguration : IEntityTypeConfiguration<PlantationBlock>
{
    public void Configure(EntityTypeBuilder<PlantationBlock> b)
    {
        b.ToTable("PlantationBlocks", t =>
        {
            t.HasCheckConstraint("CK_Block_Area", "[TotalAreaHa] >= 0 AND [PlantableAreaHa] >= 0");
            t.HasCheckConstraint("CK_Block_Plantable", "[PlantableAreaHa] <= [TotalAreaHa]");
        });
        b.Property(x => x.Code).HasMaxLength(20).IsRequired();
        b.Property(x => x.Name).HasMaxLength(150).IsRequired();
        b.Property(x => x.MapReference).HasMaxLength(100);
        b.Property(x => x.TotalAreaHa).HasPrecision(18, 4);
        b.Property(x => x.PlantableAreaHa).HasPrecision(18, 4);
        b.Property(x => x.Latitude).HasPrecision(9, 6);
        b.Property(x => x.Longitude).HasPrecision(9, 6);
        b.Ignore(x => x.HasIrrigation);
        b.HasIndex(x => new { x.ZoneId, x.Code }).IsUnique();
        b.HasIndex(x => x.CompanyId);
        b.HasOne(x => x.Zone).WithMany(z => z.Blocks)
            .HasForeignKey(x => x.ZoneId).OnDelete(DeleteBehavior.Restrict);
    }
}

public class GrowingSeasonConfiguration : IEntityTypeConfiguration<GrowingSeason>
{
    public void Configure(EntityTypeBuilder<GrowingSeason> b)
    {
        b.ToTable("GrowingSeasons", t => t.HasCheckConstraint("CK_Season_Dates", "[EndDate] >= [StartDate]"));
        b.Property(s => s.Code).HasMaxLength(20).IsRequired();
        b.Property(s => s.Name).HasMaxLength(150).IsRequired();
        b.Property(s => s.Remarks).HasMaxLength(1000);
        b.HasIndex(s => new { s.CompanyId, s.Code }).IsUnique();
        b.HasOne(s => s.Company).WithMany(c => c.Seasons)
            .HasForeignKey(s => s.CompanyId).OnDelete(DeleteBehavior.Restrict);
    }
}

public class CaneVarietyConfiguration : IEntityTypeConfiguration<CaneVariety>
{
    public void Configure(EntityTypeBuilder<CaneVariety> b)
    {
        b.ToTable("CaneVarieties", t =>
        {
            t.HasCheckConstraint("CK_Variety_Loss", "[ExpectedLossPercent] >= 0 AND [ExpectedLossPercent] <= 100");
            t.HasCheckConstraint("CK_Variety_Months", "[RecommendedPlantingStartMonth] BETWEEN 1 AND 12 AND [RecommendedPlantingEndMonth] BETWEEN 1 AND 12");
        });
        b.Property(v => v.Code).HasMaxLength(20).IsRequired();
        b.Property(v => v.Name).HasMaxLength(150).IsRequired();
        b.Property(v => v.SeedRatePerHa).HasPrecision(18, 4);
        b.Property(v => v.ExpectedYieldPerHa).HasPrecision(18, 4);
        b.Property(v => v.ExpectedLossPercent).HasPrecision(9, 4);
        b.HasIndex(v => new { v.CompanyId, v.Code }).IsUnique();
    }
}
