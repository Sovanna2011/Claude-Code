using ErpS4.Api.Security;
using ErpS4.Application.Workflow;
using Microsoft.AspNetCore.Mvc;

namespace ErpS4.Api.Endpoints;

/// <summary>Approval inbox and decisions.</summary>
public static class ApprovalEndpoints
{
    public static IEndpointRouteBuilder MapApprovalEndpoints(this IEndpointRouteBuilder app)
    {
        var group = app.MapGroup("/api/v1/approvals")
            .WithTags("Approvals")
            .RequireAuthorization();

        group.MapGet("/inbox", InboxAsync)
            .WithSummary("Tasks waiting for the signed-in user")
            .WithDescription("Includes tasks the user holds as a substitute, flagged as such.")
            .Produces<IReadOnlyList<InboxItem>>();

        group.MapPost("/{instanceNumber}/approve", ApproveAsync)
            .WithSummary("Approve a step")
            .WithDescription(
                "The last approval posts the document. Whoever submitted it cannot decide " +
                "on it, however senior they are.")
            .RequireAuthorization(Policies.Permission(Policies.ApproveJournalEntry))
            .Produces<DecisionResult>()
            .ProducesProblem(StatusCodes.Status422UnprocessableEntity);

        group.MapPost("/{instanceNumber}/reject", RejectAsync)
            .WithSummary("Reject the document")
            .RequireAuthorization(Policies.Permission(Policies.ApproveJournalEntry))
            .Produces<DecisionResult>()
            .ProducesProblem(StatusCodes.Status422UnprocessableEntity);

        return app;
    }

    private static async Task<IResult> InboxAsync(
        IWorkflowService workflow,
        ICurrentUserAccessor currentUser,
        CancellationToken cancellationToken) =>
        Results.Ok(await workflow.GetInboxAsync(currentUser.UserName, cancellationToken));

    private static async Task<IResult> ApproveAsync(
        string instanceNumber,
        [FromBody] DecisionBody? body,
        IWorkflowService workflow,
        CancellationToken cancellationToken) =>
        Decision(await workflow.ApproveAsync(
            new ApprovalRequest(instanceNumber, body?.Comment), cancellationToken));

    private static async Task<IResult> RejectAsync(
        string instanceNumber,
        [FromBody] DecisionBody body,
        IWorkflowService workflow,
        CancellationToken cancellationToken) =>
        Decision(await workflow.RejectAsync(
            new ApprovalRequest(instanceNumber, body.Comment), cancellationToken));

    private static IResult Decision(DecisionResult result) =>
        result.IsSuccess
            ? Results.Ok(result)
            : Results.Problem(
                title: "The decision was refused",
                detail: result.Violations.FirstOrDefault()?.Message,
                statusCode: StatusCodes.Status422UnprocessableEntity,
                type: "https://errors.erps4.local/WF.REJECTED",
                extensions: new Dictionary<string, object?>
                {
                    ["errorCode"] = "WF.REJECTED",
                    ["errors"] = result.Violations
                        .Select(v => new { v.Code, v.Message, v.Field }).ToArray(),
                    ["postingErrors"] = result.PostingErrors
                        .Select(e => new { e.Code, e.Message, e.Field, e.LineNumber }).ToArray(),
                });
}

public sealed record DecisionBody(string? Comment);
