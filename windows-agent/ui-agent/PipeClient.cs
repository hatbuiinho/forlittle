using System.IO.Pipes;
using System.Diagnostics;
using System.Text;
using System.Text.Json;
using System.Text.Json.Serialization;
using System.Runtime.InteropServices;
using System.Security.Principal;

namespace ForLittle.TimeControl.Agent;

public sealed class PipeClient(OverlayController overlays, CancellationToken cancellation)
{
    private const string PipeName = "ForLittleTimeControl";
    private static readonly TimeSpan ReconnectDelay = TimeSpan.FromSeconds(3);

    public async Task RunAsync()
    {
        while (!cancellation.IsCancellationRequested)
        {
            try
            {
                using var pipe = new NamedPipeClientStream(".", PipeName, PipeDirection.InOut, PipeOptions.Asynchronous);
                await pipe.ConnectAsync(5000, cancellation);
                using var reader = new StreamReader(pipe, Encoding.UTF8, leaveOpen: true);
                using var writer = new StreamWriter(pipe, Encoding.UTF8, leaveOpen: true) { AutoFlush = true };
                using var heartbeat = new PeriodicTimer(TimeSpan.FromSeconds(5));
                using var connectionCancellation = CancellationTokenSource.CreateLinkedTokenSource(cancellation);
                var readTask = ReadMessagesAsync(reader, connectionCancellation.Token);
                var heartbeatTask = SendHeartbeatsAsync(writer, heartbeat, connectionCancellation.Token);
                await Task.WhenAny(readTask, heartbeatTask);
                connectionCancellation.Cancel();
                try { await Task.WhenAll(readTask, heartbeatTask); } catch (OperationCanceledException) { }
            }
            catch (OperationCanceledException) when (cancellation.IsCancellationRequested)
            {
                return;
            }
            catch
            {
                // The Service watchdog will request a scheduled-task restart if this
                // process is terminated. A short reconnect delay also handles service restarts.
            }
            await Task.Delay(ReconnectDelay, cancellation);
        }
    }

    private async Task ReadMessagesAsync(StreamReader reader, CancellationToken connectionCancellation)
    {
        while (!connectionCancellation.IsCancellationRequested)
        {
            var line = await reader.ReadLineAsync(connectionCancellation);
            if (line is null) return;
            var message = JsonSerializer.Deserialize<ServiceMessage>(line);
            if (message is null) continue;
            if (message.Type == "FORCE_LOCK") overlays.LockWorkstation();
            else if (message.Type == "FORCE_LOGOUT") Process.Start(new ProcessStartInfo("shutdown.exe", "/l") { UseShellExecute = false });
            else overlays.Apply(message.State ?? "BLOCKED", message.Reason ?? "policy", message.NextAllowedAt, message.ExtendedUntil);
        }
    }

    private async Task SendHeartbeatsAsync(StreamWriter writer, PeriodicTimer heartbeat, CancellationToken connectionCancellation)
    {
        var ticks = 0;
        while (await heartbeat.WaitForNextTickAsync(connectionCancellation))
        {
            await writer.WriteLineAsync("{\"type\":\"AGENT_HEARTBEAT\"}");
            ticks++;
            if (ticks % 3 == 0)
            {
                var sample = UsageSample.Create();
                await writer.WriteLineAsync(JsonSerializer.Serialize(sample));
            }
        }
    }

    private sealed record ServiceMessage(
        [property: JsonPropertyName("type")] string? Type,
        [property: JsonPropertyName("state")] string? State,
        [property: JsonPropertyName("reason")] string? Reason,
        [property: JsonPropertyName("next_allowed_at")] DateTimeOffset? NextAllowedAt,
        [property: JsonPropertyName("extended_until")] DateTimeOffset? ExtendedUntil);

    private sealed record UsageSample(
        [property: JsonPropertyName("type")] string Type,
        [property: JsonPropertyName("windows_user")] string WindowsUser,
        [property: JsonPropertyName("application")] string Application,
        [property: JsonPropertyName("active_seconds")] int ActiveSeconds,
        [property: JsonPropertyName("idle_seconds")] int IdleSeconds)
    {
        public static UsageSample Create()
        {
            var application = "unknown.exe";
            try
            {
                var handle = GetForegroundWindow();
                GetWindowThreadProcessId(handle, out var processId);
                if (processId > 0) application = Process.GetProcessById((int)processId).ProcessName + ".exe";
            }
            catch { }

            var input = new LastInputInfo { cbSize = (uint)Marshal.SizeOf<LastInputInfo>() };
            var idleSeconds = GetLastInputInfo(ref input)
                ? Math.Max(0, (int)((Environment.TickCount64 - input.dwTime) / 1000))
                : 0;
            var idle = idleSeconds >= 300;
            var user = WindowsIdentity.GetCurrent().Name;
            return new UsageSample("USAGE_SAMPLE", user, application, idle ? 0 : 15, idle ? 15 : 0);
        }

        [StructLayout(LayoutKind.Sequential)]
        private struct LastInputInfo { public uint cbSize; public uint dwTime; }

        [DllImport("user32.dll")]
        private static extern bool GetLastInputInfo(ref LastInputInfo info);

        [DllImport("user32.dll")]
        private static extern IntPtr GetForegroundWindow();

        [DllImport("user32.dll")]
        private static extern uint GetWindowThreadProcessId(IntPtr window, out uint processId);
    }
}
