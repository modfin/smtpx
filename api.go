package guerrilla

import (
	"fmt"
	"github.com/phires/go-guerrilla/backends"
	"io"
	"log/slog"
)

// Daemon provides a convenient API when using go-guerrilla as a package in your Go project.
// Is's facade for Guerrilla, AppConfig, backends.Backend and log.Logger
type Daemon struct {
	Config  *AppConfig
	Logger  *slog.Logger
	Backend backends.Backend

	// Guerrilla will be managed through the API
	g Guerrilla
}

// Starts the daemon, initializing d.Config, d.Logger and d.Backend with defaults
// can only be called once through the lifetime of the program
func (d *Daemon) Start() (err error) {
	if d.g == nil {
		if d.Config == nil {
			d.Config = &AppConfig{}
		}
		if err = d.configureDefaults(); err != nil {
			return err
		}
		if d.Logger == nil {
			d.Logger = slog.New(slog.NewTextHandler(io.Discard, nil))
		}
		if d.Backend == nil {
			return fmt.Errorf("no backend configured")
		}
		d.g, err = New(d.Config, d.Backend, d.Logger)
		if err != nil {
			return err
		}

	}
	err = d.g.Start()
	return err
}

// Shuts down the daemon, including servers and getBackend.
// Do not call Start on it again, use a new server.
func (d *Daemon) Shutdown() {
	if d.g != nil {
		d.g.Shutdown()
	}
}

// set the default values for the servers and getBackend config options
func (d *Daemon) configureDefaults() error {
	err := d.Config.setDefaults()
	if err != nil {
		return err
	}
	if d.Backend == nil {
		return fmt.Errorf("no backend configured")
	}
	return err
}
