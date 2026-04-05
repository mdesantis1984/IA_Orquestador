# blazor-server — Blazor Server Patterns

## Purpose
Expert guidance for Blazor Server development: SignalR management, circuit handling, scoped services, state management, real-time updates, and server-side rendering patterns.

## When to Use
- Building interactive web UIs with .NET server-side logic
- Real-time dashboards, admin panels, or data-heavy apps
- When SEO is not critical (or using pre-rendering)
- When low latency to server is acceptable

## When NOT to Use
- Offline-first or PWA requirements (use Blazor WASM)
- High-latency connections (use Blazor WASM or hybrid)
- Static content sites (use SSG or Razor Pages)

## Key Concepts

### SignalR Circuit
- Persistent connection between client and server
- Handle disconnect/reconnect gracefully
- Circuit timeout configuration (default: 3 minutes)

### State Management
- **Scoped services**: Per-circuit instances (user-specific state)
- **Singleton services**: Shared across all circuits (global state, cache)
- **Transient services**: Created per DI request

### Performance
- Minimize SignalR payload: avoid sending large objects
- Use `@key` directive to preserve component identity
- Virtualize large lists (`Virtualize<T>` component)
- Pre-render for initial page load speed

## Examples

### Example 1: Scoped State Service
```csharp
// Services/UserSessionState.cs
public class UserSessionState
{
    public string UserId { get; set; }
    public Dictionary<string, object> Data { get; } = new();
}

// Program.cs
builder.Services.AddScoped<UserSessionState>();

// Component
@inject UserSessionState SessionState

@code {
    protected override void OnInitialized()
    {
        SessionState.UserId = "user-123";
    }
}
```

### Example 2: Real-Time Updates
```csharp
// Hub
public class NotificationHub : Hub
{
    public async Task SendNotification(string message)
    {
        await Clients.All.SendAsync("ReceiveNotification", message);
    }
}

// Component
@implements IAsyncDisposable
@inject NavigationManager Navigation

<ul>
    @foreach (var msg in messages)
    {
        <li>@msg</li>
    }
</ul>

@code {
    private HubConnection hubConnection;
    private List<string> messages = new();

    protected override async Task OnInitializedAsync()
    {
        hubConnection = new HubConnectionBuilder()
            .WithUrl(Navigation.ToAbsoluteUri("/notificationHub"))
            .Build();

        hubConnection.On<string>("ReceiveNotification", (message) =>
        {
            messages.Add(message);
            InvokeAsync(StateHasChanged);
        });

        await hubConnection.StartAsync();
    }

    public async ValueTask DisposeAsync()
    {
        if (hubConnection is not null)
        {
            await hubConnection.DisposeAsync();
        }
    }
}
```

### Example 3: Circuit Disconnect Handling
```csharp
// Program.cs
builder.Services.AddServerSideBlazor()
    .AddCircuitOptions(options =>
    {
        options.DisconnectedCircuitRetentionPeriod = TimeSpan.FromMinutes(5);
        options.JSInteropDefaultCallTimeout = TimeSpan.FromMinutes(1);
    });

// Component
@implements IDisposable

@code {
    protected override void OnInitialized()
    {
        CircuitHandler.OnCircuitClosed += HandleCircuitClosed;
    }

    private void HandleCircuitClosed(object sender, EventArgs e)
    {
        // Cleanup resources
    }

    public void Dispose()
    {
        CircuitHandler.OnCircuitClosed -= HandleCircuitClosed;
    }
}
```

## Best Practices
1. **Use scoped services** for per-user state (not static fields)
2. **Pre-render critical content** for faster initial load
3. **Virtualize large lists** (`Virtualize<T>`) to reduce DOM size
4. **Handle disconnects** gracefully with reconnect UI
5. **Minimize JS interop** (expensive over SignalR)
6. **Use `@key`** for dynamic lists to preserve component state

## Anti-Patterns
- ❌ Storing state in static fields (breaks multi-user scenarios)
- ❌ Long-running synchronous operations in event handlers (blocks UI)
- ❌ Large model bindings in forms (use DTOs, validate server-side)
- ❌ Ignoring circuit lifetime (memory leaks from unclosed connections)

## Gotchas
- **Circuit timeout**: Default 3 min idle → disconnects user
- **Re-rendering**: Avoid unnecessary `StateHasChanged()` calls
- **JS interop**: Async only, no return values from JS to .NET synchronously
- **AuthenticationStateProvider**: Must be scoped, not singleton

## References
- Official docs: https://learn.microsoft.com/aspnet/core/blazor/
- Blazor samples: https://github.com/dotnet/blazor-samples
- SignalR docs: https://learn.microsoft.com/aspnet/core/signalr/
