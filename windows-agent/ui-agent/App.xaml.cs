using System.Windows;

namespace ForLittle.TimeControl.Agent;

public partial class App : System.Windows.Application
{
    private readonly CancellationTokenSource cancellation = new();
    private readonly OverlayController overlays = new();

    protected override void OnStartup(StartupEventArgs e)
    {
        base.OnStartup(e);
        // The agent normally has no visible window. Keep its dispatcher alive
        // so it can receive named-pipe messages and show an overlay later.
        ShutdownMode = System.Windows.ShutdownMode.OnExplicitShutdown;
        _ = new PipeClient(overlays, cancellation.Token).RunAsync();
    }

    protected override void OnExit(ExitEventArgs e)
    {
        cancellation.Cancel();
        overlays.Hide();
        base.OnExit(e);
    }
}
