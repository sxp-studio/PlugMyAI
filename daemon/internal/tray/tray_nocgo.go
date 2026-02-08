//go:build !cgo

package tray

import "plugmyai/internal/provider"

type Tray struct{}

func New(_ int, _ *provider.Registry, _ func()) *Tray {
	return &Tray{}
}

// Run blocks forever. Without CGO there is no system tray;
// the daemon runs headless (equivalent to --no-tray).
func (t *Tray) Run(_ func()) {
	select {}
}
