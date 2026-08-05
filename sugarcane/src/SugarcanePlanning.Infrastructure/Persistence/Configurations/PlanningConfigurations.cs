using Microsoft.EntityFrameworkCore;
using Microsoft.EntityFrameworkCore.Metadata.Builders;
using SugarcanePlanning.Domain.Entities;

namespace SugarcanePlanning.Infrastructure.Persistence.Configurations;

public class PlantingActivityConfiguration : IEntityTypeConfiguration<PlantingActivity>
{
    public void Configure(EntityTypeBuilder<PlantingActivity> b)
    {
        b.ToTable("PlantingActivities", t => t.HasCheckConstraint("CK_Activity_Sequence", "[SequenceNo] > 0"));
        b.Property(a => a.Code).HasMaxLength(20).IsRequired();
        b.Property(a => a.Name).HasMaxLength(150).IsRequired();
        b.Property(a => a.StandardCapacityPerHour).HasPrecision(18, 4);
        b.Property(a => a.StandardCapacityPerDay).HasPrecision(18, 4);
        b.Property(a => a.StandardDurationPerHa).HasPrecision(18, 4);
        b.Property(a => a.StandardLaborDaysPerHa).HasPrecision(18, 4);
        b.HasIndex(a => new { a.CompanyId, a.Code }).IsUnique();
        b.HasIndex(a => a.SequenceNo);
    }
}

public class ActivityDependencyConfiguration : IEntityTypeConfiguration<ActivityDependency>
{
    public void Configure(EntityTypeBuilder<ActivityDependency> b)
    {
        b.ToTable("ActivityDependencies", t =>
            t.HasCheckConstraint("CK_Dependency_NotSelf", "[ActivityId] <> [PredecessorActivityId]"));
        b.Property(d => d.Remarks).HasMaxLength(500);
        b.HasIndex(d => new { d.ActivityId, d.PredecessorActivityId }).IsUnique();
        b.HasOne(d => d.Activity).WithMany(a => a.Predecessors)
            .HasForeignKey(d => d.ActivityId).OnDelete(DeleteBehavior.Cascade);
        b.HasOne(d => d.PredecessorActivity).WithMany(a => a.Successors)
            .HasForeignKey(d => d.PredecessorActivityId).OnDelete(DeleteBehavior.Restrict);
    }
}

public class PlantingProjectionConfiguration : IEntityTypeConfiguration<PlantingProjection>
{
    public void Configure(EntityTypeBuilder<PlantingProjection> b)
    {
        b.ToTable("PlantingProjections", t =>
        {
            t.HasCheckConstraint("CK_Projection_Version", "[Version] > 0");
            t.HasCheckConstraint("CK_Projection_Dates", "[PlanningEndDate] >= [PlanningStartDate]");
        });
        b.Property(p => p.ProjectionNo).HasMaxLength(30).IsRequired();
        b.Property(p => p.TotalProjectedAreaHa).HasPrecision(18, 4);
        b.Property(p => p.TotalExpectedProductionTons).HasPrecision(18, 4);
        b.Property(p => p.PreparedBy).HasMaxLength(100);
        b.Property(p => p.SubmittedBy).HasMaxLength(100);
        b.Property(p => p.ReviewedBy).HasMaxLength(100);
        b.Property(p => p.ApprovedBy).HasMaxLength(100);
        b.Property(p => p.RejectedBy).HasMaxLength(100);
        b.Property(p => p.RejectionReason).HasMaxLength(1000);
        b.Property(p => p.RevisionReason).HasMaxLength(1000);
        b.Property(p => p.Remarks).HasMaxLength(1000);
        b.Ignore(p => p.IsEditable);

        // One version number per projection number and estate.
        b.HasIndex(p => new { p.EstateId, p.ProjectionNo, p.Version }).IsUnique();
        b.HasIndex(p => new { p.GrowingSeasonId, p.Status });

        b.HasOne(p => p.Estate).WithMany().HasForeignKey(p => p.EstateId).OnDelete(DeleteBehavior.Restrict);
        b.HasOne(p => p.GrowingSeason).WithMany().HasForeignKey(p => p.GrowingSeasonId).OnDelete(DeleteBehavior.Restrict);
        b.HasOne(p => p.RevisedFrom).WithMany().HasForeignKey(p => p.RevisedFromProjectionId).OnDelete(DeleteBehavior.Restrict);
    }
}

