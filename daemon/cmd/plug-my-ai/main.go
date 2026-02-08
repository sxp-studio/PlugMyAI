package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"plugmyai/internal/config"
	"plugmyai/internal/dashboard"
	"plugmyai/internal/provider"
	"plugmyai/internal/server"
	"plugmyai/internal/store"
	"plugmyai/internal/tray"

	// Provider self-registration via init().
	// Add new providers here as blank imports.
	_ "plugmyai/internal/provider/claude"
	_ "plugmyai/internal/provider/codex"
	_ "plugmyai/internal/provider/openaicompat"
)

func main() {
	configDir := flag.String("config", config.DefaultConfigDir(), "config directory path")
	port := flag.Int("port", 0, "port to listen on (overrides config)")
	noTray := flag.Bool("no-tray", false, "disable system tray icon")
	flag.Parse()

	// Handle init subcommand
	if flag.NArg() > 0 && flag.Arg(0) == "init" {
		runInit(*configDir)
		return
	}

	// Load config
	cfg, err := config.Load(*configDir)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	if *port > 0 {
		cfg.Port = *port
	}

	// Ensure data directory exists
	if err := os.MkdirAll(cfg.DataDir, 0700); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}

	// Initialize store
	st, err := store.New(cfg.DataDir)
	if err != nil {
		log.Fatalf("Failed to initialize store: %v", err)
	}
	defer st.Close()

	// Initialize providers
	registry := provider.NewRegistry()
	setupProviders(cfg, registry)

	// Log available providers
	for _, p := range registry.All() {
		if p.Available() {
			log.Printf("Provider ready: %s", p.Name())
		} else {
			log.Printf("Provider unavailable: %s", p.Name())
		}
	}

	// Load dashboard assets
	dashFS, err := dashboard.FS()
	if err != nil {
		log.Printf("Dashboard assets not found (run 'make dashboard' to build): %v", err)
	}

	// Create server
	srv := server.New(cfg, st, registry)

	// Signal handling for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	shutdown := func() {
		log.Println("Shutting down...")
		srv.Shutdown()
		st.Close()
		os.Exit(0)
	}

	// Start server in background
	go func() {
		if err := srv.Start(dashFS); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Handle signals in background
	go func() {
		<-sigCh
		shutdown()
	}()

	// System tray (blocks on macOS — must be on main goroutine)
	if !*noTray {
		t := tray.New(cfg.Port, registry, shutdown)
		t.Run(nil)
	} else {
		// No tray — just wait for signal
		select {}
	}
}

func setupProviders(cfg *config.Config, registry *provider.Registry) {
	for _, pc := range cfg.Providers {
		if !pc.Enabled {
			continue
		}
		p, err := provider.CreateProvider(pc.Type, pc.Config)
		if err != nil {
			log.Printf("Failed to create provider %s: %v", pc.Type, err)
			continue
		}
		registry.Register(p)
	}
}

func runInit(configDir string) {
	cfg, err := config.Load(configDir)
	if err != nil {
		log.Fatalf("Failed to initialize: %v", err)
	}

	fmt.Println("plug-my-ai initialized!")
	fmt.Printf("Config: %s/config.json\n", cfg.DataDir)
	fmt.Printf("Port: %d\n", cfg.Port)
	fmt.Printf("Admin token: %s\n", cfg.AdminToken)
	fmt.Println()
	fmt.Println("Start the daemon:")
	fmt.Println("  plug-my-ai")
	fmt.Println()
	fmt.Printf("Dashboard: http://localhost:%d\n", cfg.Port)
}
