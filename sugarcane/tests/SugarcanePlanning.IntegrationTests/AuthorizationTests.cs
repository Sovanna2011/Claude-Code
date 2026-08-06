using System.Reflection;
using Microsoft.AspNetCore.Authorization;
using Microsoft.AspNetCore.Mvc;
using Microsoft.AspNetCore.Mvc.Routing;
using SugarcanePlanning.Contracts.Auth;
using SugarcanePlanning.Contracts.Projections;
using SugarcanePlanning.Domain.Common;
using SugarcanePlanning.Domain.Enums;
using Xunit;

namespace SugarcanePlanning.IntegrationTests;

/// <summary>
/// Section 21 gives ten roles a different set of the View / Create / Edit / Delete / Submit /
/// Approve / Reject / Revise / Close / Export permissions. Two things have to hold: every
/// endpoint must declare which permission it needs, and the services must actually refuse a
/// caller who lacks it. An endpoint that simply forgets its attribute is authenticated-only —
/// it still answers, to anyone with a login.
/// </summary>
public class AuthorizationTests
{
    /// <summary>
    /// The only endpoints allowed to be authenticated-only, each for a stated reason. Anything
    /// else that appears here is an endpoint whose attribute was forgotten.
    /// </summary>
    private static readonly Dictionary<string, string> AuthenticatedOnly = new()
    {
        // The permission depends on the action in the request body — Submit, Approve, Reject,
        // Revise and Close each map to a different one — so no static attribute can express it.
        // ProjectionService.ExecuteWorkflowAsync checks before it reads anything; the role
        // matrix below is what proves the check is really there.
        ["ProjectionsController.Workflow"] = "policy depends on the action in the body",

        // Self-scoped: both resolve the caller from their own token rather than from a
        // parameter, and the password change requires the current password. Every signed-in
        // user must be able to do these, so there is no narrower policy to apply.
        ["AuthController.Me"] = "returns the caller's own profile",
        ["AuthController.ChangePassword"] = "changes the caller's own password"
    };

    [Fact]
    public void Every_endpoint_declares_the_permission_it_requires()
    {
        var controllers = typeof(Program).Assembly.GetTypes()
            .Where(t => typeof(ControllerBase).IsAssignableFrom(t) && !t.IsAbstract)
            .ToList();

        Assert.NotEmpty(controllers);

        var ungated = new List<string>();
        var actions = 0;

        foreach (var controller in controllers)
        {
            foreach (var method in controller.GetMethods(BindingFlags.Public | BindingFlags.Instance | BindingFlags.DeclaredOnly))
            {
                if (!method.GetCustomAttributes<HttpMethodAttribute>().Any()) continue;
                actions++;

                if (method.GetCustomAttribute<AllowAnonymousAttribute>() is not null) continue;

                var policy = method.GetCustomAttributes<AuthorizeAttribute>()
                    .Select(a => a.Policy)
                    .FirstOrDefault(p => !string.IsNullOrWhiteSpace(p));

                var name = $"{controller.Name}.{method.Name}";
                if (policy is null && !AuthenticatedOnly.ContainsKey(name)) ungated.Add(name);

                // A typo in a policy name is silently permissive-looking at compile time and
                // fails closed at run time; either way it must not reach production.
                if (policy is not null)
                    Assert.True(Policies.RoleMap.ContainsKey(policy),
                        $"{name} requires policy '{policy}', which is not in Policies.RoleMap.");
            }
        }

        Assert.True(actions > 100, $"expected the full API surface, found only {actions} actions");
        Assert.True(ungated.Count == 0,
            "these endpoints are authenticated-only, so any signed-in user can call them: "
            + string.Join(", ", ungated));
    }