public class ProjectionLineConfiguration : IEntityTypeConfiguration<ProjectionLine>
{
    public void Configure(EntityTypeBuilder<ProjectionLine> b)
    {
        b.ToTable("ProjectionLines", t =>
        {
            t.HasCheckConstraint("CK_Line_Area", "[ProjectedPlantingAreaHa] > 0");
            t.HasCheckConstraint("CK_Line_Loss", "[ExpectedLossPercent] >= 0 AND [ExpectedLossPercent] <= 100");
            t.HasCheckConstraint("CK_Line_Dates", "[PlannedPlantingEnd] >= [PlannedPlantingStart]");
            t.HasCheckConstraint("CK_Line_Priority", "[Priority] BETWEEN 1 AND 9");
        });
        b.Property(l => l.AvailableAreaHa).HasPrecision(18, 4);
        b.Property(l => l.ProjectedPlantingAreaHa).HasPrecision(18, 4);
        b.Property(l => l.ExpectedYieldPerHa).HasPrecision(18, 4);
        b.Property(l => l.ExpectedLossPercent).HasPrecision(9, 4);
        b.Property(l => l.HarvestableAreaHa).HasPrecision(18, 4);
        b.Property(l => l.ExpectedCaneProductionTons).HasPrecision(18, 4);
        b.Property(l => l.Remarks).HasMaxLength(500);

        b.HasIndex(l => new { l.BlockId, l.PlannedPlantingStart, l.PlannedPlantingEnd });
        b.HasOne(l => l.Projection).WithMany(p => p.Lines)
            .HasForeignKey(l => l.ProjectionId).OnDelete(DeleteBehavior.Cascade);
        b.HasOne(l => l.Block).WithMany().HasForeignKey(l => l.BlockId).OnDelete(DeleteBehavior.Restrict);
        b.HasOne(l => l.Farm).WithMany().HasForeignKey(l => l.FarmId).OnDelete(DeleteBehavior.NoAction);
        b.HasOne(l => l.Zone).WithMany().HasForeignKey(l => l.ZoneId).OnDelete(DeleteBehavior.NoAction);
        b.HasOne(l => l.CaneVariety).WithMany().HasForeignKey(l => l.CaneVarietyId).OnDelete(DeleteBehavior.Restrict);
    }
}

public class ApprovalHistoryConfiguration : IEntityTypeConfiguration<ProjectionApprovalHistory>
{
    public void Configure(EntityTypeBuilder<ProjectionApprovalHistory> b)
    {
        b.ToTable("ProjectionApprovalHistory");
        b.Property(h => h.ActionBy).HasMaxLength(100).IsRequired();
        b.Property(h => h.Comments).HasMaxLength(1000);
        b.HasIndex(h => new { h.ProjectionId, h.ActionAtUtc });
        b.HasOne(h => h.Projection).WithMany(p => p.ApprovalHistory)
            .HasForeignKey(h => h.ProjectionId).OnDelete(DeleteBehavior.Cascade);
    }
}

