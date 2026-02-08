package server

import (
	"fmt"
	"log"
	"os/exec"
	"runtime"
	"strings"
)

// showConnectDialog shows a native macOS dialog for connection approval.
// On non-macOS platforms, falls back to opening the browser approval page.
// Runs asynchronously — returns immediately.
func showConnectDialog(appName, approveURL string, onApprove, onDeny func()) {
	if runtime.GOOS != "darwin" {
		go openBrowser(approveURL)
		return
	}

	go func() {
		escaped := strings.ReplaceAll(appName, `"`, `\"`)
		script := fmt.Sprintf(
			`display dialog "%s wants to connect to your AI" `+
				`buttons {"Deny", "Open Dashboard", "Approve"} `+
				`default button "Approve" with title "PlugMyAI"`,
			escaped)

		cmd := exec.Command("osascript", "-e", script)
		out, err := cmd.Output()
		if err != nil {
			// User pressed Cancel or dialog failed — treat as deny
			log.Printf("connect dialog error: %v", err)
			onDeny()
			return
		}

		result := string(out)
		switch {
		case strings.Contains(result, "Approve"):
			onApprove()
		case strings.Contains(result, "Open Dashboard"):
			openBrowser(approveURL)
			// Don't call onApprove or onDeny — user will decide via dashboard
		default:
			onDeny()
		}
	}()
}

// openBrowser opens the given URL in the user's default browser.
func openBrowser(url string) {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		log.Printf("Cannot open browser on %s — open manually: %s", runtime.GOOS, url)
		return
	}

	if err := cmd.Start(); err != nil {
		log.Printf("Failed to open browser: %v — open manually: %s", err, url)
	}
}
