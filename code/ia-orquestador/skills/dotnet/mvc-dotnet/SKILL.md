# mvc-dotnet — ASP.NET MVC Patterns

## Purpose
Expert patterns for ASP.NET MVC: routing, model binding, validation, filters, areas, dependency injection.

## When to Use
- Traditional server-side rendered web apps
- Need full MVC features (views, filters, action results)

## Key Patterns

### 1. Model Binding & Validation
```csharp
public class CreateUserRequest
{
    [Required, MinLength(3)]
    public string Username { get; set; }
    
    [Required, EmailAddress]
    public string Email { get; set; }
}

[HttpPost]
public IActionResult Create(CreateUserRequest model)
{
    if (!ModelState.IsValid)
        return View(model);
    
    _userService.CreateUser(model);
    return RedirectToAction("Index");
}
```

### 2. Action Filters
```csharp
public class LogActionFilter : IActionFilter
{
    public void OnActionExecuting(ActionExecutingContext context)
    {
        Console.WriteLine($"Executing: {context.ActionDescriptor.DisplayName}");
    }
    
    public void OnActionExecuted(ActionExecutedContext context) { }
}

[ServiceFilter(typeof(LogActionFilter))]
public IActionResult Index() => View();
```

### 3. Areas for Organization
```
/Areas
  /Admin
    /Controllers
    /Views
```

## References
- https://learn.microsoft.com/aspnet/core/mvc/
