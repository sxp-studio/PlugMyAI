import AppKit
import Sparkle

/// Wraps Sparkle's updater controller and handles pre-update daemon shutdown.
final class UpdaterManager: NSObject, SPUUpdaterDelegate {
    private lazy var controller: SPUStandardUpdaterController = {
        SPUStandardUpdaterController(
            startingUpdater: true,
            updaterDelegate: self,
            userDriverDelegate: nil
        )
    }()

    private let stopDaemon: () -> Void
    private let unloadLaunchAgent: () -> Void

    init(stopDaemon: @escaping () -> Void, unloadLaunchAgent: @escaping () -> Void) {
        self.stopDaemon = stopDaemon
        self.unloadLaunchAgent = unloadLaunchAgent
        super.init()
        // Access the lazy var to trigger Sparkle initialization
        _ = controller
    }

    /// Action target for the "Check for Updates…" menu item.
    @objc func checkForUpdates(_ sender: Any?) {
        controller.checkForUpdates(sender)
    }

    /// Whether the updater is idle and can check now.
    var canCheckForUpdates: Bool {
        controller.updater.canCheckForUpdates
    }

    // MARK: - SPUUpdaterDelegate

    func updater(
        _ updater: SPUUpdater,
        shouldPostponeRelaunchForUpdate item: SUAppcastItem,
        untilInvoking installHandler: @escaping () -> Void
    ) {
        NSLog("[Updater] Stopping daemon before update install…")
        stopDaemon()
        unloadLaunchAgent()

        // Give launchctl a moment to finish unloading before Sparkle replaces the app
        DispatchQueue.main.asyncAfter(deadline: .now() + 1) {
            NSLog("[Updater] Proceeding with install")
            installHandler()
        }
    }
}
