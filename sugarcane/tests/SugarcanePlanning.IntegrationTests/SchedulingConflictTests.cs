using SugarcanePlanning.Contracts.Activities;
using SugarcanePlanning.Contracts.Auth;
using SugarcanePlanning.Contracts.Common;
using SugarcanePlanning.Contracts.Machinery;
using SugarcanePlanning.Contracts.Projections;
using SugarcanePlanning.Contracts.Scheduling;
using SugarcanePlanning.Domain.Common;
using SugarcanePlanning.Domain.Enums;
using Xunit;

namespace SugarcanePlanning.IntegrationTests;

/// <summary>Each of the eight conflict checks of section 10, plus the manager override.</summary>
public class SchedulingConflictTests : IDisposable
{
    private readonly PlanningTestHost _host = new();
    public void Dispose() => _host.Dispose();

    private async Task<IReadOnlyList<ActivityPlanDto>> PlansAsync()
    {
        var projection = await _host.Projections.CreateAsync(_host.NewProjection());
        await _host.Projections.ExecuteWorkflowAsync(projection.Id, new WorkflowActionDto { Action = ApprovalAction.Submit });
        await _host.Projections.ExecuteWorkflowAsync(projection.Id, new WorkflowActionDto { Action = ApprovalAction.Approve });
        await _host.ActivityPlans.GenerateAsync(new GenerateActivityPlanRequest { ProjectionId = projection.Id });

        var plans = await _host.ActivityPlans.GetPlansAsync(projection.Id, null, null, new QueryParameters());
        return plans.Items;
    }

    private static ResourceScheduleUpsertDto Booking(ActivityPlanDto plan, DateOnly date, int startHour = 7,
        int hours = 8, int? tractorId = null, int? equipmentId = null, int? operatorId = null) => new()
    {
        ActivityPlanId = plan.Id,
        ScheduleDate = date,
        PlannedStart = date.ToDateTime(new TimeOnly(startHour, 0)),
        PlannedEnd = date.ToDateTime(new TimeOnly(startHour, 0)).AddHours(hours),
        TractorId = tractorId,
        EquipmentId = equipmentId,
        OperatorId = operatorId,
        PlannedAreaHa = 4m,
        ExpectedWorkingHours = hours
    };

    // -------------------------------------------------------------- happy path

    [Fact]
    public async Task A_clean_booking_is_accepted_and_moves_the_plan_to_Scheduled()
    {
        var plans = await PlansAsync();
        var plough = plans.Single(p => p.ActivityId == _host.PloughActivityId);

        var booking = await _host.Scheduling.CreateAsync(
            Booking(plough, plough.PlannedStartDate, tractorId: _host.TractorAId, operatorId: _host.OperatorId));

        Assert.Equal(ScheduleStatus.Planned, booking.Status);
        Assert.Equal("TR-01", booking.TractorCode);

        var reloaded = await _host.ActivityPlans.GetPlanAsync(plough.Id);
        Assert.Equal(ActivityStatus.Scheduled, reloaded.Status);
    }

    // ------------------------------------------------------- 1 tractor clash

    [Fact]
    public async Task A_tractor_cannot_be_booked_twice_in_overlapping_hours()
    {
        var plans = await PlansAsync();
        var plough = plans.Single(p => p.ActivityId == _host.PloughActivityId);
        var date = plough.PlannedStartDate;

        await _host.Scheduling.CreateAsync(Booking(plough, date, startHour: 7, hours: 8, tractorId: _host.TractorAId));

        var validation = await _host.Scheduling.ValidateAsync(
            Booking(plough, date, startHour: 12, hours: 4, tractorId: _host.TractorAId), null);

        Assert.False(validation.IsValid);
        Assert.Contains(validation.Conflicts, c => c.Type == ConflictType.TractorDoubleBooking);

        var ex = await Assert.ThrowsAsync<BusinessRuleException>(() => _host.Scheduling.CreateAsync(
            Booking(plough, date, startHour: 12, hours: 4, tractorId: _host.TractorAId)));
        Assert.Equal("SCHEDULE_CONFLICT", ex.Code);
    }

    [Fact]
    public async Task Back_to_back_bookings_of_the_same_tractor_are_allowed()
    {
        var plans = await PlansAsync();
        var plough = plans.Single(p => p.ActivityId == _host.PloughActivityId);
        var date = plough.PlannedStartDate;

        await _host.Scheduling.CreateAsync(Booking(plough, date, startHour: 6, hours: 4, tractorId: _host.TractorAId));
        var second = await _host.Scheduling.CreateAsync(Booking(plough, date, startHour: 10, hours: 4, tractorId: _host.TractorAId));

        Assert.True(second.Id > 0);
    }

