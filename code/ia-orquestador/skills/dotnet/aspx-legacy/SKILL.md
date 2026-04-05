# aspx-legacy — ASP.NET WebForms (Legacy)

## Purpose
Guidance for maintaining and migrating ASP.NET WebForms (ASPX) applications.

## When to Use
- Maintaining legacy WebForms apps
- Planning migration to Blazor/MVC/Razor Pages

## Key Concepts
- **ViewState**: Heavy state management (avoid when possible)
- **Postbacks**: Full page refresh on every interaction
- **Server controls**: asp:Button, asp:GridView, etc.

## Migration Strategies
1. **Strangler Fig**: Incrementally replace pages with Blazor/Razor
2. **Hybrid**: Run WebForms + Blazor side-by-side
3. **Big Bang**: Full rewrite (high risk)

## Example: Strangler Fig
```csharp
// Route legacy ASPX to /legacy/, new Blazor to /
app.UseEndpoints(endpoints =>
{
    endpoints.MapBlazorHub();
    endpoints.MapFallbackToPage("/_Host"); // Blazor
    endpoints.Map("/legacy/{**path}", context => 
        context.Response.Redirect($"/WebForms/{context.Request.Path}"));
});
```

## Best Practices
1. **Avoid ViewState** (use session or client-side state)
2. **Extract business logic** to services (prepare for migration)
3. **Test heavily** before/after migration

## References
- https://learn.microsoft.com/aspnet/web-forms/
