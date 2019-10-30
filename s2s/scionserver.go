/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package s2s

import (
	"context"
	"sync/atomic"

	"github.com/lucas-clemente/quic-go"
	"github.com/netsec-ethz/scion-apps/pkg/appnet"
	"github.com/netsec-ethz/scion-apps/pkg/appnet/appquic"
	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/transport"
	"github.com/scionproto/scion/go/lib/snet/squic"
)

type scionServer struct {
	server
	lnQUIC         quic.Listener
	listeningSCION uint32
}

func (s *scionServer) start() {
	if s.cfg.Scion != nil {
		go s.startScion()
	}
	s.server.start()
}

func (s *scionServer) startScion() {
	serverPort := uint16(s.cfg.Scion.Port)
	// TODO(juagargi) check this
	// TODO(juagargi) use appnet and quit if scion is not available? -> use appnet whenever possible
	// address, err := snet.ParseUDPAddr(s.cfg.Scion.Address)
	// if err != nil {
	// 	log.Fatalf("s2s_in: can't get local scion address")
	// }
	// address.Host.Port = int(serverPort)

	if err := s.listenScionConn(serverPort); err != nil {
		log.Fatalf("%v", err)
	}
	log.Infof("s2s_in: Listening for SCION s2s on port %d", serverPort)
}

func (s *scionServer) listenScionConn(port uint16) error {
	_ = appnet.DefNetwork() // we don't have control over the errors: the library will quit
	err := squic.Init(s.cfg.Scion.Key, s.cfg.Scion.Cert)
	if err != nil {
		return err
	}
	listener, err := appquic.ListenPort(port, nil, nil)
	if err != nil {
		return err
	}
	log.Infof("listening at %s", listener.Addr())
	s.lnQUIC = listener
	atomic.StoreUint32(&s.listeningSCION, 1)
	for atomic.LoadUint32(&s.listeningSCION) == 1 {
		sess, err := s.lnQUIC.Accept(context.TODO())
		if err == nil {
			log.Infof("New SCION connection")
			accStream, err := sess.AcceptStream(context.TODO())
			if err != nil {
				log.Infof("No streams opened by the dialer")
			}
			go s.startInStream(transport.NewQUICSocketTransport(sess, accStream,
				s.cfg.Scion.Compress))
			continue
		}
	}

	return nil
}

func (s *scionServer) startInStream(tr transport.Transport) {
	stm := newInStream(
		&inConfig{
			keyGen:         &keyGen{s.cfg.DialbackSecret},
			connectTimeout: s.cfg.ConnectTimeout,
			keepAlive:      s.cfg.KeepAlive,
			timeout:        s.cfg.Timeout,
			maxStanzaSize:  s.cfg.MaxStanzaSize,
			onDisconnect:   s.unregisterInStream,
		},
		tr,
		s.mods,
		s.newOutFn,
		s.router,
		true,
	)
	s.registerInStream(stm)
}
