using System.Windows;

namespace ForLittle.TimeControl.Agent;

public partial class App : System.Windows.Application
{
    private readonly CancellationTokenSource cancellation = new();
    private readonly OverlayController overlays = new();
    private Mutex? instanceMutex;
    private bool ownsInstanceMutex;

    protected override void OnStartup(StartupEventArgs e)
    {
        base.OnStartup(e);
		if (e.Args.Contains("--show-schedule", StringComparer.OrdinalIgnoreCase))
		{
			ShutdownMode = System.Windows.ShutdownMode.OnExplicitShutdown;
			_ = ShowScheduleAsync();
			return;
		}
        instanceMutex = new Mutex(true, @"Local\ForLittleTimeControlAgent", out var isFirstInstance);
        if (!isFirstInstance)
        {
            AgentLog.Write("another agent instance is already running; exiting");
            Shutdown();
            return;
        }
        ownsInstanceMutex = true;
        // The agent normally has no visible window. Keep its dispatcher alive
        // so it can receive named-pipe messages and show an overlay later.
        ShutdownMode = System.Windows.ShutdownMode.OnExplicitShutdown;
        DispatcherUnhandledException += (_, exception) =>
        {
            AgentLog.Write($"unhandled dispatcher exception: {exception.Exception}");
            exception.Handled = true;
        };
        AgentLog.Write("agent started");
        _ = new PipeClient(overlays, cancellation.Token).RunAsync();
    }

	private async Task ShowScheduleAsync()
	{
		try
		{
			var snapshot = await ScheduleViewer.LoadAsync(cancellation.Token);
			ScheduleViewer.Show(snapshot, Shutdown);
		}
		catch (Exception exception)
		{
			AgentLog.Write($"could not load schedule: {exception}");
			System.Windows.MessageBox.Show(
				"Chưa thể tải lịch dùng máy. Các Chú Tiểu hãy thử lại sau ít phút.",
				"For Little",
				System.Windows.MessageBoxButton.OK,
				System.Windows.MessageBoxImage.Information);
			Shutdown();
		}
	}

    protected override void OnExit(ExitEventArgs e)
    {
        cancellation.Cancel();
        overlays.Hide();
        if (ownsInstanceMutex) instanceMutex?.ReleaseMutex();
        instanceMutex?.Dispose();
        base.OnExit(e);
    }
}
