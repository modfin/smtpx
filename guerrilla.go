package guerrilla

import (
	"errors"
	"fmt"
	"github.com/phires/go-guerrilla/backends"
	"io"
	"log/slog"
	"sync"
)

const (
	// all configured servers were just been created
	daemonStateNew = iota
	// ... been started and running
	daemonStateStarted
	// ... been stopped
	daemonStateStopped
)

type Guerrilla interface {
	Start() error
	Shutdown()
	SetLogger(logger *slog.Logger)
}

type guerrilla struct {
	Config AppConfig

	server  *server
	backend backends.Backend
	// guard controls access to g.servers
	guard sync.Mutex
	state int8

	logger *slog.Logger
}

type daemonEvent func(c *AppConfig)
type serverEvent func(sc *ServerConfig)

// Get loads the log.logger in an atomic operation. Returns a stderr logger if not able to load
func (g *guerrilla) log() *slog.Logger {
	if g.logger == nil {
		return slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	return g.logger
}

func (g *guerrilla) SetLogger(logger *slog.Logger) {
	g.logger = logger
}

// Returns a new instance of Guerrilla with the given config, not yet running. Backend started.
func New(ac *AppConfig, backend backends.Backend, logger *slog.Logger) (Guerrilla, error) {
	g := &guerrilla{
		Config:  *ac, // take a local copy
		server:  &server{},
		backend: backend,
		logger:  logger,
	}

	g.state = daemonStateNew
	err := g.makeServers()
	if err != nil {
		return g, err
	}

	// start getBackend for processing email
	err = g.backend.Start()
	if err != nil {
		return g, err
	}

	return g, err
}

// Instantiate servers
func (g *guerrilla) makeServers() error {
	g.log().Debug("making servers")

	sc := g.Config.Server
	if err := sc.ValidateTLS(); err != nil {
		g.log().Error("Failed to create server", "interface", sc.ListenInterface, "err", err)
		return err
	}

	server, err := newServer(&sc, g.backend, g.log())
	if err != nil {
		g.log().Error("Failed to create server", "interface", sc.ListenInterface, "err", err)
		return err
	}

	g.server = server

	return nil
}

// setConfig sets the app config
func (g *guerrilla) setConfig(c *AppConfig) {
	g.guard.Lock()
	defer g.guard.Unlock()
	g.Config = *c
}

// setServerConfig config updates the server's config, which will update for the next connected client
func (g *guerrilla) setServerConfig(sc *ServerConfig) {
	g.guard.Lock()
	defer g.guard.Unlock()
	g.server.setConfig(sc)
}

// Entry point for the application. Starts all servers.
func (g *guerrilla) Start() error {
	var startErrors error
	g.guard.Lock()
	defer func() {
		g.state = daemonStateStarted
		g.guard.Unlock()
	}()

	if g.state == daemonStateStopped {
		// when a getBackend is shutdown, we need to re-initialize before it can be started again
		if err := g.backend.Reinitialize(); err != nil {
			startErrors = errors.Join(startErrors, fmt.Errorf("failed to reinitialize backend: %w", err))
		}
		if err := g.backend.Start(); err != nil {
			startErrors = errors.Join(startErrors, fmt.Errorf("failed to start backend: %w", err))
		}
	}

	g.log().Info("Starting server", "interface", g.server.listenInterface)
	err := g.server.Start()
	if err != nil {
		startErrors = errors.Join(startErrors, fmt.Errorf("failed to start server: %w", err))
	}

	return startErrors
}

func (g *guerrilla) Shutdown() {

	if g.server.state == ServerStateRunning {
		g.server.Shutdown()
		g.log().Info("shutdown completed", "interface", g.server.listenInterface)
	}

	g.guard.Lock()
	defer func() {
		defer g.guard.Unlock()
		g.state = daemonStateStopped
	}()

	if err := g.backend.Shutdown(); err != nil {
		g.log().Warn("Backend failed to shutdown", "err", err)
	} else {
		g.log().Info("Backend shutdown completed")
	}
}
