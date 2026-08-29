using System.IO;

namespace ForLittle.TimeControl.Agent;

internal static class AgentLog
{
    private static readonly object Sync = new();
    private static readonly string LogPath = Path.Combine(
        Environment.GetFolderPath(Environment.SpecialFolder.LocalApplicationData),
        "ForLittle",
        "TimeControl",
        "agent.log");

    internal static void Write(string message)
    {
        try
        {
            lock (Sync)
            {
                Directory.CreateDirectory(Path.GetDirectoryName(LogPath)!);
                File.AppendAllText(LogPath, $"{DateTimeOffset.UtcNow:O} {message}{Environment.NewLine}");
            }
        }
        catch
        {
            // Logging must never terminate the interactive agent.
        }
    }
}
