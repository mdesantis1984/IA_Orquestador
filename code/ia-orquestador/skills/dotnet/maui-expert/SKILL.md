# maui-expert — .NET MAUI Cross-Platform Apps

## Purpose
Expert .NET MAUI patterns: MVVM, platform-specific code, dependency injection, navigation, data binding.

## When to Use
- Cross-platform mobile/desktop apps (iOS, Android, Windows, macOS)
- Shared C# codebase for UI and logic

## Key Patterns

### 1. MVVM with CommunityToolkit.MVVM
```csharp
public partial class MainViewModel : ObservableObject
{
    [ObservableProperty]
    private string _title = "Hello MAUI";
    
    [RelayCommand]
    private async Task NavigateToDetails()
    {
        await Shell.Current.GoToAsync("details");
    }
}
```

### 2. Platform-Specific Code
```csharp
// Platforms/Android/MainActivity.cs
#if ANDROID
public class MainActivity : MauiAppCompatActivity
{
    protected override void OnCreate(Bundle savedInstanceState)
    {
        base.OnCreate(savedInstanceState);
        // Android-specific code
    }
}
#endif

// Or use dependency injection
public interface IDeviceService
{
    string GetPlatform();
}

// Platforms/Android/DeviceService.cs
public class DeviceService : IDeviceService
{
    public string GetPlatform() => "Android";
}

// MauiProgram.cs
#if ANDROID
builder.Services.AddSingleton<IDeviceService, Platforms.Android.DeviceService>();
#endif
```

### 3. Shell Navigation
```xaml
<Shell>
    <ShellContent Title="Home" ContentTemplate="{DataTemplate local:MainPage}" Route="main" />
    <ShellContent Title="Details" ContentTemplate="{DataTemplate local:DetailsPage}" Route="details" />
</Shell>
```

```csharp
await Shell.Current.GoToAsync("details", new Dictionary<string, object>
{
    { "UserId", userId }
});
```

## Best Practices
1. **Use Shell** for navigation
2. **CommunityToolkit.MVVM** for ViewModels
3. **Abstract platform code** behind interfaces
4. **Test on real devices** (emulators miss edge cases)

## References
- https://learn.microsoft.com/dotnet/maui/
