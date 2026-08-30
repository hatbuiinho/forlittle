using System.IO;
using System.IO.Pipes;
using System.Text;
using System.Text.Json;
using System.Text.Json.Serialization;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Media;

namespace ForLittle.TimeControl.Agent;

internal static class ScheduleViewer
{
    private const string PipeName = "ForLittleTimeControl";
    private static readonly string[] Weekdays = ["Chủ nhật", "Thứ Hai", "Thứ Ba", "Thứ Tư", "Thứ Năm", "Thứ Sáu", "Thứ Bảy"];

    internal static async Task<ScheduleSnapshot> LoadAsync(CancellationToken cancellation)
    {
        AgentLog.Write("connecting to service pipe for schedule viewer");
        using var timeout = CancellationTokenSource.CreateLinkedTokenSource(cancellation);
        timeout.CancelAfter(TimeSpan.FromSeconds(10));
        using var pipe = new NamedPipeClientStream(".", PipeName, PipeDirection.InOut, PipeOptions.Asynchronous);
        await pipe.ConnectAsync(5000, timeout.Token);
        AgentLog.Write("connected to service pipe for schedule viewer");
        var utf8WithoutBom = new UTF8Encoding(encoderShouldEmitUTF8Identifier: false);
        using var reader = new StreamReader(pipe, utf8WithoutBom, leaveOpen: true);
        using var writer = new StreamWriter(pipe, utf8WithoutBom, leaveOpen: true) { AutoFlush = true };
        await writer.WriteLineAsync("{\"type\":\"REQUEST_POLICY_SCHEDULE\"}");
        AgentLog.Write("requested policy schedule from service");

        while (true)
        {
            var line = await reader.ReadLineAsync(timeout.Token);
            if (line is null) throw new IOException("service closed the schedule request");
            var snapshot = JsonSerializer.Deserialize<ScheduleSnapshot>(line);
            if (snapshot?.Type == "POLICY_SNAPSHOT" && snapshot.Policy is not null)
            {
                AgentLog.Write($"received policy schedule version={snapshot.Policy.Version}");
                return snapshot;
            }
        }
    }

    internal static Window ShowLoading(Action closeApplication)
    {
        var viewer = CreateWindow();
        viewer.Content = new StackPanel
        {
            Margin = new Thickness(28),
            Width = 420,
            Children =
            {
                new TextBlock
                {
                    Text = "📅 Lịch dùng máy",
                    FontSize = 28,
                    FontWeight = FontWeights.SemiBold,
                    Foreground = new SolidColorBrush(Color.FromRgb(31, 75, 40))
                },
                new TextBlock
                {
                    Text = "Đang tải lịch đã được áp dụng trên máy...",
                    FontSize = 16,
                    Foreground = new SolidColorBrush(Color.FromRgb(70, 82, 72)),
                    Margin = new Thickness(0, 14, 0, 4),
                    TextWrapping = TextWrapping.Wrap
                }
            }
        };
        viewer.Closed += (_, _) => closeApplication();
        viewer.Show();
        return viewer;
    }

    internal static void Show(Window viewer, ScheduleSnapshot snapshot)
    {
        var policy = snapshot.Policy!;
        var content = new StackPanel { Margin = new Thickness(28), Width = 520 };
        content.Children.Add(new TextBlock
        {
            Text = "📅 Lịch dùng máy",
            FontSize = 28,
            FontWeight = FontWeights.SemiBold,
            Foreground = new SolidColorBrush(Color.FromRgb(31, 75, 40))
        });
        content.Children.Add(new TextBlock
        {
            Text = policy.Enabled ? "Các Chú Tiểu được dùng máy trong các khung giờ sau." : "Hiện Sư Chú chưa bật lịch dùng máy.",
            FontSize = 16,
            Foreground = new SolidColorBrush(Color.FromRgb(70, 82, 72)),
            TextWrapping = TextWrapping.Wrap,
            Margin = new Thickness(0, 10, 0, 4)
        });
        content.Children.Add(new TextBlock
        {
            Text = $"Múi giờ: {policy.Timezone} · Phiên bản lịch: {policy.Version}",
            FontSize = 13,
            Foreground = new SolidColorBrush(Color.FromRgb(98, 112, 100)),
            Margin = new Thickness(0, 0, 0, 18)
        });

        if (policy.Enabled && policy.Schedule.Count > 0)
        {
            foreach (var window in policy.Schedule.OrderBy(item => item.DayOfWeek).ThenBy(item => item.StartMinutes))
            {
                content.Children.Add(new Border
                {
                    Background = new SolidColorBrush(Color.FromRgb(241, 246, 239)),
                    CornerRadius = new CornerRadius(10),
                    Padding = new Thickness(15, 11, 15, 11),
                    Margin = new Thickness(0, 0, 0, 8),
                    Child = new TextBlock
                    {
                        Text = $"{Weekdays[window.DayOfWeek]}   {FormatTime(window.StartMinutes)} - {FormatTime(window.EndMinutes)}",
                        FontSize = 17,
                        FontWeight = FontWeights.SemiBold,
                        Foreground = new SolidColorBrush(Color.FromRgb(29, 53, 33))
                    }
                });
            }
        }
        else if (policy.Enabled)
        {
            content.Children.Add(new TextBlock
            {
                Text = "Sư Chú chưa đặt khung giờ cụ thể.",
                FontSize = 16,
                Foreground = new SolidColorBrush(Color.FromRgb(120, 83, 30))
            });
        }

        if (snapshot.NextAllowedAt is not null)
        {
            content.Children.Add(new TextBlock
            {
                Text = $"Lần dùng máy tiếp theo: {FormatPolicyTime(snapshot.NextAllowedAt.Value, policy.Timezone)}",
                FontSize = 15,
                Foreground = new SolidColorBrush(Color.FromRgb(70, 82, 72)),
                Margin = new Thickness(0, 18, 0, 0)
            });
        }

        var closeButton = new Button
        {
            Content = "Đã hiểu",
            Padding = new Thickness(24, 9, 24, 9),
            Margin = new Thickness(0, 24, 0, 0),
            HorizontalAlignment = HorizontalAlignment.Right,
            Background = new SolidColorBrush(Color.FromRgb(43, 89, 48)),
            Foreground = Brushes.White,
            BorderThickness = new Thickness(0)
        };
        closeButton.Click += (_, _) => viewer.Close();
        content.Children.Add(closeButton);
        viewer.Content = content;
    }

