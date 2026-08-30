using System.IO;
using System.IO.Pipes;
using System.Diagnostics;
using System.Text;
using System.Text.Json;
using System.Text.Json.Serialization;
using System.Runtime.InteropServices;
using System.Security.Principal;
using System.Threading.Channels;

namespace ForLittle.TimeControl.Agent;

public sealed class PipeClient(OverlayController overlays, CancellationToken cancellation)
{
    private const string PipeName = "ForLittleTimeControl";
    private static readonly TimeSpan ReconnectDelay = TimeSpan.FromSeconds(3);
    private readonly Channel<string> refreshRequests = Channel.CreateUnbounded<string>();
    private readonly SemaphoreSlim writeLock = new(1, 1);

    public async Task RunAsync()
    {
        overlays.PolicyRefreshRequested += RequestPolicyRefresh;
        try
        {
            while (!cancellation.IsCancellationRequested)
            {
                try
                {
                    using var pipe = new NamedPipeClientStream(".", PipeName, PipeDirection.InOut, PipeOptions.Asynchronous);
                    await pipe.ConnectAsync(5000, cancellation);
                    AgentLog.Write("connected to service pipe");
                    // A BOM is valid for files but corrupts the first JSON message
                    // sent through a line-oriented named-pipe protocol.
                    var utf8WithoutBom = new UTF8Encoding(encoderShouldEmitUTF8Identifier: false);
                    using var reader = new StreamReader(pipe, utf8WithoutBom, leaveOpen: true);
                    using var writer = new StreamWriter(pipe, utf8WithoutBom, leaveOpen: true) { AutoFlush = true };
                    using var connectionCancellation = CancellationTokenSource.CreateLinkedTokenSource(cancellation);
                    var readTask = ReadMessagesAsync(reader, connectionCancellation.Token);
                    // Request the current state only after this client has set up its reader.
                    // This avoids a connect-time write race with Windows named pipes.
                    await WriteMessageAsync(writer, "{\"type\":\"AGENT_HEARTBEAT\"}", cancellation);
                    using var heartbeat = new PeriodicTimer(TimeSpan.FromSeconds(5));
                    var heartbeatTask = SendHeartbeatsAsync(writer, heartbeat, connectionCancellation.Token);
                    var refreshTask = SendRefreshRequestsAsync(writer, connectionCancellation.Token);
                    await Task.WhenAny(readTask, heartbeatTask, refreshTask);
                    connectionCancellation.Cancel();
                    try { await Task.WhenAll(readTask, heartbeatTask, refreshTask); } catch (OperationCanceledException) { }
                }
                catch (OperationCanceledException) when (cancellation.IsCancellationRequested)
                {
                    return;
                }
                catch (Exception exception)
                {
                    AgentLog.Write($"pipe connection failed: {exception}");
                }
                await Task.Delay(ReconnectDelay, cancellation);
            }
        }
        finally
        {
            overlays.PolicyRefreshRequested -= RequestPolicyRefresh;
        }
    }

    private void RequestPolicyRefresh()
    {
        AgentLog.Write("policy refresh requested from overlay");
        refreshRequests.Writer.TryWrite("{\"type\":\"REQUEST_POLICY_SYNC\"}");
    }

    private async Task ReadMessagesAsync(StreamReader reader, CancellationToken connectionCancellation)
    {
        while (!connectionCancellation.IsCancellationRequested)
        {
            var line = await reader.ReadLineAsync(connectionCancellation);
            if (line is null) return;
            var message = JsonSerializer.Deserialize<ServiceMessage>(line);
            if (message is null) continue;
            AgentLog.Write($"received {message.Type ?? "unknown"} state={message.State ?? ""}");
            if (message.Type == "FORCE_LOCK") overlays.LockWorkstation();
            else if (message.Type == "FORCE_LOGOUT") Process.Start(new ProcessStartInfo("shutdown.exe", "/l") { UseShellExecute = false });
            else if (message.Type == "STATE_CHANGED") overlays.Apply(message.State ?? "BLOCKED", message.Reason ?? "policy", message.NextAllowedAt, message.ExtendedUntil, message.Timezone);
        }
    }

    private async Task SendHeartbeatsAsync(StreamWriter writer, PeriodicTimer heartbeat, CancellationToken connectionCancellation)
    {
        var ticks = 0;
        while (await heartbeat.WaitForNextTickAsync(connectionCancellation))
        {
            await WriteMessageAsync(writer, "{\"type\":\"AGENT_HEARTBEAT\"}", connectionCancellation);
            ticks++;
            if (ticks % 3 == 0)
            {
                var sample = UsageSample.Create();
                await WriteMessageAsync(writer, JsonSerializer.Serialize(sample), connectionCancellation);
            }
        }
    }

    private async Task SendRefreshRequestsAsync(StreamWriter writer, CancellationToken connectionCancellation)
    {
        await foreach (var request in refreshRequests.Reader.ReadAllAsync(connectionCancellation))
        {
            await WriteMessageAsync(writer, request, connectionCancellation);
        }
    }

    private async Task WriteMessageAsync(StreamWriter writer, string message, CancellationToken cancellationToken)
    {
        await writeLock.WaitAsync(cancellationToken);
        try
        {
            await writer.WriteLineAsync(message);
        }
        finally
        {
            writeLock.Release();
        }
    }

    private sealed record ServiceMessage(
        [property: JsonPropertyName("type")] string? Type,
        [property: JsonPropertyName("state")] string? State,
        [property: JsonPropertyName("reason")] string? Reason,
        [property: JsonPropertyName("next_allowed_at")] DateTimeOffset? NextAllowedAt,
        [property: JsonPropertyName("extended_until")] DateTimeOffset? ExtendedUntil,
        [property: JsonPropertyName("timezone")] string? Timezone);

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
