package guerrilla

import (
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/phires/go-guerrilla/backends"
	"log/slog"
	"os"
)

// AppConfig is the holder of the configuration of the app
type AppConfig struct {
	Log *slog.Logger

	// Server can have one or more items.
	/// Default to server listening on 127.0.0.1:2525
	Server ServerConfig `json:"servers"`

	// BackendConfig configures the email envelope processing getBackend
	BackendConfig backends.BackendConfig `json:"backend_config"`
}

// ServerConfig specifies config options for a single server
type ServerConfig struct {
	// TLS Configuration
	TLS TLS `json:"tls,omitempty"`

	// Hostname will be used in the server's reply to HELO/EHLO. If TLS enabled
	// make sure that the Hostname matches the cert. Defaults to os.Hostname()
	// Hostname will also be used to fill the 'Host' property when the "RCPT TO" address is
	// addressed to just <postmaster>
	Hostname string `json:"host_name"`
	// Listen interface specified in <ip>:<port> - defaults to 127.0.0.1:2525
	ListenInterface string `json:"listen_interface"`
	// MaxSize is the maximum size of an email that will be accepted for delivery.
	// Defaults to 10 Mebibytes
	MaxSize int64 `json:"max_size"`
	// Timeout specifies the connection timeout in seconds. Defaults to 30
	Timeout int `json:"timeout"`
	// MaxClients controls how many maximum clients we can handle at once.
	// Defaults to defaultMaxClients
	MaxClients int `json:"max_clients"`

	// XClientOn when using a proxy such as Nginx, XCLIENT command is used to pass the
	// original client's IP address & client's HELO
	XClientOn bool `json:"xclient_on,omitempty"`
	// Proxied when using a loadbalancer such as HAProxy, set to true to enable
	ProxyOn bool `json:"proxyon,omitempty"`
}

type TLSCertManager interface {
	GetCertificate(hello *tls.ClientHelloInfo) (*tls.Certificate, error)
}

type TLS struct {
	CertManager TLSCertManager
	// StartTLSOn should we offer STARTTLS command. Cert must be valid.
	// False by default
	StartTLSOn bool `json:"start_tls_on,omitempty"`
	// AlwaysOn run this server as a pure TLS server, i.e. SMTPS
	AlwaysOn bool `json:"tls_always_on,omitempty"`
}

const defaultMaxClients = 100
const defaultTimeout = 30
const defaultInterface = "127.0.0.1:2525"
const defaultMaxSize = int64(10 << 20) // 10 Mebibytes

// setDefaults fills in default server settings for values that were not configured
// The defaults are:
// * Server listening to 127.0.0.1:2525
// * use your hostname to determine your which hosts to accept email for
// * 100 maximum clients
// * 10MB max message size
// * log to Stderr,
// * log level set to "`debug`"
// * timeout to 30 sec
// * Backend configured with the following processors: `HeadersParser|Header|Debugger`
// where it will log the received emails.
func (c *AppConfig) setDefaults() error {
	if c.Log == nil {
		c.Log = slog.Default()
	}

	h, err := os.Hostname()
	if err != nil {
		return err
	}

	if c.Server.ListenInterface == "" {
		c.Server.ListenInterface = defaultInterface
	}
	if c.Server.Hostname == "" {
		c.Server.Hostname = h
	}
	if c.Server.MaxClients == 0 {
		c.Server.MaxClients = defaultMaxClients
	}
	if c.Server.Timeout == 0 {
		c.Server.Timeout = defaultTimeout
	}
	if c.Server.MaxSize == 0 {
		c.Server.MaxSize = defaultMaxSize // 10 Mebibytes
	}
	// validate the server config
	err = c.Server.ValidateTLS()
	if err != nil {
		return err
	}

	return nil
}

// ValidateTLS validates the server's configuration.
func (sc *ServerConfig) ValidateTLS() error {

	if !sc.TLS.StartTLSOn && !sc.TLS.AlwaysOn {
		return nil
	}
	var err error
	if sc.TLS.CertManager == nil {
		err = errors.Join(err, errors.New("no CertManager is set"))
	}
	if sc.TLS.CertManager != nil {
		if _, errx := sc.TLS.CertManager.GetCertificate(&tls.ClientHelloInfo{ServerName: sc.Hostname}); errx != nil {
			err = errors.Join(err, fmt.Errorf("failed to get certificate for host %s, err: %w", sc.Hostname, errx))
		}
	}

	return nil
}
