# microservices-dotnet — .NET Microservices Architecture

## Purpose
Best practices for .NET microservices: service mesh, resilience (Polly), API gateway, health checks, observability, and service discovery.

## When to Use
- Distributed systems with multiple services
- Independent deployment and scaling per service
- Polyglot persistence (different DBs per service)

## When NOT to Use
- Small monolithic apps (YAGNI)
- Team < 3 developers (overhead not justified)

## Key Patterns
- **API Gateway**: YARP, Ocelot
- **Resilience**: Polly (circuit breaker, retry, timeout)
- **Service Discovery**: Consul, Kubernetes service mesh
- **Observability**: OpenTelemetry, Seq, Jaeger

## Example: Polly Circuit Breaker
```csharp
var retryPolicy = Policy
    .Handle<HttpRequestException>()
    .WaitAndRetryAsync(3, retryAttempt => 
        TimeSpan.FromSeconds(Math.Pow(2, retryAttempt)));

var circuitBreakerPolicy = Policy
    .Handle<HttpRequestException>()
    .CircuitBreakerAsync(5, TimeSpan.FromMinutes(1));

var policyWrap = Policy.WrapAsync(retryPolicy, circuitBreakerPolicy);
await policyWrap.ExecuteAsync(() => httpClient.GetAsync("/api/orders"));
```

## Best Practices
1. **Isolate failures** with circuit breakers
2. **Use health checks** (`/health` endpoint)
3. **Centralized logging** (Serilog + Seq)
4. **Service mesh** for complex routing

## References
- https://github.com/dotnet/eshop
- https://www.pollydocs.org/
