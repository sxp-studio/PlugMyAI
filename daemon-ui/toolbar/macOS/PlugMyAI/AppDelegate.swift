import AppKit

final class AppDelegate: NSObject, NSApplicationDelegate {
    private let statusBar = StatusBarController()
    private let daemon = DaemonManager()
    private var poller: StatusPoller?
    private var updater: UpdaterManager!

    func applicationDidFinishLaunching(_ notification: Notification) {
        // Prevent macOS from auto-terminating this LSUIElement (no-window) app.
        ProcessInfo.processInfo.disableAutomaticTermination("Menu bar daemon")
        ProcessInfo.processInfo.disableSuddenTermination()

        // Set up status bar FIRST — before anything that might crash (e.g. Sparkle).
        statusBar.setup(updater: nil)
        statusBar.onQuit = { [weak self] in self?.shutdown() }

        // Initialize Sparkle AFTER the status bar is visible.
        DispatchQueue.main.async { [self] in
            updater = UpdaterManager(
                stopDaemon: { [weak self] in
                    self?.daemon.stop()
                },
                unloadLaunchAgent: {
                    LaunchAgentManager.bootoutOnly()
                }
            )
            statusBar.updaterManager = updater
        }

        daemon.start { launched in
            if !launched {
                NSLog("[App] Warning: daemon binary not found, running in monitor-only mode")
            }
        }

        poller = StatusPoller { [weak self] status in
            self?.statusBar.update(status: status)
        }
        poller?.start()
    }

    // Handle plugmyai:// URL scheme — just launching the app is enough.
    // The website detects the daemon coming online and handles pairing via HTTP.
    func application(_ application: NSApplication, open urls: [URL]) {
        for url in urls {
            guard url.scheme == "plugmyai" else { continue }
            NSLog("[App] Received URL: %@", url.absoluteString)
            // App is now running, daemon is starting (or already running).
            // Nothing else to do — the website will detect the daemon and pair via HTTP.
        }
    }

    func applicationWillTerminate(_ notification: Notification) {
        cleanup()
    }

    private func shutdown() {
        cleanup()
        NSApp.terminate(nil)
    }

    private func cleanup() {
        poller?.stop()
        // Only stop daemon if we launched it as a subprocess (not launchd-managed)
        if daemon.isSubprocess {
            daemon.stop()
        }
    }
}
