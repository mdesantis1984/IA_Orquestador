# dotnet-architecture — Clean Architecture in .NET

## Purpose
Expert guidance for Clean Architecture: separation of concerns, dependency inversion, domain-driven design, CQRS with MediatR.

## When to Use
- Complex business logic requiring testability
- Domain-driven design (DDD)
- Long-term maintainability critical

## Layers
1. **Core/Domain**: Entities, value objects, interfaces (no dependencies)
2. **Application**: Use cases, DTOs, MediatR handlers
3. **Infrastructure**: EF Core, external APIs, file system
4. **WebUI**: Controllers, Razor Pages, Blazor

## Example: CQRS with MediatR
```csharp
// Application/Users/Queries/GetUserQuery.cs
public record GetUserQuery(Guid Id) : IRequest<UserDto>;

public class GetUserQueryHandler : IRequestHandler<GetUserQuery, UserDto>
{
    private readonly IApplicationDbContext _context;
    
    public async Task<UserDto> Handle(GetUserQuery request, CancellationToken ct)
    {
        var user = await _context.Users.FindAsync(request.Id);
        return new UserDto(user.Id, user.Username);
    }
}

// WebUI/Controllers/UsersController.cs
[ApiController, Route("[controller]")]
public class UsersController : ControllerBase
{
    private readonly IMediator _mediator;
    
    [HttpGet("{id}")]
    public async Task<UserDto> Get(Guid id) => 
        await _mediator.Send(new GetUserQuery(id));
}
```

## References
- https://github.com/jasontaylordev/CleanArchitecture
