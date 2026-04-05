# blazor-wasm — Blazor WebAssembly Patterns

## Purpose
Expert guidance for Blazor WebAssembly: offline-first PWAs, client-side state, lazy loading, service workers, and performance optimization.

## When to Use
- Offline-first or PWA requirements
- Client-heavy interactivity (no server roundtrips)
- High-latency or unreliable connections
- Static hosting (CDN, GitHub Pages)

## When NOT to Use
- Need server-side secrets or direct DB access (use Blazor Server or API backend)
- Large app size concerns (WASM payload can be >5MB)
- SEO-critical content without SSR

## Key Patterns
- **Lazy loading**: Split assemblies, load on demand
- **PWA**: Service worker for offline caching
- **State**: Use local storage, IndexedDB for persistence
- **API calls**: HttpClient to backend, handle auth with JWT

## Example: PWA with Offline Support
```csharp
// Program.cs
builder.Services.AddScoped(sp => 
    new HttpClient { BaseAddress = new Uri(builder.HostEnvironment.BaseAddress) });

// wwwroot/service-worker.js
self.addEventListener('install', event => {
    event.waitUntil(caches.open('v1').then(cache => 
        cache.addAll(['/index.html', '/css/app.css'])));
});
```

## Best Practices
1. **Lazy load assemblies** for faster initial load
2. **Use compression** (Brotli, gzip)
3. **Cache API responses** in IndexedDB
4. **Minimize payload** (trim unused assemblies)

## References
- https://learn.microsoft.com/aspnet/core/blazor/progressive-web-app
