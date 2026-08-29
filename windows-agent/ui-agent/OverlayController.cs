using System.Runtime.InteropServices;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Media;

namespace ForLittle.TimeControl.Agent;

public sealed class OverlayController
{
    private readonly List<Window> windows = [];

    public void Apply(string state, string reason, DateTimeOffset? nextAllowedAt, DateTimeOffset? extendedUntil)
    {
        System.Windows.Application.Current.Dispatcher.Invoke(() =>
        {
            if (state is "ALLOWED" or "EXTENDED")
            {
                Hide();
                return;
            }

            var detail = nextAllowedAt is not null
                ? $"Bạn có thể sử dụng máy tính lại vào lúc {nextAllowedAt.Value.LocalDateTime:t}."
                : "Vui lòng trao đổi với người phụ trách để được sử dụng máy tính.";
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

    private static UIElement CreateContent(string detail, string reason)
    {
        var panel = new StackPanel
        {
            Width = 620,
            VerticalAlignment = VerticalAlignment.Center,
            HorizontalAlignment = System.Windows.HorizontalAlignment.Center
        };
        panel.Children.Add(new TextBlock
        {
            Text = "For Little",
            Foreground = new SolidColorBrush(System.Windows.Media.Color.FromRgb(177, 207, 152)),
            FontSize = 16,
            FontWeight = FontWeights.Bold,
            TextAlignment = TextAlignment.Center
        });
        panel.Children.Add(new TextBlock
        {
            Text = "Đã hết thời gian sử dụng máy tính",
            Foreground = System.Windows.Media.Brushes.White,
            FontSize = 38,
            FontWeight = FontWeights.SemiBold,
            TextAlignment = TextAlignment.Center,
            Margin = new Thickness(0, 18, 0, 14),
            TextWrapping = TextWrapping.Wrap
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
        return panel;
    }

    private static string ReasonLabel(string reason) => reason switch
    {
        "outside_schedule" => "Ngoài khung giờ được phép",
        "force_block" => "Bị chặn bởi quản trị viên",
        "force_lock" => "Máy đã bị khóa bởi quản trị viên",
        _ => "Thời gian sử dụng đã kết thúc"
    };

    private static class NativeMethods
    {
        [DllImport("user32.dll", SetLastError = true)]
        internal static extern bool LockWorkStation();
    }
}
