using Microsoft.EntityFrameworkCore;
using Microsoft.EntityFrameworkCore.Metadata.Builders;
using SugarcanePlanning.Domain.Entities;

namespace SugarcanePlanning.Infrastructure.Persistence.Configurations;

public class TractorConfiguration : IEntityTypeConfiguration<Tractor>
{
    public void Configure(EntityTypeBuilder<Tractor> b)
    {
        b.ToTable("Tractors", t =>
        {
            t.HasCheckConstraint("CK_Tractor_Hp", "[Horsepower] > 0");
            t.HasCheckConstraint("CK_Tractor_Capacity", "[DailyCapacityHa] >= 0");
        });
        b.Property(t => t.AssetNo).HasMaxLength(30).IsRequired();
        b.Property(t => t.Code).HasMaxLength(20).IsRequired();
        b.Property(t => t.RegistrationNo).HasMaxLength(30);
        b.Property(t => t.Brand).HasMaxLength(60);
        b.Property(t => t.Model).HasMaxLength(60);
        b.Property(t => t.DailyCapacityHa).HasPrecision(18, 4);
        b.Property(t => t.FuelConsumptionPerHour).HasPrecision(18, 4);
        b.Property(t => t.FuelConsumptionPerHa).HasPrecision(18, 4);
        b.HasIndex(t => new { t.CompanyId, t.Code }).IsUnique();
        b.HasIndex(t => new { t.CompanyId, t.AssetNo }).IsUnique();
        b.HasIndex(t => t.Availability);
        b.HasOne(t => t.Estate).WithMany().HasForeignKey(t => t.EstateId).OnDelete(DeleteBehavior.SetNull);
        b.HasOne(t => t.CurrentFarm).WithMany().HasForeignKey(t => t.CurrentFarmId).OnDelete(DeleteBehavior.SetNull);
    }
}

public class EquipmentConfiguration : IEntityTypeConfiguration<EquipmentItem>
{
    public void Configure(EntityTypeBuilder<EquipmentItem> b)
    {
        b.ToTable("Equipment", t => t.HasCheckConstraint("CK_Equipment_Hp", "[MinimumTractorHp] >= 0"));
        b.Property(e => e.Code).HasMaxLength(20).IsRequired();
        b.Property(e => e.Name).HasMaxLength(150).IsRequired();
        b.Property(e => e.CapacityPerHour).HasPrecision(18, 4);
        b.Property(e => e.CapacityPerDay).HasPrecision(18, 4);
        b.HasIndex(e => new { e.CompanyId, e.Code }).IsUnique();
        b.HasIndex(e => e.Category);
        b.HasOne(e => e.Estate).WithMany().HasForeignKey(e => e.EstateId).OnDelete(DeleteBehavior.SetNull);
        b.HasOne(e => e.CurrentFarm).WithMany().HasForeignKey(e => e.CurrentFarmId).OnDelete(DeleteBehavior.SetNull);
    }
}

public class CompatibilityConfiguration : IEntityTypeConfiguration<TractorEquipmentCompatibility>
{
    public void Configure(EntityTypeBuilder<TractorEquipmentCompatibility> b)
    {
        b.ToTable("TractorEquipmentCompatibility");
        b.Property(c => c.Remarks).HasMaxLength(500);
        b.HasIndex(c => new { c.TractorId, c.EquipmentId }).IsUnique();
        b.HasOne(c => c.Tractor).WithMany().HasForeignKey(c => c.TractorId).OnDelete(DeleteBehavior.Cascade);
        b.HasOne(c => c.Equipment).WithMany(e => e.Compatibilities)
            .HasForeignKey(c => c.EquipmentId).OnDelete(DeleteBehavior.Cascade);
    }
}

