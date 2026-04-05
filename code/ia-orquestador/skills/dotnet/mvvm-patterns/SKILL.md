# mvvm-patterns — MVVM in WPF/MAUI

## Purpose
Expert MVVM patterns for WPF and MAUI: data binding, commands, CommunityToolkit.MVVM, dependency injection.

## Key Concepts
- **Model**: Data entities
- **View**: XAML UI
- **ViewModel**: Presentation logic, commands, INotifyPropertyChanged

## Example: CommunityToolkit.MVVM
```csharp
using CommunityToolkit.Mvvm.ComponentModel;
using CommunityToolkit.Mvvm.Input;

public partial class UserViewModel : ObservableObject
{
    [ObservableProperty]
    private string _username;
    
    [ObservableProperty]
    private string _email;
    
    [RelayCommand]
    private async Task SaveAsync()
    {
        await _userService.SaveUserAsync(new User(Username, Email));
    }
}
```

```xaml
<TextBox Text="{Binding Username}" />
<TextBox Text="{Binding Email}" />
<Button Command="{Binding SaveCommand}" Content="Save" />
```

## Best Practices
1. **Use CommunityToolkit.MVVM** (source generators)
2. **Inject services** into ViewModels
3. **Avoid code-behind** (keep logic in ViewModel)
4. **Test ViewModels** (no UI dependencies)

## References
- https://learn.microsoft.com/windows/communitytoolkit/mvvm/
