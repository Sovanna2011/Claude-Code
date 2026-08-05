using System.Text.Json;
using Microsoft.EntityFrameworkCore;
using SugarcanePlanning.Contracts.Common;
using SugarcanePlanning.Domain.Common;

namespace SugarcanePlanning.Api.Middleware;

/// <summary>
/// Single place where domain, concurrency and unexpected exceptions become HTTP responses
/// with a stable <see cref="ApiErrorDto"/> body (section 1: global error handling).
/// </summary>
public class ExceptionHandlingMiddleware
{
    private static readonly JsonSerializerOptions JsonOptions = new(JsonSerializerDefaults.Web);

    /// <summary>Nginx's "client closed request"; ASP.NET Core has no constant for it.</summary>
    private const int ClientClosedRequest = 499;

    private readonly RequestDelegate _next;
    private readonly ILogger<ExceptionHandlingMiddleware> _logger;

    public ExceptionHandlingMiddleware(RequestDelegate next, ILogger<ExceptionHandlingMiddleware> logger)
    {
        _next = next;
        _logger = logger;
    }

    public async Task InvokeAsync(HttpContext context)
    {
        try
        {
            await _next(context);
        }
        catch (Exception ex)
        {
            var (status, error) = Translate(ex, context.TraceIdentifier);

            if (status >= StatusCodes.Status500InternalServerError)
                _logger.LogError(ex, "Unhandled exception on {Method} {Path}", context.Request.Method, context.Request.Path);
            else
                _logger.LogInformation("{Code} on {Method} {Path}: {Message}",
                    error.Code, context.Request.Method, context.Request.Path, error.Message);

            if (context.Response.HasStarted) throw;

            context.Response.Clear();
            context.Response.StatusCode = status;
            context.Response.ContentType = "application/json";
            await context.Response.WriteAsync(JsonSerializer.Serialize(error, JsonOptions));
        }
    }

    private static (int Status, ApiErrorDto Error) Translate(Exception ex, string traceId) => ex switch
    {
        NotFoundException e => (StatusCodes.Status404NotFound,
            new ApiErrorDto { Code = "NOT_FOUND", Message = e.Message, TraceId = traceId }),

        ForbiddenException e => (StatusCodes.Status403Forbidden,
            new ApiErrorDto { Code = "FORBIDDEN", Message = e.Message, TraceId = traceId }),

        BusinessRuleException e => (StatusCodes.Status422UnprocessableEntity,
            new ApiErrorDto { Code = e.Code, Message = e.Message, TraceId = traceId }),

        DbUpdateConcurrencyException => (StatusCodes.Status409Conflict,
            new ApiErrorDto
            {
                Code = "CONCURRENCY_CONFLICT",
                Message = "The record was changed by another user. Reload it and apply your changes again.",
                TraceId = traceId
            }),

        DbUpdateException e => (StatusCodes.Status409Conflict,
            new ApiErrorDto
            {
                Code = "DB_CONSTRAINT",
                Message = "The change violates a database constraint: " + (e.InnerException?.Message ?? e.Message),
                TraceId = traceId
            }),

        ArgumentOutOfRangeException e => (StatusCodes.Status400BadRequest,
            new ApiErrorDto { Code = "INVALID_ARGUMENT", Message = e.Message, TraceId = traceId }),

        OperationCanceledException => (ClientClosedRequest,
            new ApiErrorDto { Code = "CANCELLED", Message = "The request was cancelled.", TraceId = traceId }),

        _ => (StatusCodes.Status500InternalServerError,
            new ApiErrorDto
            {
                Code = "INTERNAL_ERROR",
                Message = "An unexpected error occurred. Quote the trace id when reporting it.",
                TraceId = traceId
            })
    };
}
