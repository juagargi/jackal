/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package transport

import (
	"bufio"
	"crypto/tls"

	"github.com/lucas-clemente/quic-go"
)

type quicSocketTransport struct {
	socketTransport
	conn quic.Session
}

// NewQUICSocketTransport create and return a new quicSocketTransport.
func NewQUICSocketTransport(conn quic.Session, uniStream quic.Stream,
	compress bool) Transport {
	s := &quicSocketTransport{
		socketTransport: socketTransport{
			rw:         uniStream,
			br:         bufio.NewReaderSize(uniStream, socketBuffSize),
			bw:         bufio.NewWriterSize(uniStream, socketBuffSize),
			compressed: compress,
		},
		conn: conn,
	}
	return s
}

func (s *quicSocketTransport) Read(p []byte) (n int, err error) {
	n, err = s.br.Read(p)
	return n, err
}

func (s *quicSocketTransport) StartTLS(cfg *tls.Config, asClient bool) {
}
