using Microsoft.EntityFrameworkCore;
using Microsoft.EntityFrameworkCore.Metadata.Builders;
using SugarcanePlanning.Domain.Entities;

namespace SugarcanePlanning.Infrastructure.Persistence.Configurations;

public class MaterialConfiguration : IEntityTypeConfiguration<Material>
{
    public void Configure(EntityTypeBuilder<Material> b)
    {
        b.ToTable("Materials", t =>
        {
            t.HasCheckConstraint("CK_Material_Conversion", "[UnitConversionFactor] > 0");
            t.HasCheckConstraint("CK_Material_Rates", "[MinApplicationRate] >= 0 AND [MaxApplicationRate] >= 0");
        });
        b.Property(m => m.Code).HasMaxLength(20).IsRequired();
        b.Property(m => m.Name).HasMaxLength(150).IsRequired();
        b.Property(m => m.UnitConversionFactor).HasPrecision(18, 6);
        b.Property(m => m.StandardRatePerHa).HasPrecision(18, 4);
        b.Property(m => m.MinApplicationRate).HasPrecision(18, 4);
        b.Property(m => m.MaxApplicationRate).HasPrecision(18, 4);
        b.HasIndex(m => new { m.CompanyId, m.Code }).IsUnique();
        b.HasIndex(m => m.Category);
    }
}

public class ActivityMaterialStandardConfiguration : IEntityTypeConfiguration<ActivityMaterialStandard>
{
    public void Configure(EntityTypeBuilder<ActivityMaterialStandard> b)
    {
        b.ToTable("ActivityMaterialStandards", t =>
        {
            t.HasCheckConstraint("CK_Standard_Rate", "[StandardRatePerHa] > 0");
            t.HasCheckConstraint("CK_Standard_Applications", "[NumberOfApplications] >= 1");
            t.HasCheckConstraint("CK_Standard_Waste", "[WastePercent] >= 0 AND [WastePercent] <= 100");
        });
        b.Property(s => s.StandardRatePerHa).HasPrecision(18, 4);
        b.Property(s => s.WastePercent).HasPrecision(9, 4);
        b.HasIndex(s => new { s.ActivityId, s.MaterialId, s.CropType, s.EffectiveFrom });
        b.HasOne(s => s.Activity).WithMany(a => a.MaterialStandards)
            .HasForeignKey(s => s.ActivityId).OnDelete(DeleteBehavior.Cascade);
        b.HasOne(s => s.Material).WithMany(m => m.Standards)
            .HasForeignKey(s => s.MaterialId).OnDelete(DeleteBehavior.Restrict);
        b.HasOne(s => s.CaneVariety).WithMany().HasForeignKey(s => s.CaneVarietyId).OnDelete(DeleteBehavior.Restrict);
    }
}

public class ActivityMaterialRequirementConfiguration : IEntityTypeConfiguration<ActivityMaterialRequirement>
{
    public void Configure(EntityTypeBuilder<ActivityMaterialRequirement> b)
    {
        b.ToTable("ActivityMaterialRequirements");
        b.Property(r => r.PlannedAreaHa).HasPrecision(18, 4);
        b.Property(r => r.StandardRatePerHa).HasPrecision(18, 4);
        b.Property(r => r.WastePercent).HasPrecision(9, 4);
        b.Property(r => r.BaseRequirement).HasPrecision(18, 4);
        b.Property(r => r.WasteQuantity).HasPrecision(18, 4);
        b.Property(r => r.TotalRequirement).HasPrecision(18, 4);
        b.HasIndex(r => new { r.ActivityPlanId, r.MaterialId }).IsUnique();
        b.HasIndex(r => r.RequiredDeliveryDate);
        b.HasOne(r => r.ActivityPlan).WithMany(p => p.MaterialRequirements)
            .HasForeignKey(r => r.ActivityPlanId).OnDelete(DeleteBehavior.Cascade);
        b.HasOne(r => r.Material).WithMany().HasForeignKey(r => r.MaterialId).OnDelete(DeleteBehavior.Restrict);
    }
}

