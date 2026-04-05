# csharp-expert — Advanced C# Patterns

## Purpose
Expert C# patterns: async/await, LINQ, records, pattern matching, generics, delegates, nullable reference types.

## Key Patterns

### 1. Async/Await Best Practices
```csharp
// ✅ ConfigureAwait(false) in libraries
public async Task<User> GetUserAsync(Guid id)
{
    return await _httpClient.GetFromJsonAsync<User>($"/users/{id}")
        .ConfigureAwait(false);
}

// ✅ Avoid async void (except event handlers)
public async Task SaveDataAsync() { /* ... */ }
```

### 2. Pattern Matching (C# 11+)
```csharp
public decimal CalculateDiscount(Customer customer) => customer switch
{
    { IsPremium: true, YearsActive: > 5 } => 0.20m,
    { IsPremium: true } => 0.10m,
    { YearsActive: > 3 } => 0.05m,
    _ => 0m
};
```

### 3. Records for DTOs
```csharp
public record UserDto(Guid Id, string Username, string Email);

// With validation
public record CreateUserRequest(string Username, string Email)
{
    public CreateUserRequest : this()
    {
        if (string.IsNullOrWhiteSpace(Username))
            throw new ArgumentException(nameof(Username));
    }
}
```

### 4. Nullable Reference Types
```csharp
#nullable enable

public class UserService
{
    public User? FindUser(Guid id) => /* may return null */;
    
    public User GetUser(Guid id) => /* never null or throws */;
}
```

## References
- https://learn.microsoft.com/dotnet/csharp/
