import Foundation

/// Manages the ~/Library/LaunchAgents/ plist for auto-starting the daemon on login.
struct LaunchAgentManager {
    static let label = "ai.plugmy.daemon"

    private static var plistURL: URL {
        FileManager.default.homeDirectoryForCurrentUser
            .appendingPathComponent("Library/LaunchAgents")
            .appendingPathComponent("\(label).plist")
    }

    private static var daemonPath: String {
        let resourceURL = Bundle.main.resourceURL ?? Bundle.main.bundleURL
        return resourceURL.appendingPathComponent("plug-my-ai").path
    }

    /// Whether the LaunchAgent plist is currently installed.
    static var isInstalled: Bool {
        FileManager.default.fileExists(atPath: plistURL.path)
    }

    /// Install the LaunchAgent plist so the daemon starts at login.
    static func install() throws {
        let dir = plistURL.deletingLastPathComponent()
        try FileManager.default.createDirectory(at: dir, withIntermediateDirectories: true)

        let plist: [String: Any] = [
            "Label": label,
            "ProgramArguments": [daemonPath, "--no-tray"],
            "RunAtLoad": true,
            "KeepAlive": true,
            "EnvironmentVariables": [
                "PATH": "/usr/local/bin:/usr/bin:/bin:/opt/homebrew/bin"
            ],
            "StandardOutPath": "/tmp/plug-my-ai.stdout.log",
            "StandardErrorPath": "/tmp/plug-my-ai.stderr.log",
        ]

        let data = try PropertyListSerialization.data(fromPropertyList: plist, format: .xml, options: 0)
        try data.write(to: plistURL, options: .atomic)
        NSLog("[LaunchAgent] Installed at %@", plistURL.path)
    }

    /// Remove the LaunchAgent plist and unload it.
    static func remove() throws {
        // Unload first (best effort)
        let task = Process()
        task.executableURL = URL(fileURLWithPath: "/bin/launchctl")
        task.arguments = ["bootout", "gui/\(getuid())", plistURL.path]
        try? task.run()
        task.waitUntilExit()

        try FileManager.default.removeItem(at: plistURL)
        NSLog("[LaunchAgent] Removed %@", plistURL.path)
    }

    /// Unload the agent without deleting the plist (used before updates).
    static func bootoutOnly() {
        guard isInstalled else { return }
        let task = Process()
        task.executableURL = URL(fileURLWithPath: "/bin/launchctl")
        task.arguments = ["bootout", "gui/\(getuid())", plistURL.path]
        try? task.run()
        task.waitUntilExit()
        NSLog("[LaunchAgent] Bootout only (plist preserved)")
    }

    /// Re-write the plist with the current binary path and bootstrap it.
    /// Used after an update replaces the .app bundle so launchd picks up the new binary.
    static func bootstrapIfInstalled() {
        guard isInstalled else { return }
        // Re-install to update the binary path (it may have changed if .app moved)
        try? install()
        let task = Process()
        task.executableURL = URL(fileURLWithPath: "/bin/launchctl")
        task.arguments = ["bootstrap", "gui/\(getuid())", plistURL.path]
        try? task.run()
        task.waitUntilExit()
        NSLog("[LaunchAgent] Bootstrapped with updated binary path")
    }

    /// Toggle install/remove and return the new state.
    @discardableResult
    static func toggle() -> Bool {
        if isInstalled {
            try? remove()
            return false
        } else {
            try? install()
            return true
        }
    }
}