public class MaterialStockConfiguration : IEntityTypeConfiguration<MaterialStock>
{
    public void Configure(EntityTypeBuilder<MaterialStock> b)
    {
        b.ToTable("MaterialStocks", t =>
            t.HasCheckConstraint("CK_Stock_Quantities", "[AvailableStock] >= 0 AND [ReservedQuantity] >= 0 AND [IncomingQuantity] >= 0"));
        b.Property(s => s.AvailableStock).HasPrecision(18, 4);
        b.Property(s => s.ReservedQuantity).HasPrecision(18, 4);
        b.Property(s => s.IncomingQuantity).HasPrecision(18, 4);
        b.Property(s => s.SourceSystem).HasMaxLength(50);
        b.HasIndex(s => new { s.MaterialId, s.EstateId }).IsUnique();
        b.HasOne(s => s.Material).WithMany(m => m.Stocks).HasForeignKey(s => s.MaterialId).OnDelete(DeleteBehavior.Cascade);
        b.HasOne(s => s.Estate).WithMany().HasForeignKey(s => s.EstateId).OnDelete(DeleteBehavior.Cascade);
    }
}

public class ActivityActualConfiguration : IEntityTypeConfiguration<ActivityActual>
{
    public void Configure(EntityTypeBuilder<ActivityActual> b)
    {
        b.ToTable("ActivityActuals", t => t.HasCheckConstraint("CK_Actual_Area", "[ActualCompletedAreaHa] >= 0"));
        b.Property(a => a.ActualCompletedAreaHa).HasPrecision(18, 4);
        b.Property(a => a.ActualWorkingHours).HasPrecision(18, 4);
        b.Property(a => a.ActualFuelLiters).HasPrecision(18, 4);
        b.Property(a => a.ActualLaborDays).HasPrecision(18, 4);
        b.Property(a => a.AreaVariance).HasPrecision(18, 4);
        b.Property(a => a.FuelVariance).HasPrecision(18, 4);
        b.Property(a => a.CompletionPercent).HasPrecision(9, 4);
        b.Property(a => a.DelayReason).HasMaxLength(500);
        b.Property(a => a.Remarks).HasMaxLength(1000);
        b.HasIndex(a => a.ActivityPlanId).IsUnique();
        b.HasOne(a => a.ActivityPlan).WithMany(p => p.Actuals)
            .HasForeignKey(a => a.ActivityPlanId).OnDelete(DeleteBehavior.Cascade);
        b.HasOne(a => a.ActualTractor).WithMany().HasForeignKey(a => a.ActualTractorId).OnDelete(DeleteBehavior.Restrict);
        b.HasOne(a => a.ActualEquipment).WithMany().HasForeignKey(a => a.ActualEquipmentId).OnDelete(DeleteBehavior.Restrict);
        b.HasOne(a => a.ActualOperator).WithMany().HasForeignKey(a => a.ActualOperatorId).OnDelete(DeleteBehavior.Restrict);
    }
}

public class ActualMaterialUsageConfiguration : IEntityTypeConfiguration<ActualMaterialUsage>
{
    public void Configure(EntityTypeBuilder<ActualMaterialUsage> b)
    {
        b.ToTable("ActualMaterialUsages");
        b.Property(u => u.PlannedQuantity).HasPrecision(18, 4);
        b.Property(u => u.ActualQuantity).HasPrecision(18, 4);
        b.Property(u => u.Variance).HasPrecision(18, 4);
        b.Property(u => u.Remarks).HasMaxLength(500);
        b.HasIndex(u => new { u.ActivityActualId, u.MaterialId }).IsUnique();
        b.HasOne(u => u.ActivityActual).WithMany(a => a.MaterialUsages)
            .HasForeignKey(u => u.ActivityActualId).OnDelete(DeleteBehavior.Cascade);
        b.HasOne(u => u.Material).WithMany().HasForeignKey(u => u.MaterialId).OnDelete(DeleteBehavior.Restrict);
    }
}
