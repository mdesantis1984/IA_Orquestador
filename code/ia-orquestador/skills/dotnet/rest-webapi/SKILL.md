# rest-webapi — REST API with ASP.NET WebAPI

## Purpose
Expert REST API patterns: versioning, pagination, filtering, HATEOAS, error handling, OpenAPI.

## Key Patterns

### 1. API Versioning
```csharp
// Program.cs
builder.Services.AddApiVersioning(options =>
{
    options.DefaultApiVersion = new ApiVersion(1, 0);
    options.AssumeDefaultVersionWhenUnspecified = true;
    options.ReportApiVersions = true;
});

// Controller
[ApiController, Route("api/v{version:apiVersion}/[controller]")]
[ApiVersion("1.0")]
public class UsersController : ControllerBase
{
    [HttpGet]
    public IActionResult GetV1() => Ok("Version 1");
}

[ApiVersion("2.0")]
public class UsersV2Controller : ControllerBase
{
    [HttpGet]
    public IActionResult GetV2() => Ok("Version 2");
}
```

### 2. Pagination
```csharp
public record PagedResult<T>(IEnumerable<T> Items, int Page, int PageSize, int TotalCount);

[HttpGet]
public IActionResult Get([FromQuery] int page = 1, [FromQuery] int pageSize = 20)
{
    var users = _userService.GetUsers().Skip((page - 1) * pageSize).Take(pageSize);
    var total = _userService.GetUserCount();
    
    return Ok(new PagedResult<User>(users, page, pageSize, total));
}
```

### 3. Error Handling
```csharp
public class ErrorHandlingMiddleware
{
    public async Task InvokeAsync(HttpContext context, RequestDelegate next)
    {
        try { await next(context); }
        catch (NotFoundException ex)
        {
            context.Response.StatusCode = 404;
            await context.Response.WriteAsJsonAsync(new { error = ex.Message });
        }
    }
}
```

## References
- https://learn.microsoft.com/aspnet/core/web-api/
