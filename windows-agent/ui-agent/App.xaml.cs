using System.Windows;

namespace ForLittle.TimeControl.Agent;

public partial class App : Application
{
    private readonly CancellationTokenSource cancellation = new();
    private readonly OverlayController overlays = new();

    protected override void OnStartup(StartupEventArgs e)
    {
        base.OnStartup(e);
        _ = new PipeClient(overlays, cancellation.Token).RunAsync();
    }

    protected override void OnExit(ExitEventArgs e)
    {
        cancellation.Cancel();
        overlays.Hide();
        base.OnExit(e);
    }
}
