import AppKit

/// Manages the lifecycle of the Go daemon binary as a subprocess.
/// Skips launching if the daemon is already reachable (e.g. via launchd).
final class DaemonManager {
    private var process: Process?
    private let binaryPath: String
    private let restartDelay: TimeInterval = 2
    private var shouldKeepRunning = false

    init() {
        let resourceURL = Bundle.main.resourceURL ?? Bundle.main.bundleURL
        self.binaryPath = resourceURL.appendingPathComponent("plug-my-ai").path
    }

    /// Start the daemon. Checks if it's already running first.
    /// If a LaunchAgent is installed but daemon is unreachable (e.g. after an update),
    /// re-bootstraps the agent with the new binary path.
    func start(completion: @escaping (Bool) -> Void) {
        checkDaemonReachable { [weak self] reachable in
            guard let self else { return }
            if reachable {
                NSLog("[DaemonManager] Daemon already reachable, skipping subprocess launch")
                completion(true)
                return
            }

            // Post-update: LaunchAgent plist exists but daemon was stopped for the update.
            // Re-bootstrap launchd with the (possibly new) binary path.
            if LaunchAgentManager.isInstalled {
                NSLog("[DaemonManager] LaunchAgent installed but daemon unreachable — re-bootstrapping")
                LaunchAgentManager.bootstrapIfInstalled()
                completion(true)
                return
            }

            guard FileManager.default.isExecutableFile(atPath: self.binaryPath) else {
                NSLog("[DaemonManager] Binary not found at: %@", self.binaryPath)
                completion(false)
                return
            }

            self.shouldKeepRunning = true
            self.launchProcess()
            completion(true)
        }
    }

    /// Stop the daemon subprocess.
    func stop() {
        shouldKeepRunning = false
        guard let process, process.isRunning else { return }
        process.terminate()
        self.process = nil
        NSLog("[DaemonManager] Daemon stopped")
    }

    var isSubprocess: Bool {
        process?.isRunning == true
    }

    // MARK: - Private

    private func launchProcess() {
        let proc = Process()
        proc.executableURL = URL(fileURLWithPath: binaryPath)
        proc.arguments = ["--no-tray"]

        // Inherit PATH so daemon can find claude CLI etc.
        var env = ProcessInfo.processInfo.environment
        let extraPaths = "/usr/local/bin:/opt/homebrew/bin"
        if let existing = env["PATH"] {
            env["PATH"] = "\(extraPaths):\(existing)"
        } else {
            env["PATH"] = extraPaths
        }
        proc.environment = env

        proc.terminationHandler = { [weak self] process in
            guard let self, self.shouldKeepRunning else { return }
            NSLog("[DaemonManager] Daemon exited with code %d, restarting in %.0fs", process.terminationStatus, self.restartDelay)
            DispatchQueue.main.asyncAfter(deadline: .now() + self.restartDelay) {
                if self.shouldKeepRunning {
                    self.launchProcess()
                }
            }
        }

        do {
            try proc.run()
            self.process = proc
            NSLog("[DaemonManager] Daemon launched (PID %d)", proc.processIdentifier)
        } catch {
            NSLog("[DaemonManager] Failed to launch daemon: %@", error.localizedDescription)
        }
    }

    private func checkDaemonReachable(completion: @escaping (Bool) -> Void) {
        let url = URL(string: "http://localhost:21110/v1/status")!
        let config = URLSessionConfiguration.ephemeral
        config.timeoutIntervalForRequest = 2
        let session = URLSession(configuration: config)

        session.dataTask(with: url) { _, response, _ in
            let reachable = (response as? HTTPURLResponse)?.statusCode == 200
            DispatchQueue.main.async { completion(reachable) }
        }.resume()
    }
}