public class ActivityPlanConfiguration : IEntityTypeConfiguration<ActivityPlan>
{
    public void Configure(EntityTypeBuilder<ActivityPlan> b)
    {
        b.ToTable("ActivityPlans", t =>
        {
            t.HasCheckConstraint("CK_Plan_Dates", "[PlannedEndDate] >= [PlannedStartDate]");
            t.HasCheckConstraint("CK_Plan_Area", "[PlannedAreaHa] >= 0");
        });
        b.Property(p => p.PlannedAreaHa).HasPrecision(18, 4);
        b.Property(p => p.DailyTargetHa).HasPrecision(18, 4);
        b.Property(p => p.PlannedWorkingHours).HasPrecision(18, 4);
        b.Property(p => p.RequiredLaborDays).HasPrecision(18, 4);
        b.Property(p => p.PlannedFuelLiters).HasPrecision(18, 4);
        b.Property(p => p.RequiredTractorType).HasMaxLength(60);
        b.Property(p => p.SupervisorName).HasMaxLength(150);
        b.Property(p => p.Remarks).HasMaxLength(1000);

        b.HasIndex(p => new { p.ProjectionId, p.BlockId, p.ActivityId });
        b.HasIndex(p => new { p.PlannedStartDate, p.PlannedEndDate });
        b.HasIndex(p => p.Status);

        b.HasOne(p => p.Projection).WithMany(x => x.ActivityPlans)
            .HasForeignKey(p => p.ProjectionId).OnDelete(DeleteBehavior.Cascade);
        b.HasOne(p => p.ProjectionLine).WithMany(l => l.ActivityPlans)
            .HasForeignKey(p => p.ProjectionLineId).OnDelete(DeleteBehavior.NoAction);
        b.HasOne(p => p.Activity).WithMany().HasForeignKey(p => p.ActivityId).OnDelete(DeleteBehavior.Restrict);
        b.HasOne(p => p.Block).WithMany().HasForeignKey(p => p.BlockId).OnDelete(DeleteBehavior.Restrict);
    }
}

public class PlanningScenarioConfiguration : IEntityTypeConfiguration<PlanningScenario>
{
    public void Configure(EntityTypeBuilder<PlanningScenario> b)
    {
        b.ToTable("PlanningScenarios");
        b.Property(s => s.Name).HasMaxLength(150).IsRequired();
        b.Property(s => s.Description).HasMaxLength(1000);
        b.Property(s => s.SimulatedBy).HasMaxLength(100);
        b.HasIndex(s => new { s.ProjectionId, s.Name }).IsUnique();
        b.HasOne(s => s.Projection).WithMany().HasForeignKey(s => s.ProjectionId).OnDelete(DeleteBehavior.Cascade);
    }
}

public class ScenarioAdjustmentConfiguration : IEntityTypeConfiguration<ScenarioAdjustment>
{
    public void Configure(EntityTypeBuilder<ScenarioAdjustment> b)
    {
        b.ToTable("ScenarioAdjustments");
        b.Property(a => a.Value).HasPrecision(18, 4);
        b.Property(a => a.Remarks).HasMaxLength(500);
        b.HasOne(a => a.Scenario).WithMany(s => s.Adjustments)
            .HasForeignKey(a => a.ScenarioId).OnDelete(DeleteBehavior.Cascade);
        b.HasOne(a => a.Activity).WithMany().HasForeignKey(a => a.ActivityId).OnDelete(DeleteBehavior.Restrict);
    }
}

public class AuditLogConfiguration : IEntityTypeConfiguration<AuditLog>
{
    public void Configure(EntityTypeBuilder<AuditLog> b)
    {
        b.ToTable("AuditLogs");
        b.Property(a => a.UserName).HasMaxLength(100).IsRequired();
        b.Property(a => a.TableName).HasMaxLength(100).IsRequired();
        b.Property(a => a.RecordId).HasMaxLength(60).IsRequired();
        b.Property(a => a.ChangedColumns).HasMaxLength(2000);
        b.Property(a => a.IpAddress).HasMaxLength(60);
        b.Property(a => a.DeviceInfo).HasMaxLength(300);
        b.Property(a => a.Remarks).HasMaxLength(1000);
        b.HasIndex(a => a.TimestampUtc);
        b.HasIndex(a => new { a.TableName, a.RecordId });
        b.HasIndex(a => a.UserName);
    }
}
