package main

import (
	"fmt"
	"github.com/jessevdk/go-flags"
	"github.com/spacemeshos/post/shared"
	"os"
	"runtime"
)

var (
	cfg     *config
	Version = shared.Version
)

func postMain() error {
	// Load configuration and parse command line. This function also
	// initializes logging and configures it accordingly.
	loadedConfig, err := loadConfig()
	if err != nil {
		return err
	}
	cfg = loadedConfig
	defer func() {
		if logRotator != nil {
			postLog.Info("Shutdown complete")
			_ = logRotator.Close()
		}
	}()

	// Show version at startup.
	postLog.Infof("Version: %s, logging: %s, space per unit: %v, difficulty: %v", Version(), cfg.LogLevel, cfg.Params.SpacePerUnit, cfg.Params.Difficulty)

	if err := startServer(); err != nil {
		return err
	}

	return nil
}

func main() {
	// Disable go default unbounded memory profiler.
	runtime.MemProfileRate = 0

	// Use all processor cores.
	runtime.GOMAXPROCS(runtime.NumCPU())

	// Call the "real" main in a nested manner so the defers will properly
	// be executed in the case of a graceful shutdown.
	if err := postMain(); err != nil {
		// If it's the flag utility error don't print it,
		// because it was already printed.
		if e, ok := err.(*flags.Error); ok && e.Type == flags.ErrHelp {
		} else {
			_, _ = fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(1)
	}
}