    // ---------------------------------------------------- 2 equipment clash

    [Fact]
    public async Task Equipment_cannot_be_booked_twice_in_overlapping_hours()
    {
        var plans = await PlansAsync();
        var plant = plans.Single(p => p.ActivityId == _host.PlantActivityId);
        var date = plant.PlannedStartDate;

        await _host.Scheduling.CreateAsync(Booking(plant, date, tractorId: _host.TractorAId, equipmentId: _host.PlanterId));

        var validation = await _host.Scheduling.ValidateAsync(
            Booking(plant, date, startHour: 9, equipmentId: _host.PlanterId), null);

        Assert.Contains(validation.Conflicts, c => c.Type == ConflictType.EquipmentDoubleBooking);
    }

    // ---------------------------------------------------- 3 operator clash

    [Fact]
    public async Task An_operator_cannot_be_booked_twice_in_overlapping_hours()
    {
        var plans = await PlansAsync();
        var plough = plans.Single(p => p.ActivityId == _host.PloughActivityId);
        var date = plough.PlannedStartDate;

        await _host.Scheduling.CreateAsync(Booking(plough, date, tractorId: _host.TractorAId, operatorId: _host.OperatorId));

        var validation = await _host.Scheduling.ValidateAsync(
            Booking(plough, date, startHour: 10, tractorId: _host.TractorBId, operatorId: _host.OperatorId), null);

        Assert.Contains(validation.Conflicts, c => c.Type == ConflictType.OperatorDoubleBooking);
    }

    // -------------------------------------------------------- 4 maintenance

    [Fact]
    public async Task A_tractor_inside_its_maintenance_window_cannot_be_booked()
    {
        var plans = await PlansAsync();
        var plough = plans.Single(p => p.ActivityId == _host.PloughActivityId);
        var date = plough.PlannedStartDate;

        var tractor = await _host.Machinery.GetTractorAsync(_host.TractorAId);
        await _host.Machinery.UpdateTractorAsync(tractor.Id, new TractorUpsertDto
        {
            CompanyId = tractor.CompanyId,
            EstateId = tractor.EstateId,
            AssetNo = tractor.AssetNo,
            Code = tractor.Code,
            Horsepower = tractor.Horsepower,
            CurrentFarmId = tractor.CurrentFarmId,
            DailyCapacityHa = tractor.DailyCapacityHa,
            FuelConsumptionPerHour = tractor.FuelConsumptionPerHour,
            FuelConsumptionPerHa = tractor.FuelConsumptionPerHa,
            Availability = AvailabilityStatus.UnderMaintenance,
            Maintenance = MaintenanceStatus.InService,
            MaintenanceFromDate = date.AddDays(-1),
            MaintenanceToDate = date.AddDays(5),
            IsActive = true
        });

        var validation = await _host.Scheduling.ValidateAsync(Booking(plough, date, tractorId: _host.TractorAId), null);

        Assert.False(validation.IsValid);
        Assert.Contains(validation.Conflicts, c => c.Type == ConflictType.MachineUnderMaintenance);
    }

    // --------------------------------------------------------- 5 horsepower

    [Fact]
    public async Task A_tractor_below_the_implement_minimum_horsepower_is_rejected()
    {
        var plans = await PlansAsync();
        var plant = plans.Single(p => p.ActivityId == _host.PlantActivityId);

        // The planter needs 120 hp; tractor B delivers 90 hp.
        var validation = await _host.Scheduling.ValidateAsync(
            Booking(plant, plant.PlannedStartDate, tractorId: _host.TractorBId, equipmentId: _host.PlanterId), null);

        Assert.False(validation.IsValid);
        Assert.Contains(validation.Conflicts, c => c.Type == ConflictType.InsufficientHorsepower);
    }

    [Fact]
    public async Task A_tractor_that_meets_the_minimum_horsepower_is_accepted()
    {
        var plans = await PlansAsync();
        var plant = plans.Single(p => p.ActivityId == _host.PlantActivityId);

        var validation = await _host.Scheduling.ValidateAsync(
            Booking(plant, plant.PlannedStartDate, tractorId: _host.TractorAId, equipmentId: _host.PlanterId), null);

        Assert.DoesNotContain(validation.Conflicts, c => c.Type == ConflictType.InsufficientHorsepower);
    }

    // ----------------------------------------------- 8 compatibility matrix

