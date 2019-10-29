/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package s2s

import (
	"sync/atomic"

	"github.com/lucas-clemente/quic-go"
	"github.com/netsec-ethz/scion-apps/lib/scionutil"
	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/transport"
	"github.com/scionproto/scion/go/lib/addr"
	"github.com/scionproto/scion/go/lib/sciond"
	"github.com/scionproto/scion/go/lib/snet"
	"github.com/scionproto/scion/go/lib/snet/squic"
	"github.com/scionproto/scion/go/lib/sock/reliable"
)

type scionServer struct {
	server
	lnQUIC         quic.Listener
	listeningSCION uint32
}

func (s *scionServer) start() {
	if s.cfg.ListenScion {
		go s.startScion()
	}
	s.server.start()
}

func (s *scionServer) startScion() {
	serverPort := uint16(s.cfg.Scion.Port)
	var address *snet.Addr
	var err error
	if s.cfg.Scion.Address == "localhost" {
		address, err = scionutil.GetLocalhost()
	} else {
		address, err = snet.AddrFromString(s.cfg.Scion.Address)
	}
	if err != nil {
		log.Fatalf("s2s_in: can't get local scion address")
	}
	address.Host.L4 = addr.NewL4UDPInfo(serverPort)

	if err := s.listenScionConn(address); err != nil {
		log.Fatalf("%v", err)
	}
	log.Infof("s2s_in: Listening for SCION s2s on port %d", serverPort)
}

func (s *scionServer) listenScionConn(address *snet.Addr) error {
	var sciondPath string
	var dispatcherPath string = "/run/shm/dispatcher/default.sock"

	sciondPath = sciond.GetDefaultSCIONDPath(nil)
	snet.Init(address.IA, sciondPath, reliable.NewDispatcherService(dispatcherPath))
	err := squic.Init(s.cfg.Scion.Key, s.cfg.Scion.Cert)
	if err != nil {
		return err
	}

	listener, err := squic.ListenSCION(nil, address, nil)
	if err != nil {
		return err
	}
	log.Infof("listening at %s", address)
	s.lnQUIC = listener
	atomic.StoreUint32(&s.listeningSCION, 1)
	for atomic.LoadUint32(&s.listeningSCION) == 1 {
		conn, err := s.lnQUIC.Accept()
		if err == nil {
			log.Infof("New SCION connection")
			accStream, err := conn.AcceptStream()
			if err != nil {
				log.Infof("No streams opened by the dialer")
			}
			go s.startInStream(transport.NewQUICSocketTransport(conn, accStream,
				s.cfg.Scion.KeepAlive))
			continue
		}
	}

	return nil
}

func (s *scionServer) startInStream(tr transport.Transport) {
	stm := newInStream(&streamConfig{
		keyGen:         &keyGen{s.cfg.DialbackSecret},
		transport:      tr,
		connectTimeout: s.cfg.ConnectTimeout,
		maxStanzaSize:  s.cfg.MaxStanzaSize,
		dialer:         s.dialer,
		onInDisconnect: s.unregisterInStream,
	}, s.mods, s.router, true)
	s.registerInStream(stm)
}
