# onion-architecture — Onion Architecture in .NET

## Purpose
Expert guidance for Onion Architecture: domain-centric design, dependency flow inward, infrastructure at edges.

## Layers (Inside-Out)
1. **Domain Model**: Entities, value objects (no dependencies)
2. **Domain Services**: Business logic (depends on Domain Model)
3. **Application Services**: Use cases, orchestration (depends on Domain)
4. **Infrastructure**: DB, APIs, UI (depends on Application)

## Key Rule
Dependencies point INWARD. Infrastructure depends on Application, not vice versa.

## Example
```csharp
// Domain/Entities/Order.cs (innermost)
public class Order
{
    public Guid Id { get; private set; }
    public decimal Total { get; private set; }
    
    public void AddItem(OrderItem item) => Total += item.Price;
}

// Application/Services/OrderService.cs (middle)
public class OrderService
{
    private readonly IOrderRepository _repo;
    
    public async Task PlaceOrder(Guid userId, List<OrderItem> items)
    {
        var order = new Order();
        items.ForEach(order.AddItem);
        await _repo.SaveAsync(order);
    }
}

// Infrastructure/Repositories/OrderRepository.cs (outer)
public class OrderRepository : IOrderRepository
{
    private readonly DbContext _context;
    
    public async Task SaveAsync(Order order) => 
        await _context.Orders.AddAsync(order);
}
```

## References
- https://jeffreypalermo.com/2008/07/the-onion-architecture-part-1/
