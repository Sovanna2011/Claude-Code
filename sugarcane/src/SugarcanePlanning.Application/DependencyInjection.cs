using Microsoft.Extensions.DependencyInjection;
using SugarcanePlanning.Application.Interfaces;
using SugarcanePlanning.Application.Services;

namespace SugarcanePlanning.Application;

/// <summary>Registers every application service. Infrastructure supplies their dependencies.</summary>
public static class DependencyInjection
{
    public static IServiceCollection AddApplication(this IServiceCollection services)
    {
        services.AddScoped<ILandStructureService, LandStructureService>();
        services.AddScoped<ISeasonService, SeasonService>();
        services.AddScoped<IActivityMasterService, ActivityMasterService>();
        services.AddScoped<IMachineryService, MachineryService>();
        services.AddScoped<IMaterialMasterService, MaterialMasterService>();
        services.AddScoped<IProjectionService, ProjectionService>();
        services.AddScoped<IActivityPlanService, ActivityPlanService>();
        services.AddScoped<ISchedulingService, SchedulingService>();
        services.AddScoped<IMaterialRequirementService, MaterialRequirementService>();
        services.AddScoped<IFuelLaborService, FuelLaborService>();
        services.AddScoped<ICapacityService, CapacityService>();
        services.AddScoped<IScenarioService, ScenarioService>();
        services.AddScoped<IExecutionService, ExecutionService>();
        services.AddScoped<IDashboardService, DashboardService>();
        services.AddScoped<IReportService, ReportService>();
        services.AddScoped<IAuditQueryService, AuditQueryService>();
        services.AddScoped<ILookupService, LookupService>();
        return services;
    }
}