    [Fact]
    public async Task Once_a_compatibility_list_exists_only_the_listed_tractors_may_be_used()
    {
        var plans = await PlansAsync();
        var plant = plans.Single(p => p.ActivityId == _host.PlantActivityId);

        // Register a third tractor and pair only that one with the planter.
        var spare = await _host.Machinery.CreateTractorAsync(new TractorUpsertDto
        {
            CompanyId = _host.CompanyId,
            EstateId = _host.EstateId,
            CurrentFarmId = _host.FarmId,
            AssetNo = "AST-3",
            Code = "TR-03",
            Horsepower = 200,
            DailyCapacityHa = 6m
        });
        await _host.Machinery.AddCompatibilityAsync(new CompatibilityUpsertDto
        {
            TractorId = spare!.Id,
            EquipmentId = _host.PlanterId
        });

        var listed = await _host.Scheduling.ValidateAsync(
            Booking(plant, plant.PlannedStartDate, tractorId: spare.Id, equipmentId: _host.PlanterId), null);
        Assert.DoesNotContain(listed.Conflicts, c => c.Type == ConflictType.IncompatibleEquipment);

        var unlisted = await _host.Scheduling.ValidateAsync(
            Booking(plant, plant.PlannedStartDate, tractorId: _host.TractorAId, equipmentId: _host.PlanterId), null);
        Assert.Contains(unlisted.Conflicts, c => c.Type == ConflictType.IncompatibleEquipment);
    }

    [Fact]
    public async Task A_pairing_below_the_minimum_horsepower_cannot_even_be_registered()
    {
        var ex = await Assert.ThrowsAsync<BusinessRuleException>(() =>
            _host.Machinery.AddCompatibilityAsync(new CompatibilityUpsertDto
            {
                TractorId = _host.TractorBId,           // 90 hp
                EquipmentId = _host.PlanterId           // needs 120 hp
            }));

        Assert.Equal("INSUFFICIENT_HP", ex.Code);
    }

    // ------------------------------------------------- 7 activity dependency

    [Fact]
    public async Task A_dependent_activity_cannot_be_scheduled_before_its_predecessor_finishes()
    {
        var plans = await PlansAsync();
        var plough = plans.Single(p => p.ActivityId == _host.PloughActivityId);
        var plant = plans.Single(p => p.ActivityId == _host.PlantActivityId);

        // The last day of ploughing is inside the plan window but before the predecessor finishes.
        var validation = await _host.Scheduling.ValidateAsync(
            Booking(plant, plough.PlannedEndDate, tractorId: _host.TractorAId), null);

        Assert.False(validation.IsValid);
        Assert.Contains(validation.Conflicts, c => c.Type == ConflictType.ActivityDependency);
        Assert.DoesNotContain(validation.Conflicts, c => c.Type == ConflictType.OutsidePlanPeriod);
    }

    [Fact]
    public async Task A_manager_may_override_a_dependency_and_the_reason_is_stored()
    {
        var plans = await PlansAsync();
        var plough = plans.Single(p => p.ActivityId == _host.PloughActivityId);
        var plant = plans.Single(p => p.ActivityId == _host.PlantActivityId);

        var request = Booking(plant, plough.PlannedEndDate, tractorId: _host.TractorAId);
        request.OverrideDependency = true;
        request.DependencyOverrideReason = "Land already prepared last season";

        var booking = await _host.Scheduling.CreateAsync(request);

        Assert.True(booking.DependencyOverrideApproved);
        Assert.Equal("Land already prepared last season", booking.DependencyOverrideReason);
    }

    [Fact]
    public async Task A_user_without_the_override_policy_cannot_bypass_a_dependency()
    {
        var plans = await PlansAsync();
        var plough = plans.Single(p => p.ActivityId == _host.PloughActivityId);
        var plant = plans.Single(p => p.ActivityId == _host.PlantActivityId);

        _host.User.Roles = new[] { AppRoles.FieldSupervisor };

        var request = Booking(plant, plough.PlannedEndDate, tractorId: _host.TractorAId);
        request.OverrideDependency = true;
        request.DependencyOverrideReason = "trust me";

        var validation = await _host.Scheduling.ValidateAsync(request, null);
        Assert.False(validation.IsValid);
        Assert.Contains(validation.Conflicts, c => c.Type == ConflictType.ActivityDependency && c.IsBlocking);
    }

    // ------------------------------------------------------ 6 plan period

    [Fact]
    public async Task Scheduling_outside_the_approved_plan_period_is_rejected()
    {
        var plans = await PlansAsync();
        var plant = plans.Single(p => p.ActivityId == _host.PlantActivityId);

        var validation = await _host.Scheduling.ValidateAsync(
            Booking(plant, new DateOnly(2026, 12, 1), tractorId: _host.TractorAId), null);

        Assert.Contains(validation.Conflicts, c => c.Type == ConflictType.OutsidePlanPeriod);
    }

    // ------------------------------------------------------------- edit & board

