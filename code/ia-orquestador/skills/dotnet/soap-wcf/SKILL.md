# soap-wcf — SOAP/WCF in .NET

## Purpose
Guidance for SOAP/WCF services: legacy maintenance, migration to gRPC/REST, interoperability.

## When to Use
- Maintaining legacy WCF services
- SOAP interop with external systems
- Planning migration to modern APIs

## Migration Path
1. **Preferred**: Migrate to gRPC (similar contract-first approach)
2. **Alternative**: REST/WebAPI (simpler, more widely supported)
3. **Last resort**: CoreWCF (WCF on .NET Core, limited support)

## Example: Call SOAP Service
```csharp
// Add Connected Service in VS (generates client)
var client = new MyServiceClient();
var result = await client.GetDataAsync(123);
```

## Best Practices
1. **Extract business logic** from WCF services (prepare for migration)
2. **Use contracts (interfaces)** for testability
3. **Migrate incrementally** (strangler fig pattern)

## References
- https://github.com/CoreWCF/CoreWCF
