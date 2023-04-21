// Package sshpool provides pooling for SSH connections with tunneling.
package sshpool

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/sync/singleflight"
)

const (
	credentialScheme = "sftp"
	keyPathName      = "keyPath"
)

type Credentials struct {
	User     string
	Password string
	KeyPath  string
	Host     string
}

func (c Credentials) String() string {
	var query string
	if len(c.KeyPath) != 0 {
		v := url.Values{}
		v.Set(keyPathName, c.KeyPath)
		query = v.Encode()
	}
	return (&url.URL{
		Scheme:   credentialScheme,
		User:     url.UserPassword(c.User, c.Password),
		Host:     c.Host,
		RawQuery: query,
	}).String()
}

func (c *Credentials) UnmarshalText(t []byte) error {
	uri, err := url.Parse(string(t))
	if err != nil {
		return fmt.Errorf("invalid %T text: %w", c, err)
	}
	if uri.Scheme != credentialScheme {
		return fmt.Errorf("expected %T scheme %q, got %q", c, credentialScheme, uri.Scheme)
	}
	if uri.User != nil {
		c.User = uri.User.Username()
		c.Password, _ = uri.User.Password()
	}
	c.Host = uri.Host
	c.KeyPath = uri.Query().Get(keyPathName)
	return nil
}

type Cmd struct {
	Command string
	Args    []string
	Env     []string
	Stdin   io.Reader
	Stdout  io.Writer
	Stderr  io.Writer
}

type conn struct {
	mu      sync.Mutex
	ssh     *ssh.Client
	tunnels map[string]struct{}
}

func (c *conn) useTunnel(tunnel string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	_, ok := c.tunnels[tunnel]
	if ok {
		return false
	}
	if c.tunnels == nil {
		c.tunnels = map[string]struct{}{}
	}
	c.tunnels[tunnel] = struct{}{}
	return true
}

// Pool maintains SSH connections through SSH tunnels.
type Pool struct {
	// MaxTunnelConns limits the number of allowed tunnels per single SSH connection.
	MaxTunnelConns int
	// KeepAliveTime sets the maximum duration before unused SSH connections are closed.
	// Zero value allows them to stay open forever.
	KeepAliveTime time.Duration
	mu            sync.RWMutex
	sf            singleflight.Group
	conns         map[string][]*conn
}

// Session returns an SSH session for the last hop, connected through SSH tunnels of
// all previous hops. Minimum of 1 hop is required, which uses direct SSH connection.
func (p *Pool) Session(hops []Credentials) (*ssh.Client, error) {
	return p.session(hops, "")
}

func (p *Pool) session(hops []Credentials, tunnel string) (*ssh.Client, error) {
	count := len(hops)
	if count == 0 {
		return nil, errors.New("no provided hops")
	}
	key := makeKey(hops)
	if tun, ok := p.takeConn(key, tunnel); ok {
		return tun.ssh, nil
	}
	tunnels, hop := hops[:count-1], hops[count-1]
	dialer := net.Dial
	var tun *ssh.Client
	var err error
	if len(tunnels) != 0 {
		tun, err = p.session(tunnels, hop.String())
		if err != nil {
			return nil, err
		}
		dialer = tun.Dial
	}
	con, err, _ := p.sf.Do(key, func() (any, error) {
		if tun, ok := p.takeConn(key, tunnel); ok || err != nil {
			return tun.ssh, nil // race condition
		}
		netConn, err := dialer("tcp", hop.Host)
		if err != nil {
			return nil, err
		}
		con, err := authConn(netConn, hop, tunnel, tun)
		if err != nil {
			return nil, err
		}
		p.putConn(key, con)
		return con.ssh, nil
	})
	return con.(*ssh.Client), err
}

func readKeyFile(tun *ssh.Client, keyPath string) (ssh.Signer, error) {
	if tun == nil {
		pem, err := os.ReadFile(keyPath)
		if err != nil {
			return nil, err
		}
		return ssh.ParsePrivateKey(pem)
	}
	sess, err := tun.NewSession()
	if err != nil {
		return nil, err
	}
	key, err := sess.CombinedOutput(fmt.Sprintf("cat %s", escapeCmdArg(keyPath)))
	if err != nil {
		return nil, err
	}
	return ssh.ParsePrivateKey(key)
}

var (
	pattern  = regexp.MustCompile(`[^\w@%+=:,./-]`)
	replacer = strings.NewReplacer("'", "'\"'\"'")
)

func escapeCmdArg(s string) string {
	if len(s) == 0 {
		return "''"
	}

	if pattern.MatchString(s) {
		return "'" + replacer.Replace(s) + "'"
	}

	return s
}

func authConn(c net.Conn, hop Credentials, tunnel string, tun *ssh.Client) (*conn, error) {
	var auth []ssh.AuthMethod
	if len(hop.Password) != 0 {
		auth = append(auth, ssh.Password(hop.Password))
	}
	if len(hop.KeyPath) != 0 {
		signer, err := readKeyFile(tun, hop.KeyPath)
		if err != nil {
			return nil, err
		}
		auth = append(auth, ssh.PublicKeys(signer))
	}
	conf := &ssh.ClientConfig{
		User:            hop.User,
		Auth:            auth,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         20 * time.Second,
	}
	sshCon, chans, reqs, err := ssh.NewClientConn(c, hop.Host, conf)
	if err != nil {
		return nil, err
	}
	con := &conn{ssh: ssh.NewClient(sshCon, chans, reqs)}
	if len(tunnel) != 0 {
		con.tunnels = map[string]struct{}{tunnel: {}}
	}
	return con, nil
}

func makeKey(hops []Credentials) string {
	var keyBuilder strings.Builder
	for i, hop := range hops {
		if i > 0 {
			keyBuilder.WriteByte('\n')
		}
		keyBuilder.WriteString(hop.String())
	}
	return keyBuilder.String()
}

func (p *Pool) takeConn(key, tunnel string) (*conn, bool) {
	p.mu.RLock()
	conns := p.conns[key]
	p.mu.RUnlock()
	if len(conns) == 0 {
		return nil, false
	} else if len(tunnel) == 0 {
		return conns[0], true
	}
	var minConn *conn
	for _, c := range conns {
		if (minConn == nil || len(c.tunnels) < len(minConn.tunnels)) && len(minConn.tunnels) < p.MaxTunnelConns {
			minConn = c
		}
	}
	if minConn == nil {
		return nil, false
	}
	if !minConn.useTunnel(tunnel) {
		return nil, false
	}
	return minConn, true
}

func (p *Pool) putConn(key string, con *conn) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.conns == nil {
		p.conns = map[string][]*conn{}
	}
	p.conns[key] = append(p.conns[key], con)
}

func (p *Pool) Close(con *ssh.Client) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	for key, keyConns := range p.conns {
		for i, keyCon := range keyConns {
			if con != keyCon.ssh {
				continue
			}
			keyConns[i], keyConns[len(keyConns)-1] = keyConns[len(keyConns)-1], keyConns[i]
			keyConns = keyConns[:len(keyConns)-1]
			if len(keyConns) == 0 {
				delete(p.conns, key)
			}
			return keyCon.ssh.Close()
		}
	}
	return con.Close()
}