    [Fact]
    public async Task Editing_a_booking_does_not_report_it_as_a_conflict_with_itself()
    {
        var plans = await PlansAsync();
        var plough = plans.Single(p => p.ActivityId == _host.PloughActivityId);
        var date = plough.PlannedStartDate;

        var booking = await _host.Scheduling.CreateAsync(Booking(plough, date, tractorId: _host.TractorAId));

        var request = Booking(plough, date, startHour: 8, tractorId: _host.TractorAId);
        request.RowVersion = booking.RowVersion;
        var updated = await _host.Scheduling.UpdateAsync(booking.Id, request);

        Assert.Equal(booking.Id, updated.Id);
        Assert.Equal(8, updated.PlannedStart.Hour);
    }

    [Fact]
    public async Task A_cancelled_booking_frees_the_resource()
    {
        var plans = await PlansAsync();
        var plough = plans.Single(p => p.ActivityId == _host.PloughActivityId);
        var date = plough.PlannedStartDate;

        var booking = await _host.Scheduling.CreateAsync(Booking(plough, date, tractorId: _host.TractorAId));
        await _host.Scheduling.CancelAsync(booking.Id);

        var validation = await _host.Scheduling.ValidateAsync(Booking(plough, date, tractorId: _host.TractorAId), null);
        Assert.DoesNotContain(validation.Conflicts, c => c.Type == ConflictType.TractorDoubleBooking);
    }

    [Fact]
    public async Task The_board_groups_bookings_by_day_and_by_resource()
    {
        var plans = await PlansAsync();
        var plough = plans.Single(p => p.ActivityId == _host.PloughActivityId);
        var date = plough.PlannedStartDate;

        await _host.Scheduling.CreateAsync(Booking(plough, date, tractorId: _host.TractorAId, operatorId: _host.OperatorId));
        await _host.Scheduling.CreateAsync(Booking(plough, date.AddDays(1), tractorId: _host.TractorAId));

        var byDay = await _host.Scheduling.GetBoardAsync(date, date.AddDays(6), "day", null, null);
        Assert.Equal(7, byDay.Days.Count);
        Assert.Equal(1, byDay.Days.First(d => d.Date == date).BookingCount);

        var byTractor = await _host.Scheduling.GetBoardAsync(date, date.AddDays(6), "tractor", null, null);
        var lane = Assert.Single(byTractor.Rows);
        Assert.Equal("TR-01", lane.ResourceCode);
        Assert.Equal(2, lane.UtilizedDays);
        Assert.Equal(16m, lane.TotalHours);
    }

    [Fact]
    public async Task A_booking_whose_end_precedes_its_start_is_refused()
    {
        var plans = await PlansAsync();
        var plough = plans.Single(p => p.ActivityId == _host.PloughActivityId);

        var request = Booking(plough, plough.PlannedStartDate);
        request.PlannedEnd = request.PlannedStart.AddHours(-1);

        var ex = await Assert.ThrowsAsync<BusinessRuleException>(() => _host.Scheduling.ValidateAsync(request, null));
        Assert.Equal("TIME_RANGE", ex.Code);
    }

    [Fact]
    public async Task Regenerating_a_plan_with_live_bookings_is_blocked()
    {
        var projection = await _host.Projections.CreateAsync(_host.NewProjection());
        await _host.Projections.ExecuteWorkflowAsync(projection.Id, new WorkflowActionDto { Action = ApprovalAction.Submit });
        await _host.Projections.ExecuteWorkflowAsync(projection.Id, new WorkflowActionDto { Action = ApprovalAction.Approve });
        await _host.ActivityPlans.GenerateAsync(new GenerateActivityPlanRequest { ProjectionId = projection.Id });

        var plans = await _host.ActivityPlans.GetPlansAsync(projection.Id, null, null, new QueryParameters());
        var plough = plans.Items.Single(p => p.ActivityId == _host.PloughActivityId);
        await _host.Scheduling.CreateAsync(Booking(plough, plough.PlannedStartDate, tractorId: _host.TractorAId));

        var ex = await Assert.ThrowsAsync<BusinessRuleException>(() => _host.ActivityPlans.GenerateAsync(
            new GenerateActivityPlanRequest { ProjectionId = projection.Id, Regenerate = true }));

        Assert.Equal("PLANS_SCHEDULED", ex.Code);
    }

    [Fact]
    public async Task A_machine_with_live_bookings_cannot_be_deleted()
    {
        var plans = await PlansAsync();
        var plough = plans.Single(p => p.ActivityId == _host.PloughActivityId);
        await _host.Scheduling.CreateAsync(Booking(plough, plough.PlannedStartDate, tractorId: _host.TractorAId));

        var ex = await Assert.ThrowsAsync<BusinessRuleException>(() => _host.Machinery.DeleteTractorAsync(_host.TractorAId));
        Assert.Equal("IN_USE", ex.Code);
    }
}
