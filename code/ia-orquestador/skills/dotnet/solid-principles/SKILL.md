# solid-principles — SOLID Principles in C#

## Purpose
Expert application of SOLID principles in C#: SRP, OCP, LSP, ISP, DIP with practical examples.

## Principles

### 1. Single Responsibility Principle (SRP)
A class should have only one reason to change.

```csharp
// ❌ Bad: Multiple responsibilities
public class UserService
{
    public void SaveUser(User user) { /* DB logic */ }
    public void SendEmail(User user) { /* Email logic */ }
}

// ✅ Good: Separate responsibilities
public class UserRepository
{
    public void Save(User user) { /* DB logic */ }
}

public class EmailService
{
    public void SendWelcomeEmail(User user) { /* Email logic */ }
}
```

### 2. Open/Closed Principle (OCP)
Open for extension, closed for modification.

```csharp
// ✅ Use strategy pattern
public interface IDiscountStrategy
{
    decimal Calculate(decimal price);
}

public class SeasonalDiscount : IDiscountStrategy
{
    public decimal Calculate(decimal price) => price * 0.9m;
}
```

### 3. Dependency Inversion Principle (DIP)
Depend on abstractions, not concretions.

```csharp
// ✅ Inject interfaces
public class OrderService
{
    private readonly IOrderRepository _repo;
    
    public OrderService(IOrderRepository repo) => _repo = repo;
}
```

## References
- https://en.wikipedia.org/wiki/SOLID
