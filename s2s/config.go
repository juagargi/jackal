/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package s2s

import (
	"crypto/tls"
	"fmt"
	"time"

	"github.com/netsec-ethz/scion-apps/lib/scionutil"
	"github.com/ortuman/jackal/module"
	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/transport"
	"github.com/ortuman/jackal/xmpp"
	"github.com/pkg/errors"
	"github.com/scionproto/scion/go/lib/sciond"
	"github.com/scionproto/scion/go/lib/sock/reliable"
)

const (
	defaultTransportPort      = 5269
	defaultScionTransportPort = 52690
	defaultTransportKeepAlive = time.Duration(10) * time.Minute
	defaultDialTimeout        = time.Duration(15) * time.Second
	defaultConnectTimeout     = time.Duration(5) * time.Second
	defaultMaxStanzaSize      = 131072
)

// TransportConfig represents s2s transport configuration.
type TransportConfig struct {
	BindAddress string
	Port        int
	KeepAlive   time.Duration
}

type transportConfigProxy struct {
	BindAddress string `yaml:"bind_addr"`
	Port        int    `yaml:"port"`
	KeepAlive   int    `yaml:"keep_alive"`
}

// UnmarshalYAML satisfies Unmarshaler interface.
func (c *TransportConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	p := transportConfigProxy{}
	if err := unmarshal(&p); err != nil {
		return err
	}
	c.BindAddress = p.BindAddress
	c.Port = p.Port
	if c.Port == 0 {
		c.Port = defaultTransportPort
	}
	if p.KeepAlive > 0 {
		c.KeepAlive = time.Duration(p.KeepAlive) * time.Second
	} else {
		c.KeepAlive = defaultTransportKeepAlive
	}
	return nil
}

type ScionConfig struct {
	Address    string
	Port       int
	Dispatcher string
	Sciond     string
	KeepAlive  time.Duration
	Key        string
	Cert       string
}

type scionConfigProxy struct {
	Address    string `yaml:"addr"`
	Port       int    `yaml:"port"`
	Dispatcher string `yaml:"dispatcher_path"`
	Sciond     string `yaml:sciond_path`
	KeepAlive  int    `yaml:"keep_alive"`
	Key        string `yaml:"privkey_path"`
	Cert       string `yaml:"cert_path"`
}

// UnmarshalYAML satisfies Unmarshaler interface.
func (c *ScionConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	p := scionConfigProxy{}
	if err := unmarshal(&p); err != nil {
		return err
	}
	c.Address = p.Address
	if len(c.Address) == 0 {
		return errors.New("s2s: specify SCION listening address")
	}
	if c.Address == "localhost" {
		address, err := scionutil.GetLocalhostString()
		if err != nil {
			return fmt.Errorf("Cannot resolve localhost: %v", err)
		}
		c.Address = address
	}
	c.Port = p.Port
	if c.Port == 0 {
		c.Port = defaultScionTransportPort
	}
	if p.KeepAlive > 0 {
		c.KeepAlive = time.Duration(p.KeepAlive) * time.Second
	} else {
		c.KeepAlive = defaultTransportKeepAlive
	}
	c.Key = p.Key
	if len(c.Key) == 0 {
		return errors.New("s2s: specify private key path")
	}
	c.Cert = p.Cert
	if len(c.Cert) == 0 {
		return errors.New("s2s: specify certificate path")
	}
	c.Dispatcher = p.Dispatcher
	if len(c.Dispatcher) == 0 {
		c.Dispatcher = reliable.DefaultDispPath
	}
	c.Sciond = p.Sciond
	if len(c.Sciond) == 0 {
		c.Sciond = sciond.DefaultSCIONDPath
	}
	return nil
}

// TLSConfig represents a server TLS configuration.
type TLSConfig struct {
	CertFile    string `yaml:"cert_path"`
	PrivKeyFile string `yaml:"privkey_path"`
}

// Config represents an s2s configuration.
type Config struct {
	ID             string
	DialTimeout    time.Duration
	ConnectTimeout time.Duration
	DialbackSecret string
	MaxStanzaSize  int
	Transport      TransportConfig
	Scion          *ScionConfig
	ListenScion    bool
}

type configProxy struct {
	ID             string          `yaml:"id"`
	DialTimeout    int             `yaml:"dial_timeout"`
	ConnectTimeout int             `yaml:"connect_timeout"`
	DialbackSecret string          `yaml:"dialback_secret"`
	MaxStanzaSize  int             `yaml:"max_stanza_size"`
	Transport      TransportConfig `yaml:"transport"`
	Scion          *ScionConfig    `yaml:"scion_transport"`
}

// UnmarshalYAML satisfies Unmarshaler interface.
func (c *Config) UnmarshalYAML(unmarshal func(interface{}) error) error {
	p := configProxy{}
	if err := unmarshal(&p); err != nil {
		return err
	}
	c.ID = p.ID
	c.DialbackSecret = p.DialbackSecret
	if len(c.DialbackSecret) == 0 {
		return errors.New("s2s.Config: must specify a dialback secret")
	}
	c.DialTimeout = time.Duration(p.DialTimeout) * time.Second
	if c.DialTimeout == 0 {
		c.DialTimeout = defaultDialTimeout
	}
	c.ConnectTimeout = time.Duration(p.ConnectTimeout) * time.Second
	if c.ConnectTimeout == 0 {
		c.ConnectTimeout = defaultConnectTimeout
	}
	c.Transport = p.Transport
	c.MaxStanzaSize = p.MaxStanzaSize
	if c.MaxStanzaSize == 0 {
		c.MaxStanzaSize = defaultMaxStanzaSize
	}
	c.Scion = p.Scion
	if c.Scion != nil {
		c.ListenScion = true
	}
	return nil
}

type streamConfig struct {
	modConfig       *module.Config
	keyGen          *keyGen
	localDomain     string
	remoteDomain    string
	connectTimeout  time.Duration
	tls             *tls.Config
	transport       transport.Transport
	maxStanzaSize   int
	dbVerify        xmpp.XElement
	dialer          *dialer
	onInDisconnect  func(s stream.S2SIn)
	onOutDisconnect func(s stream.S2SOut)
}
