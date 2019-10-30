/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package s2s

import (
	"context"
	"net"
	"strconv"
	"strings"

	"github.com/netsec-ethz/scion-apps/pkg/appnet"
	"github.com/ortuman/jackal/log"
	"github.com/scionproto/scion/go/lib/snet"
)

type Dialer interface {
	Dial(ctx context.Context, remoteDomain string) (net.Conn, error)
}

type srvResolveFunc func(service, proto, name string) (cname string, addrs []*net.SRV, err error)
type dialFunc func(ctx context.Context, network, address string) (net.Conn, error)

type dialer struct {
	srvResolve  srvResolveFunc
	dialContext dialFunc
}

func newDialer() *dialer {
	var d net.Dialer
	return &dialer{
		srvResolve:  net.LookupSRV,
		dialContext: d.DialContext,
	}
}

func (d *dialer) Dial(ctx context.Context, remoteDomain string) (net.Conn, error) {
	var conn net.Conn
	var err error
	isSCION, scionRAddr := scionLookup(remoteDomain)
	if isSCION {
		conn, err = d.dialQUIC(scionRAddr)
	} else {
		conn, err = d.dialTCP(ctx, remoteDomain)
	}
	return conn, err
}

func scionLookup(remoteDomain string) (bool, *snet.UDPAddr) {
	host, port, err := net.SplitHostPort(remoteDomain)
	if err != nil {
		host = remoteDomain
		port = "52690"
	}
	p, err := strconv.ParseUint(port, 10, 16)
	addr, err := appnet.ResolveUDPAddr(host + ".")
	if err != nil {
		return false, nil
	}
	addr.Host.Port = int(p)
	return true, addr
}

func (d *dialer) dialTCP(ctx context.Context, remoteDomain string) (net.Conn, error) {
	_, address, err := d.srvResolve("xmpp-server", "tcp", remoteDomain)
	if err != nil {
		log.Warnf("srv lookup error: %v", err)
	}
	var target string

	if err != nil || len(address) == 1 && address[0].Target == "." {
		target = remoteDomain + ":5269"
	} else {
		target = strings.TrimSuffix(address[0].Target, ".") + ":" + strconv.Itoa(int(address[0].Port))
	}
	conn, err := d.dialContext(ctx, "tcp", target)
	if err != nil {
		return nil, err
	}
	return conn, err
}

func (d *dialer) dialQUIC(raddr *snet.UDPAddr) (net.Conn, error) {
	return appnet.DialAddr(raddr)
}
