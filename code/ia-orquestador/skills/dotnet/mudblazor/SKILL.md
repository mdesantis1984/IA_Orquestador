# mudblazor — MudBlazor Component Library

## Purpose
Expert patterns for MudBlazor: theming, component usage, forms, tables, dialogs, and server-side rendering.

## When to Use
- Building Material Design UIs in Blazor
- Need rich component library (tables, forms, dialogs)
- Want consistent theming and accessibility

## Key Components
- **MudDataGrid**: High-performance data grids
- **MudForm**: Validation with FluentValidation
- **MudDialog**: Modal dialogs
- **MudTheme**: Custom themes, dark mode

## Example: Custom Theme
```csharp
// Program.cs
builder.Services.AddMudServices();

// MainLayout.razor
<MudThemeProvider Theme="@customTheme" />

@code {
    MudTheme customTheme = new()
    {
        Palette = new Palette
        {
            Primary = "#1E88E5",
            AppbarBackground = "#1E88E5"
        }
    };
}
```

## Best Practices
1. **Use MudForm** for validation
2. **Customize theme** via MudTheme
3. **Virtualize MudTable** for large datasets

## References
- https://github.com/MudBlazor/MudBlazor