public class OperatorConfiguration : IEntityTypeConfiguration<Operator>
{
    public void Configure(EntityTypeBuilder<Operator> b)
    {
        b.ToTable("Operators", t => t.HasCheckConstraint("CK_Operator_Hours", "[StandardHoursPerDay] > 0"));
        b.Property(o => o.Code).HasMaxLength(20).IsRequired();
        b.Property(o => o.FullName).HasMaxLength(150).IsRequired();
        b.Property(o => o.LicenseNo).HasMaxLength(40);
        b.Property(o => o.StandardHoursPerDay).HasPrecision(9, 2);
        b.HasIndex(o => new { o.CompanyId, o.Code }).IsUnique();
        b.HasOne(o => o.Farm).WithMany().HasForeignKey(o => o.FarmId).OnDelete(DeleteBehavior.SetNull);
        b.HasOne(o => o.WorkTeam).WithMany(t => t.Members).HasForeignKey(o => o.WorkTeamId).OnDelete(DeleteBehavior.SetNull);
    }
}

public class WorkTeamConfiguration : IEntityTypeConfiguration<WorkTeam>
{
    public void Configure(EntityTypeBuilder<WorkTeam> b)
    {
        b.ToTable("WorkTeams", t => t.HasCheckConstraint("CK_Team_Members", "[MemberCount] >= 0"));
        b.Property(t => t.Code).HasMaxLength(20).IsRequired();
        b.Property(t => t.Name).HasMaxLength(150).IsRequired();
        b.Property(t => t.SupervisorName).HasMaxLength(150);
        b.Property(t => t.StandardHoursPerDay).HasPrecision(9, 2);
        b.HasIndex(t => new { t.CompanyId, t.Code }).IsUnique();
        b.HasOne(t => t.Farm).WithMany().HasForeignKey(t => t.FarmId).OnDelete(DeleteBehavior.SetNull);
    }
}

public class ResourceScheduleConfiguration : IEntityTypeConfiguration<ResourceSchedule>
{
    public void Configure(EntityTypeBuilder<ResourceSchedule> b)
    {
        b.ToTable("ResourceSchedules", t =>
        {
            t.HasCheckConstraint("CK_Schedule_Time", "[PlannedEnd] > [PlannedStart]");
            t.HasCheckConstraint("CK_Schedule_Area", "[PlannedAreaHa] >= 0");
        });
        b.Property(s => s.PlannedAreaHa).HasPrecision(18, 4);
        b.Property(s => s.DailyTargetHa).HasPrecision(18, 4);
        b.Property(s => s.ExpectedWorkingHours).HasPrecision(9, 2);
        b.Property(s => s.SupervisorName).HasMaxLength(150);
        b.Property(s => s.Remarks).HasMaxLength(1000);
        b.Property(s => s.DependencyOverrideBy).HasMaxLength(100);
        b.Property(s => s.DependencyOverrideReason).HasMaxLength(500);

        // Indexes that make the double-booking checks a seek rather than a scan.
        b.HasIndex(s => new { s.TractorId, s.PlannedStart, s.PlannedEnd });
        b.HasIndex(s => new { s.EquipmentId, s.PlannedStart, s.PlannedEnd });
        b.HasIndex(s => new { s.OperatorId, s.PlannedStart, s.PlannedEnd });
        b.HasIndex(s => s.ScheduleDate);

        b.HasOne(s => s.ActivityPlan).WithMany(p => p.Schedules)
            .HasForeignKey(s => s.ActivityPlanId).OnDelete(DeleteBehavior.Cascade);
        b.HasOne(s => s.Block).WithMany().HasForeignKey(s => s.BlockId).OnDelete(DeleteBehavior.NoAction);
        b.HasOne(s => s.Activity).WithMany().HasForeignKey(s => s.ActivityId).OnDelete(DeleteBehavior.NoAction);
        b.HasOne(s => s.Tractor).WithMany(t => t.Schedules).HasForeignKey(s => s.TractorId).OnDelete(DeleteBehavior.Restrict);
        b.HasOne(s => s.Equipment).WithMany(e => e.Schedules).HasForeignKey(s => s.EquipmentId).OnDelete(DeleteBehavior.Restrict);
        b.HasOne(s => s.Operator).WithMany(o => o.Schedules).HasForeignKey(s => s.OperatorId).OnDelete(DeleteBehavior.Restrict);
        b.HasOne(s => s.WorkTeam).WithMany().HasForeignKey(s => s.WorkTeamId).OnDelete(DeleteBehavior.Restrict);
    }
}
