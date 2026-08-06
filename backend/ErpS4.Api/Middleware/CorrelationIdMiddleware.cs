namespace ErpS4.Api.Middleware;

/// <summary>
/// Gives every request a correlation id, echoes it back, and puts it in the log
/// scope. It travels onto the audit rows and the outbox messages the request
/// writes, so one id ties a user action to its document, its event and its log
/// lines.
/// </summary>
public sealed class CorrelationIdMiddleware(RequestDelegate next, ILogger<CorrelationIdMiddleware> logger)
{
    public async Task InvokeAsync(HttpContext context)
    {
        var correlationId = context.Request.Headers[ProblemDetailsMiddleware.CorrelationHeader]
            .FirstOrDefault();

        if (string.IsNullOrWhiteSpace(correlationId) || !Guid.TryParse(correlationId, out var parsed))
        {
            parsed = Guid.NewGuid();
        }

        context.Items[ProblemDetailsMiddleware.CorrelationHeader] = parsed;
        context.Response.Headers[ProblemDetailsMiddleware.CorrelationHeader] = parsed.ToString();

        using (logger.BeginScope(new Dictionary<string, object> { ["CorrelationId"] = parsed }))
        {
            await next(context);
        }
    }
}
