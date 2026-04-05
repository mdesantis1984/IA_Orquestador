# aspire-dotnet — .NET Aspire

## Purpose
Expert guidance for .NET Aspire: cloud-native app development, service discovery, observability, configuration.

## When to Use
- Building cloud-native distributed apps
- Need integrated observability (traces, metrics, logs)
- Developing with microservices or containers

## Key Components
- **AppHost**: Orchestration and service discovery
- **ServiceDefaults**: Shared config, health checks, telemetry
- **Integrations**: Redis, PostgreSQL, RabbitMQ, etc.

## Example: AppHost
```csharp
var builder = DistributedApplication.CreateBuilder(args);

var redis = builder.AddRedis("cache");
var postgres = builder.AddPostgres("db");

builder.AddProject<Projects.ApiService>("api")
    .WithReference(redis)
    .WithReference(postgres);

builder.Build().Run();
```

## Example: Service Defaults
```csharp
// ServiceDefaults/Extensions.cs
public static IHostApplicationBuilder AddServiceDefaults(this IHostApplicationBuilder builder)
{
    builder.ConfigureOpenTelemetry();
    builder.AddDefaultHealthChecks();
    builder.Services.AddServiceDiscovery();
    return builder;
}

// In API project
builder.AddServiceDefaults();
```

## Best Practices
1. **Use AppHost** for local dev orchestration
2. **Enable telemetry** via ServiceDefaults
3. **Service discovery** for dynamic endpoints

## References
- https://learn.microsoft.com/dotnet/aspire/
