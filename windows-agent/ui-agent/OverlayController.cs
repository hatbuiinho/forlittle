using System.Runtime.InteropServices;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Media;

namespace ForLittle.TimeControl.Agent;

public sealed class OverlayController
{
    private readonly List<Window> windows = [];

    public event Action? PolicyRefreshRequested;

    public void Apply(string state, string reason, DateTimeOffset? nextAllowedAt, DateTimeOffset? extendedUntil, string? timezone)
    {
        System.Windows.Application.Current.Dispatcher.Invoke(() =>
        {
            if (state is "ALLOWED" or "EXTENDED")
            {
                Hide();
                return;
            }

            var detail = nextAllowedAt is not null
                ? $"Các Chú Tiểu có thể dùng máy lại vào lúc {FormatPolicyTime(nextAllowedAt.Value, timezone)}."
                : "Các Chú Tiểu hãy trao đổi với Sư Chú khi cần dùng máy thêm.";
            Show(detail, reason);
        });
    }

    public void LockWorkstation()
    {
        _ = NativeMethods.LockWorkStation();
    }

    public void Hide()
    {
        foreach (var window in windows) window.Close();
        windows.Clear();
    }

    private void Show(string detail, string reason)
    {
        Hide();
        // A virtual-screen overlay covers every monitor without loading the
        // Windows Forms interop layer. This is important for native ARM64 WPF.
        var window = new Window
        {
            WindowStyle = WindowStyle.None,
            ResizeMode = ResizeMode.NoResize,
            ShowInTaskbar = false,
            Topmost = true,
            Left = SystemParameters.VirtualScreenLeft,
            Top = SystemParameters.VirtualScreenTop,
            Width = SystemParameters.VirtualScreenWidth,
            Height = SystemParameters.VirtualScreenHeight,
            Background = new SolidColorBrush(System.Windows.Media.Color.FromRgb(11, 18, 12)),
            Content = CreateContent(detail, reason)
        };
        windows.Add(window);
        window.Show();
    }

    private UIElement CreateContent(string detail, string reason)
    {
        var panel = new StackPanel
        {
            Width = 620,
            VerticalAlignment = VerticalAlignment.Center,
            HorizontalAlignment = System.Windows.HorizontalAlignment.Center
        };
        panel.Children.Add(new TextBlock
        {
            Text = "🌙",
            Foreground = new SolidColorBrush(System.Windows.Media.Color.FromRgb(255, 219, 128)),
            FontSize = 54,
            FontWeight = FontWeights.Bold,
            TextAlignment = TextAlignment.Center,
            Margin = new Thickness(0, 0, 0, 8)
        });
        panel.Children.Add(new TextBlock
        {
            Text = "FOR LITTLE",
            Foreground = new SolidColorBrush(System.Windows.Media.Color.FromRgb(177, 207, 152)),
            FontSize = 15,
            FontWeight = FontWeights.Bold,
            TextAlignment = TextAlignment.Center
        });
        panel.Children.Add(new TextBlock
        {
            Text = "Đến giờ nghỉ ngơi rồi",
            Foreground = System.Windows.Media.Brushes.White,
            FontSize = 38,
            FontWeight = FontWeights.SemiBold,
            TextAlignment = TextAlignment.Center,
            Margin = new Thickness(0, 18, 0, 14),
            TextWrapping = TextWrapping.Wrap
        });
        panel.Children.Add(new TextBlock
        {
            Text = "Các Chú Tiểu hãy nghỉ ngơi, thư giãn một chút nhé.",
            Foreground = new SolidColorBrush(System.Windows.Media.Color.FromRgb(220, 226, 218)),
            FontSize = 20,
            TextAlignment = TextAlignment.Center,
            TextWrapping = TextWrapping.Wrap,
            Margin = new Thickness(0, 0, 0, 12)
        });
        panel.Children.Add(new TextBlock
        {
            Text = detail,
            Foreground = new SolidColorBrush(System.Windows.Media.Color.FromRgb(220, 226, 218)),
            FontSize = 20,
            TextAlignment = TextAlignment.Center,
            TextWrapping = TextWrapping.Wrap
        });
        panel.Children.Add(new TextBlock
        {
            Text = $"Trạng thái: {ReasonLabel(reason)}",
            Foreground = new SolidColorBrush(System.Windows.Media.Color.FromRgb(150, 166, 150)),
            FontSize = 14,
            TextAlignment = TextAlignment.Center,
            Margin = new Thickness(0, 26, 0, 0)
        });
        var refreshButton = new Button
        {
            Content = "↻ Kiểm tra lại lịch",
            FontSize = 16,
            FontWeight = FontWeights.SemiBold,
            Padding = new Thickness(20, 10, 20, 10),
            Margin = new Thickness(0, 28, 0, 0),
            HorizontalAlignment = System.Windows.HorizontalAlignment.Center,
            Background = new SolidColorBrush(System.Windows.Media.Color.FromRgb(177, 207, 152)),
            Foreground = new SolidColorBrush(System.Windows.Media.Color.FromRgb(18, 36, 22)),
            BorderThickness = new Thickness(0)
        };
        refreshButton.Click += (_, _) =>
        {
            refreshButton.IsEnabled = false;
            refreshButton.Content = "Đang kiểm tra lịch...";
            PolicyRefreshRequested?.Invoke();
        };
        panel.Children.Add(refreshButton);
        return panel;
    }

    private static string ReasonLabel(string reason) => reason switch
    {
        "outside_schedule" => "Ngoài giờ dùng máy đã được Sư Chú sắp xếp",
        "force_block" => "Sư Chú đang tạm dừng thời gian dùng máy",
        "force_lock" => "Sư Chú đã khóa máy",
        _ => "Đã đến giờ nghỉ ngơi"
    };

    private static string FormatPolicyTime(DateTimeOffset value, string? timezone)
    {
        try
        {
            if (!string.IsNullOrWhiteSpace(timezone))
            {
                var zone = TimeZoneInfo.FindSystemTimeZoneById(timezone);
                return TimeZoneInfo.ConvertTime(value, zone).ToString("HH:mm");
            }
        }
        catch (TimeZoneNotFoundException)
        {
            // Windows .NET 8 accepts IANA zone names, but retain a precise
            // fallback for older Windows installations used in this project.
            if (string.Equals(timezone, "Asia/Ho_Chi_Minh", StringComparison.OrdinalIgnoreCase))
            {
                var zone = TimeZoneInfo.FindSystemTimeZoneById("SE Asia Standard Time");
                return TimeZoneInfo.ConvertTime(value, zone).ToString("HH:mm");
            }
        }
        return value.ToString("HH:mm");
    }

    private static class NativeMethods
    {
        [DllImport("user32.dll", SetLastError = true)]
        internal static extern bool LockWorkStation();
    }
}
