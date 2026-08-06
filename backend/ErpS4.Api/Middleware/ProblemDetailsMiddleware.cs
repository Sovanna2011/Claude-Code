using System.Diagnostics;
using ErpS4.Database;
using ErpS4.Domain;
using Microsoft.AspNetCore.Mvc;
using Microsoft.EntityFrameworkCore;

namespace ErpS4.Api.Middleware;

/// <summary>
/// Turns anything that escapes a handler into an RFC 7807 response.
/// </summary>
/// <remarks>
/// Two rules drive this: the client always gets a stable application error code
/// it can branch on, and it never gets a stack trace, a SQL statement or a
/// connection string. The full detail goes to the log, keyed by the same trace
/// id the client is shown, so support can join the two.
/// </remarks>
public sealed class ProblemDetailsMiddleware(RequestDelegate next, ILogger<ProblemDetailsMiddleware> logger)
{
    public const string CorrelationHeader = "X-Correlation-Id";

    public async Task InvokeAsync(HttpContext context)
    {
        try
        {
            await next(context);
        }
        catch (Exception exception)
        {
            await WriteAsync(context, exception);
        }
    }

    private async Task WriteAsync(HttpContext context, Exception exception)
    {
        var traceId = Activity.Current?.Id ?? context.TraceIdentifier;
        var correlationId = context.Items[CorrelationHeader]?.ToString();

        var (status, code, title) = Classify(exception);

        logger.Log(
            status >= StatusCodes.Status500InternalServerError
                ? LogLevel.Error
                : LogLevel.Warning,
            exception,
            "{Code} on {Method} {Path} (trace {TraceId}, correlation {CorrelationId})",
            code, context.Request.Method, context.Request.Path, traceId, correlationId);

        if (context.Response.HasStarted)
        {
            // Too late to change the status: the log entry above is the record.
            return;
        }

        var problem = new ProblemDetails
        {
            Status = status,
            Title = title,
            Type = $"https://errors.erps4.local/{code}",
            Detail = status >= StatusCodes.Status500InternalServerError
                ? "The request could not be completed. Quote the trace id when reporting this."
                : exception.Message,
            Instance = context.Request.Path,
        };

        problem.Extensions["errorCode"] = code;
        problem.Extensions["traceId"] = traceId;
        if (correlationId is not null)
        {
            problem.Extensions["correlationId"] = correlationId;
        }

        if (exception is PostingRejectedException rejected)
        {
            problem.Extensions["errors"] = rejected.Errors
                .Select(e => new { e.Code, e.Message, e.Field, e.LineNumber })
                .ToArray();
        }

        context.Response.StatusCode = status;
        context.Response.ContentType = "application/problem+json";
        await context.Response.WriteAsJsonAsync(problem);
    }

    private static (int Status, string Code, string Title) Classify(Exception exception) =>
        exception switch
        {
            PostingRejectedException =>
                (StatusCodes.Status422UnprocessableEntity, "POSTING.REJECTED",
                    "The document was rejected"),
            NumberRangeExhaustedException =>
                (StatusCodes.Status409Conflict, "NUMBER_RANGE.EXHAUSTED",
                    "Number range exhausted"),
            DbUpdateConcurrencyException =>
                (StatusCodes.Status409Conflict, "CONCURRENCY.CONFLICT",
                    "The record was changed by someone else"),
            DbUpdateException =>
                (StatusCodes.Status409Conflict, "PERSISTENCE.CONFLICT",
                    "The change conflicts with existing data"),
            UnauthorizedAccessException =>
                (StatusCodes.Status403Forbidden, "AUTH.FORBIDDEN", "Not authorized"),
            KeyNotFoundException =>
                (StatusCodes.Status404NotFound, "RESOURCE.NOT_FOUND", "Not found"),
            ArgumentException or InvalidOperationException =>
                (StatusCodes.Status400BadRequest, "REQUEST.INVALID", "Invalid request"),
            // 499 is nginx's "client closed request"; ASP.NET Core has no
            // constant for it, and no other code says "nobody is listening".
            OperationCanceledException => (499, "REQUEST.CANCELLED", "The request was cancelled"),
            _ => (StatusCodes.Status500InternalServerError, "SERVER.ERROR", "Unexpected error"),
        };
}