    [Fact]
    public void Anonymous_access_is_allowed_only_for_signing_in()
    {
        var anonymous = typeof(Program).Assembly.GetTypes()
            .Where(t => typeof(ControllerBase).IsAssignableFrom(t) && !t.IsAbstract)
            .SelectMany(t => t.GetMethods(BindingFlags.Public | BindingFlags.Instance | BindingFlags.DeclaredOnly)
                .Where(m => m.GetCustomAttributes<HttpMethodAttribute>().Any())
                .Where(m => m.GetCustomAttribute<AllowAnonymousAttribute>() is not null)
                .Select(m => $"{t.Name}.{m.Name}"))
            .ToList();

        Assert.Equal(new[] { "AuthController.Login" }, anonymous);
    }

    // ------------------------------------------------------- behaviour, per role

    private static PlanningTestHost HostAs(params string[] roles)
    {
        var host = new PlanningTestHost();
        host.User.Roles = roles;
        return host;
    }

    /// <summary>Drives a projection to Submitted using a fully privileged user.</summary>
    private static async Task<int> SubmittedProjectionAsync(PlanningTestHost host)
    {
        var projection = await host.Projections.CreateAsync(host.NewProjection());
        await host.Projections.ExecuteWorkflowAsync(projection.Id,
            new WorkflowActionDto { Action = ApprovalAction.Submit });
        return projection.Id;
    }

    [Theory]
    [InlineData(ApprovalAction.Submit)]
    [InlineData(ApprovalAction.Approve)]
    [InlineData(ApprovalAction.Reject)]
    [InlineData(ApprovalAction.Revise)]
    [InlineData(ApprovalAction.Close)]
    public async Task A_report_viewer_cannot_move_a_projection_through_the_workflow(ApprovalAction action)
    {
        using var host = HostAs(AppRoles.ReportViewer);

        // Nothing is created yet, so a caller who got past the permission check would fail with
        // "not found" instead — which is exactly the difference this asserts.
        await Assert.ThrowsAsync<ForbiddenException>(() => host.Projections.ExecuteWorkflowAsync(
            999, new WorkflowActionDto { Action = action, Comments = "reason" }));
    }

    [Fact]
    public async Task A_planner_may_submit_but_not_approve()
    {
        using var host = HostAs(AppRoles.AgriculturalPlanner);
        var id = await SubmittedProjectionAsync(host);

        await Assert.ThrowsAsync<ForbiddenException>(() => host.Projections.ExecuteWorkflowAsync(
            id, new WorkflowActionDto { Action = ApprovalAction.Approve }));
    }

    [Fact]
    public async Task An_approver_may_approve_but_not_submit()
    {
        using var host = new PlanningTestHost();
        var id = await SubmittedProjectionAsync(host);        // arranged with full rights

        host.User.Roles = new[] { AppRoles.ManagementApprover };
        var approved = await host.Projections.ExecuteWorkflowAsync(id,
            new WorkflowActionDto { Action = ApprovalAction.Approve });
        Assert.Equal(ProjectionStatus.Approved, approved.Status);

        await Assert.ThrowsAsync<ForbiddenException>(() => host.Projections.ExecuteWorkflowAsync(
            id, new WorkflowActionDto { Action = ApprovalAction.Submit }));
    }

    // The dependency-override permission is exercised where the rule itself lives, in
    // SchedulingConflictTests: a field supervisor is refused, a plantation manager succeeds and
    // the reason is stored.

    [Fact]
    public void The_role_map_covers_every_policy_and_every_role_is_reachable()
    {
        var declared = typeof(Policies).GetFields(BindingFlags.Public | BindingFlags.Static)
            .Where(f => f.FieldType == typeof(string))
            .Select(f => (string)f.GetValue(null)!)
            .ToList();

        Assert.Equal(declared.OrderBy(p => p), Policies.RoleMap.Keys.OrderBy(p => p));

        var granted = Policies.RoleMap.Values.SelectMany(r => r).Distinct().ToList();
        Assert.All(AppRoles.All, role => Assert.Contains(role, granted));

        // Only the administrator may read the audit trail (section 22).
        Assert.Equal(new[] { AppRoles.SystemAdministrator }, Policies.RoleMap[Policies.Administer]);
    }
}