    internal static void ShowError(Window viewer)
    {
        var content = new StackPanel { Margin = new Thickness(28), Width = 460 };
        content.Children.Add(new TextBlock
        {
            Text = "📅 Chưa tải được lịch dùng máy",
            FontSize = 25,
            FontWeight = FontWeights.SemiBold,
            Foreground = new SolidColorBrush(Color.FromRgb(121, 69, 27)),
            TextWrapping = TextWrapping.Wrap
        });
        content.Children.Add(new TextBlock
        {
            Text = "Các Chú Tiểu hãy thử lại sau ít phút. Nếu vẫn chưa được, Sư Chú có thể kiểm tra dịch vụ For Little Time Control.",
            FontSize = 16,
            Foreground = new SolidColorBrush(Color.FromRgb(70, 82, 72)),
            Margin = new Thickness(0, 14, 0, 0),
            TextWrapping = TextWrapping.Wrap
        });
        var closeButton = new Button
        {
            Content = "Đã hiểu",
            Padding = new Thickness(24, 9, 24, 9),
            Margin = new Thickness(0, 24, 0, 0),
            HorizontalAlignment = HorizontalAlignment.Right,
            Background = new SolidColorBrush(Color.FromRgb(121, 69, 27)),
            Foreground = Brushes.White,
            BorderThickness = new Thickness(0)
        };
        closeButton.Click += (_, _) => viewer.Close();
        content.Children.Add(closeButton);
        viewer.Content = content;
    }

    private static Window CreateWindow() => new()
    {
        Title = "Lịch dùng máy - For Little",
        SizeToContent = SizeToContent.WidthAndHeight,
        MinWidth = 560,
        WindowStartupLocation = WindowStartupLocation.CenterScreen,
        ResizeMode = ResizeMode.NoResize,
        Background = Brushes.White
    };

    private static string FormatTime(int minutes) => $"{minutes / 60:00}:{minutes % 60:00}";

    private static string FormatPolicyTime(DateTimeOffset value, string timezone)
    {
        try
        {
            return TimeZoneInfo.ConvertTime(value, TimeZoneInfo.FindSystemTimeZoneById(timezone)).ToString("HH:mm");
        }
        catch (TimeZoneNotFoundException) when (string.Equals(timezone, "Asia/Ho_Chi_Minh", StringComparison.OrdinalIgnoreCase))
        {
            return TimeZoneInfo.ConvertTime(value, TimeZoneInfo.FindSystemTimeZoneById("SE Asia Standard Time")).ToString("HH:mm");
        }
    }

    internal sealed record ScheduleSnapshot(
        [property: JsonPropertyName("type")] string? Type,
        [property: JsonPropertyName("next_allowed_at")] DateTimeOffset? NextAllowedAt,
        [property: JsonPropertyName("policy")] SchedulePolicy? Policy);

    internal sealed record SchedulePolicy(
        [property: JsonPropertyName("version")] int Version,
        [property: JsonPropertyName("timezone")] string Timezone,
        [property: JsonPropertyName("enabled")] bool Enabled,
        [property: JsonPropertyName("schedule")] List<ScheduleWindow> Schedule);

    internal sealed record ScheduleWindow(
        [property: JsonPropertyName("day_of_week")] int DayOfWeek,
        [property: JsonPropertyName("start_minutes")] int StartMinutes,
        [property: JsonPropertyName("end_minutes")] int EndMinutes);
}
