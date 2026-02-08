package tray

import (
	"fmt"
	"log"
	"os/exec"
	"runtime"

	"fyne.io/systray"

	"plugmyai/internal/provider"
)

type Tray struct {
	port     int
	registry *provider.Registry
	onQuit   func()
}

func New(port int, registry *provider.Registry, onQuit func()) *Tray {
	return &Tray{port: port, registry: registry, onQuit: onQuit}
}

// Run starts the system tray. This blocks and must be called from the main goroutine on macOS.
func (t *Tray) Run(ready func()) {
	systray.Run(func() {
		t.onReady()
		if ready != nil {
			ready()
		}
	}, t.onExit)
}

func (t *Tray) onReady() {
	systray.SetTitle("plug-my-ai")
	systray.SetTooltip("plug-my-ai — Your AI, everywhere")

	// Status
	mStatus := systray.AddMenuItem("", "")
	mStatus.Disable()
	t.updateStatus(mStatus)

	systray.AddSeparator()

	// Open Dashboard
	mDashboard := systray.AddMenuItem("Open Dashboard", "Open web dashboard in browser")

	systray.AddSeparator()

	// Quit
	mQuit := systray.AddMenuItem("Quit", "Stop plug-my-ai")

	go func() {
		for {
			select {
			case <-mDashboard.ClickedCh:
				url := fmt.Sprintf("http://localhost:%d", t.port)
				openBrowser(url)
			case <-mQuit.ClickedCh:
				if t.onQuit != nil {
					t.onQuit()
				}
				systray.Quit()
				return
			}
		}
	}()
}

func (t *Tray) updateStatus(item *systray.MenuItem) {
	avail := t.registry.Available()
	count := len(avail)
	if count == 0 {
		item.SetTitle("○ No providers available")
	} else {
		item.SetTitle(fmt.Sprintf("● Running — %d provider(s)", count))
	}
}

func (t *Tray) onExit() {
	log.Println("System tray exiting")
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	default:
		return
	}
	cmd.Start()
}
