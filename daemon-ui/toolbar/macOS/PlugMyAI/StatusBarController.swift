import AppKit

/// Manages the NSStatusItem (menu bar icon) and its dropdown menu.
final class StatusBarController {
    private var statusItem: NSStatusItem!
    private var statusMenuItem: NSMenuItem!
    private var loginMenuItem: NSMenuItem!
    private var updateMenuItem: NSMenuItem!
    private weak var updater: UpdaterManager?

    var onQuit: (() -> Void)?

    /// Set after deferred Sparkle initialization to wire up the "Check for Updates" item.
    var updaterManager: UpdaterManager? {
        get { updater }
        set { updater = newValue }
    }

    func setup(updater: UpdaterManager?) {
        self.updater = updater

        statusItem = NSStatusBar.system.statusItem(withLength: NSStatusItem.squareLength)

        if let button = statusItem.button {
            // Use template image if available, otherwise fall back to SF Symbol
            if let icon = NSImage(named: "StatusBarIcon") {
                icon.isTemplate = true
                button.image = icon
            } else {
                let img = NSImage(systemSymbolName: "powerplug.fill",
                                  accessibilityDescription: "PlugMyAI")
                    ?? NSImage(systemSymbolName: "bolt.circle.fill",
                               accessibilityDescription: "PlugMyAI")
                button.image = img
            }
        }

        buildMenu()
    }

    /// Update the menu to reflect current daemon status.
    func update(status: DaemonStatus?) {
        if let status, status.isRunning {
            let count = status.providerCount
            statusMenuItem.title = "● Running — \(count) provider\(count == 1 ? "" : "s")"
        } else {
            statusMenuItem.title = "○ Daemon unreachable"
        }

        loginMenuItem.state = LaunchAgentManager.isInstalled ? .on : .off
        updateMenuItem.isEnabled = updater?.canCheckForUpdates ?? false
    }

    // MARK: - Private

    private func buildMenu() {
        let menu = NSMenu()

        statusMenuItem = NSMenuItem(title: "○ Checking…", action: nil, keyEquivalent: "")
        statusMenuItem.isEnabled = false
        menu.addItem(statusMenuItem)

        menu.addItem(.separator())

        let dashboardItem = NSMenuItem(title: "Open Dashboard", action: #selector(openDashboard), keyEquivalent: "d")
        dashboardItem.target = self
        menu.addItem(dashboardItem)

        updateMenuItem = NSMenuItem(title: "Check for Updates…", action: #selector(checkForUpdates), keyEquivalent: "")
        updateMenuItem.target = self
        menu.addItem(updateMenuItem)

        menu.addItem(.separator())

        loginMenuItem = NSMenuItem(title: "Start at Login", action: #selector(toggleLogin), keyEquivalent: "")
        loginMenuItem.target = self
        loginMenuItem.state = LaunchAgentManager.isInstalled ? .on : .off
        menu.addItem(loginMenuItem)

        menu.addItem(.separator())

        let quitItem = NSMenuItem(title: "Quit plug-my-ai", action: #selector(quit), keyEquivalent: "q")
        quitItem.target = self
        menu.addItem(quitItem)

        statusItem.menu = menu
    }

    @objc private func openDashboard() {
        let url = URL(string: "http://localhost:21110")!
        NSWorkspace.shared.open(url)
    }

    @objc private func checkForUpdates() {
        updater?.checkForUpdates(nil)
    }

    @objc private func toggleLogin() {
        let installed = LaunchAgentManager.toggle()
        loginMenuItem.state = installed ? .on : .off
    }

    @objc private func quit() {
        onQuit?()
    }
}
