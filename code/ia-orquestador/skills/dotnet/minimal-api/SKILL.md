# minimal-api — ASP.NET Minimal APIs

## Purpose
Expert patterns for ASP.NET Minimal APIs: routing, validation, OpenAPI, dependency injection, and middleware.

## When to Use
- Building lightweight HTTP APIs
- Prefer functional style over controllers
- Rapid prototyping or microservices

## When NOT to Use
- Complex CRUD with many endpoints (consider controllers)
- Need full MVC features (views, filters)

## Key Patterns
- **Route groups**: Organize related endpoints
- **Validation**: FluentValidation, IValidatableObject
- **OpenAPI**: Swashbuckle, NSwag
- **DI**: Inject services into route handlers

## Example: Validated Endpoint
```csharp
app.MapPost("/users", async (CreateUserRequest req, IUserService userService) =>
{
    var validator = new CreateUserRequestValidator();
    var result = await validator.ValidateAsync(req);
    
    if (!result.IsValid)
        return Results.ValidationProblem(result.ToDictionary());
    
    var user = await userService.CreateUserAsync(req);
    return Results.Created($"/users/{user.Id}", user);
});

public record CreateUserRequest(string Username, string Email);

public class CreateUserRequestValidator : AbstractValidator<CreateUserRequest>
{
    public CreateUserRequestValidator()
    {
        RuleFor(x => x.Username).NotEmpty().MinimumLength(3);
        RuleFor(x => x.Email).EmailAddress();
    }
}
```

## Best Practices
1. **Use route groups** for organization
2. **Validate with FluentValidation**
3. **Document with OpenAPI**
4. **Return typed results** (Results.Ok, Results.Created)

## References
- https://learn.microsoft.com/aspnet/core/fundamentals/minimal-apis
