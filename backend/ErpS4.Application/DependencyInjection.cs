using ErpS4.Application.BusinessPartners;
using ErpS4.Application.Assets;
using ErpS4.Application.Clearing;
using ErpS4.Application.Workflow;
using ErpS4.Application.Posting;
using ErpS4.Application.Services;
using ErpS4.Database;
using FluentValidation;
using Microsoft.Extensions.DependencyInjection;

namespace ErpS4.Application;

/// <summary>Registration for the application module.</summary>
public static class DependencyInjection
{
    /// <summary>
    /// Registers the posting engine and its collaborators. Everything is scoped
    /// to the request, alongside the DbContext, so a captive singleton cannot
    /// hold a dead context.
    /// </summary>
    public static IServiceCollection AddErpApplication(this IServiceCollection services)
    {
        services.AddScoped<IErpDataContext>(provider => provider.GetRequiredService<ErpDbContext>());
        services.AddScoped<INumberRangeService, NumberRangeService>();
        services.AddScoped<ICurrencyConverter, CurrencyConverter>();
        services.AddScoped<IPostingEngine, PostingEngine>();
        services.AddScoped<IBusinessPartnerSyncService, BusinessPartnerSyncService>();
        services.AddScoped<IClearingService, ClearingService>();
        services.AddScoped<IAssetService, AssetService>();
        services.AddScoped<IWorkflowService, WorkflowService>();
        services.AddScoped<PostJournalEntryHandler>();

        services.AddValidatorsFromAssemblyContaining<PostJournalEntryCommandValidator>();
        services.TryAddSingletonTimeProvider();

        return services;
    }

    private static void TryAddSingletonTimeProvider(this IServiceCollection services)
    {
        if (services.All(d => d.ServiceType != typeof(TimeProvider)))
        {
            services.AddSingleton(TimeProvider.System);
        }
    }
}
