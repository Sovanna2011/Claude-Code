using System.Net;
using System.Net.Http.Json;
using System.Text.Json;
using System.Text.Json.Serialization;
using SugarcanePlanning.Contracts.Common;

namespace SugarcanePlanning.Client.Services;

/// <summary>
/// Thin wrapper over <see cref="HttpClient"/> that adds the bearer token, converts an
/// <see cref="ApiErrorDto"/> body into <see cref="ApiException"/> and centralises JSON options.
/// </summary>
public class ApiClient
{
    public static readonly JsonSerializerOptions JsonOptions = CreateOptions();

    private readonly HttpClient _http;
    private readonly TokenStore _tokens;

    public ApiClient(HttpClient http, TokenStore tokens)
    {
        _http = http;
        _tokens = tokens;
    }

    private static JsonSerializerOptions CreateOptions()
    {
        var options = new JsonSerializerOptions(JsonSerializerDefaults.Web);
        options.Converters.Add(new JsonStringEnumConverter());
        return options;
    }

    public async Task<T?> GetAsync<T>(string url, CancellationToken ct = default)
    {
        await AttachTokenAsync();
        var response = await _http.GetAsync(url, ct);
        return await ReadAsync<T>(response, ct);
    }

    public async Task<T?> PostAsync<T>(string url, object? body, CancellationToken ct = default)
    {
        await AttachTokenAsync();
        var response = await _http.PostAsJsonAsync(url, body, JsonOptions, ct);
        return await ReadAsync<T>(response, ct);
    }

    public async Task<T?> PutAsync<T>(string url, object? body, CancellationToken ct = default)
    {
        await AttachTokenAsync();
        var response = await _http.PutAsJsonAsync(url, body, JsonOptions, ct);
        return await ReadAsync<T>(response, ct);
    }

    public async Task DeleteAsync(string url, CancellationToken ct = default)
    {
        await AttachTokenAsync();
        var response = await _http.DeleteAsync(url, ct);
        await ReadAsync<object>(response, ct);
    }

    /// <summary>Absolute URL of a download endpoint, with the token as a query parameter is not used —
    /// the caller streams the bytes through this client instead so the header is applied.</summary>
    public async Task<(byte[] Content, string FileName, string ContentType)> DownloadAsync(string url,
        CancellationToken ct = default)
    {
        await AttachTokenAsync();
        var response = await _http.GetAsync(url, ct);
        if (!response.IsSuccessStatusCode) await ReadAsync<object>(response, ct);

        var bytes = await response.Content.ReadAsByteArrayAsync(ct);
        var fileName = response.Content.Headers.ContentDisposition?.FileNameStar
                       ?? response.Content.Headers.ContentDisposition?.FileName?.Trim('"')
                       ?? "report";
        var contentType = response.Content.Headers.ContentType?.MediaType ?? "application/octet-stream";
        return (bytes, fileName, contentType);
    }

    private async Task AttachTokenAsync()
    {
        var token = await _tokens.GetTokenAsync();
        _http.DefaultRequestHeaders.Authorization = string.IsNullOrWhiteSpace(token)
            ? null
            : new System.Net.Http.Headers.AuthenticationHeaderValue("Bearer", token);
    }

    private static async Task<T?> ReadAsync<T>(HttpResponseMessage response, CancellationToken ct)
    {
        if (response.IsSuccessStatusCode)
        {
            if (response.StatusCode == HttpStatusCode.NoContent) return default;
            var raw = await response.Content.ReadAsStringAsync(ct);
            return string.IsNullOrWhiteSpace(raw) ? default : JsonSerializer.Deserialize<T>(raw, JsonOptions);
        }

        ApiErrorDto? error = null;
        try
        {
            error = await response.Content.ReadFromJsonAsync<ApiErrorDto>(JsonOptions, ct);
        }
        catch (JsonException)
        {
            // Not a structured error (e.g. a proxy page); fall through to the generic message.
        }

        throw new ApiException(response.StatusCode,
            error ?? new ApiErrorDto { Code = response.StatusCode.ToString(), Message = response.ReasonPhrase ?? "Request failed." });
    }
}

/// <summary>Carries the server's structured error to the UI so screens can show the real reason.</summary>
public class ApiException : Exception
{
    public HttpStatusCode StatusCode { get; }
    public ApiErrorDto Error { get; }

    public ApiException(HttpStatusCode statusCode, ApiErrorDto error) : base(error.Message)
    {
        StatusCode = statusCode;
        Error = error;
    }

    /// <summary>True when the save failed because someone else changed the record first.</summary>
    public bool IsConcurrencyConflict => Error.Code == "CONCURRENCY_CONFLICT";
}
