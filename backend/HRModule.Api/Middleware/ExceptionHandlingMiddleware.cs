using System.Net;
using System.Text.Json;

namespace HRModule.Api.Middleware;

/// <summary>
/// Converts service-layer exceptions into consistent JSON error responses.
/// Business rule violations (ArgumentException / InvalidOperationException)
/// map to 400 Bad Request; everything else to 500.
/// </summary>
public class ExceptionHandlingMiddleware
{
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
            var status = ex switch
            {
                ArgumentException or InvalidOperationException => HttpStatusCode.BadRequest,
                KeyNotFoundException => HttpStatusCode.NotFound,
                _ => HttpStatusCode.InternalServerError
            };

            if (status == HttpStatusCode.InternalServerError)
                _logger.LogError(ex, "Unhandled exception processing {Path}", context.Request.Path);
            else
                _logger.LogWarning(ex, "Business rule violation on {Path}", context.Request.Path);

            context.Response.ContentType = "application/json";
            context.Response.StatusCode = (int)status;
            await context.Response.WriteAsync(JsonSerializer.Serialize(new
            {
                status = (int)status,
                error = status.ToString(),
                message = ex.Message
            }));
        }
    }
}
