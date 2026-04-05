# wpf-expert — Advanced WPF Patterns

## Purpose
Expert WPF: custom controls, styles, templates, triggers, resources, performance optimization.

## Key Patterns

### 1. Custom Control
```csharp
public class RoundedButton : Button
{
    static RoundedButton()
    {
        DefaultStyleKeyProperty.OverrideMetadata(typeof(RoundedButton),
            new FrameworkPropertyMetadata(typeof(RoundedButton)));
    }
    
    public static readonly DependencyProperty CornerRadiusProperty =
        DependencyProperty.Register(nameof(CornerRadius), typeof(double),
            typeof(RoundedButton), new PropertyMetadata(5.0));
    
    public double CornerRadius
    {
        get => (double)GetValue(CornerRadiusProperty);
        set => SetValue(CornerRadiusProperty, value);
    }
}
```

### 2. Data Template
```xaml
<DataTemplate x:Key="UserTemplate">
    <StackPanel>
        <TextBlock Text="{Binding Username}" FontWeight="Bold" />
        <TextBlock Text="{Binding Email}" FontSize="10" />
    </StackPanel>
</DataTemplate>

<ListBox ItemsSource="{Binding Users}" 
         ItemTemplate="{StaticResource UserTemplate}" />
```

### 3. Value Converters
```csharp
public class BoolToVisibilityConverter : IValueConverter
{
    public object Convert(object value, Type targetType, object parameter, CultureInfo culture)
        => (bool)value ? Visibility.Visible : Visibility.Collapsed;
    
    public object ConvertBack(object value, Type targetType, object parameter, CultureInfo culture)
        => (Visibility)value == Visibility.Visible;
}
```

## References
- https://learn.microsoft.com/dotnet/desktop/wpf/
