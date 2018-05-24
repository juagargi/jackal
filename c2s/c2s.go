/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package c2s

import (
	"bytes"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/ortuman/jackal/auth"
	"github.com/ortuman/jackal/errors"
	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/module"
	"github.com/ortuman/jackal/module/offline"
	"github.com/ortuman/jackal/module/roster"
	"github.com/ortuman/jackal/module/xep0012"
	"github.com/ortuman/jackal/module/xep0030"
	"github.com/ortuman/jackal/module/xep0049"
	"github.com/ortuman/jackal/module/xep0054"
	"github.com/ortuman/jackal/module/xep0077"
	"github.com/ortuman/jackal/module/xep0092"
	"github.com/ortuman/jackal/module/xep0191"
	"github.com/ortuman/jackal/module/xep0199"
	"github.com/ortuman/jackal/router"
	"github.com/ortuman/jackal/server/compress"
	"github.com/ortuman/jackal/server/transport"
	"github.com/ortuman/jackal/storage"
	"github.com/ortuman/jackal/storage/model"
	"github.com/ortuman/jackal/xml"
	"github.com/pborman/uuid"
)

const streamMailboxSize = 64

const (
	connecting uint32 = iota
	connected
	authenticating
	authenticated
	sessionStarted
	disconnected
)

const (
	jabberClientNamespace     = "jabber:client"
	framedStreamNamespace     = "urn:ietf:params:xml:ns:xmpp-framing"
	streamNamespace           = "http://etherx.jabber.org/streams"
	tlsNamespace              = "urn:ietf:params:xml:ns:xmpp-tls"
	compressProtocolNamespace = "http://jabber.org/protocol/compress"
	bindNamespace             = "urn:ietf:params:xml:ns:xmpp-bind"
	sessionNamespace          = "urn:ietf:params:xml:ns:xmpp-session"
	saslNamespace             = "urn:ietf:params:xml:ns:xmpp-sasl"
	blockedErrorNamespace     = "urn:xmpp:blocking:errors"
)

// stream context keys
const (
	usernameCtxKey      = "username"
	domainCtxKey        = "domain"
	resourceCtxKey      = "resource"
	jidCtxKey           = "jid"
	securedCtxKey       = "secured"
	authenticatedCtxKey = "authenticated"
	compressedCtxKey    = "compressed"
	presenceCtxKey      = "presence"
)

// once context keys
const (
	rosterOnceCtxKey  = "rosterOnce"
	offlineOnceCtxKey = "offlineOnce"
)

type stream struct {
	cfg         *Config
	tlsCfg      *tls.Config
	tr          transport.Transport
	parser      *xml.Parser
	id          string
	connectTm   *time.Timer
	state       uint32
	ctx         *router.Context
	authrs      []auth.Authenticator
	activeAuthr auth.Authenticator
	iqHandlers  []module.IQHandler
	roster      *roster.Roster
	discoInfo   *xep0030.DiscoInfo
	register    *xep0077.Register
	ping        *xep0199.Ping
	blockCmd    *xep0191.BlockingCommand
	offline     *offline.Offline
	actorCh     chan func()
}

func New(id string, tr transport.Transport, tlsCfg *tls.Config, cfg *Config) router.C2S {
	s := &stream{
		cfg:     cfg,
		tlsCfg:  tlsCfg,
		id:      id,
		tr:      tr,
		parser:  xml.NewParser(tr, cfg.MaxStanzaSize),
		state:   connecting,
		ctx:     router.NewContext(),
		actorCh: make(chan func(), streamMailboxSize),
	}
	// initialize stream context
	secured := !(tr.Type() == transport.Socket)
	s.ctx.SetBool(secured, securedCtxKey)

	domain := router.Instance().DefaultLocalDomain()
	s.ctx.SetString(domain, domainCtxKey)

	j, _ := xml.NewJID("", domain, "", true)
	s.ctx.SetObject(j, jidCtxKey)

	// initialize authenticators
	s.initializeAuthenticators()

	// initialize register module
	if _, ok := s.cfg.Modules.Enabled["registration"]; ok {
		s.register = xep0077.New(&s.cfg.Modules.Registration, s, s.discoInfo)
	}

	if cfg.ConnectTimeout > 0 {
		s.connectTm = time.AfterFunc(time.Duration(cfg.ConnectTimeout)*time.Second, s.connectTimeout)
	}
	go s.actorLoop()
	go s.doRead() // start reading transport...

	return s
}

// ID returns stream identifier.
func (s *stream) ID() string {
	return s.id
}

// Context returns stream associated context.
func (s *stream) Context() *router.Context {
	return s.ctx
}

// Username returns current stream username.
func (s *stream) Username() string {
	return s.ctx.String(usernameCtxKey)
}

// Domain returns current stream domain.
func (s *stream) Domain() string {
	return s.ctx.String(domainCtxKey)
}

// Resource returns current stream resource.
func (s *stream) Resource() string {
	return s.ctx.String(resourceCtxKey)
}

// JID returns current user JID.
func (s *stream) JID() *xml.JID {
	return s.ctx.Object(jidCtxKey).(*xml.JID)
}

// IsAuthenticated returns whether or not the XMPP stream
// has successfully authenticated.
func (s *stream) IsAuthenticated() bool {
	return s.ctx.Bool(authenticatedCtxKey)
}

// IsSecured returns whether or not the XMPP stream
// has been secured using SSL/TLS.
func (s *stream) IsSecured() bool {
	return s.ctx.Bool(securedCtxKey)
}

// IsCompressed returns whether or not the XMPP stream
// has enabled a compression method.
func (s *stream) IsCompressed() bool {
	return s.ctx.Bool(compressedCtxKey)
}

// Presence returns last sent presence element.
func (s *stream) Presence() *xml.Presence {
	switch v := s.ctx.Object(presenceCtxKey).(type) {
	case *xml.Presence:
		return v
	}
	return nil
}

// SendElement sends the given XML element.
func (s *stream) SendElement(element xml.XElement) {
	s.actorCh <- func() {
		s.writeElement(element)
	}
}

// Disconnect disconnects remote peer by closing
// the underlying TCP socket connection.
func (s *stream) Disconnect(err error) {
	s.actorCh <- func() {
		s.disconnect(err)
	}
}

func (s *stream) initializeAuthenticators() {
	for _, a := range s.cfg.SASL {
		switch a {
		case "plain":
			s.authrs = append(s.authrs, auth.NewPlain(s))

		case "digest_md5":
			s.authrs = append(s.authrs, auth.NewDigestMD5(s))

		case "scram_sha_1":
			s.authrs = append(s.authrs, auth.NewScram(s, s.tr, auth.ScramSHA1, false))
			s.authrs = append(s.authrs, auth.NewScram(s, s.tr, auth.ScramSHA1, true))

		case "scram_sha_256":
			s.authrs = append(s.authrs, auth.NewScram(s, s.tr, auth.ScramSHA256, false))
			s.authrs = append(s.authrs, auth.NewScram(s, s.tr, auth.ScramSHA256, true))
		}
	}
}

func (s *stream) initializeModules() {
	// XEP-0030: Service Discovery (https://xmpp.org/extensions/xep-0030.html)
	s.discoInfo = xep0030.New(s)
	s.iqHandlers = append(s.iqHandlers, s.discoInfo)

	// register default disco info entities
	s.discoInfo.RegisterDefaultEntities()

	// Roster (https://xmpp.org/rfcs/rfc3921.html#roster)
	s.roster = roster.New(&s.cfg.Modules.Roster, s)
	s.iqHandlers = append(s.iqHandlers, s.roster)

	// XEP-0012: Last Activity (https://xmpp.org/extensions/xep-0012.html)
	if _, ok := s.cfg.Modules.Enabled["last_activity"]; ok {
		s.iqHandlers = append(s.iqHandlers, xep0012.New(s, s.discoInfo))
	}

	// XEP-0049: Private XML Storage (https://xmpp.org/extensions/xep-0049.html)
	if _, ok := s.cfg.Modules.Enabled["private"]; ok {
		s.iqHandlers = append(s.iqHandlers, xep0049.New(s))
	}

	// XEP-0054: vcard-temp (https://xmpp.org/extensions/xep-0054.html)
	if _, ok := s.cfg.Modules.Enabled["vcard"]; ok {
		s.iqHandlers = append(s.iqHandlers, xep0054.New(s, s.discoInfo))
	}

	// XEP-0077: In-band registration (https://xmpp.org/extensions/xep-0077.html)
	if s.register != nil {
		s.iqHandlers = append(s.iqHandlers, s.register)
	}

	// XEP-0092: Software Version (https://xmpp.org/extensions/xep-0092.html)
	if _, ok := s.cfg.Modules.Enabled["version"]; ok {
		s.iqHandlers = append(s.iqHandlers, xep0092.New(&s.cfg.Modules.Version, s, s.discoInfo))
	}

	// XEP-0191: Blocking Command (https://xmpp.org/extensions/xep-0191.html)
	if _, ok := s.cfg.Modules.Enabled["blocking_command"]; ok {
		s.blockCmd = xep0191.New(s, s.discoInfo)
		s.iqHandlers = append(s.iqHandlers, s.blockCmd)
	}

	// XEP-0199: XMPP Ping (https://xmpp.org/extensions/xep-0199.html)
	if _, ok := s.cfg.Modules.Enabled["ping"]; ok {
		s.ping = xep0199.New(&s.cfg.Modules.Ping, s, s.discoInfo)
		s.iqHandlers = append(s.iqHandlers, s.ping)
	}

	// XEP-0160: Offline message storage (https://xmpp.org/extensions/xep-0160.html)
	if _, ok := s.cfg.Modules.Enabled["offline"]; ok {
		s.offline = offline.New(&s.cfg.Modules.Offline, s, s.discoInfo)
	}
}

func (s *stream) connectTimeout() {
	s.actorCh <- func() {
		s.disconnect(streamerror.ErrConnectionTimeout)
	}
}

func (s *stream) handleElement(elem xml.XElement) {
	isWebSocketTr := s.tr.Type() == transport.WebSocket
	if isWebSocketTr && elem.Name() == "close" && elem.Namespace() == framedStreamNamespace {
		s.disconnect(nil)
		return
	}
	switch s.getState() {
	case connecting:
		s.handleConnecting(elem)
	case connected:
		s.handleConnected(elem)
	case authenticated:
		s.handleAuthenticated(elem)
	case authenticating:
		s.handleAuthenticating(elem)
	case sessionStarted:
		s.handleSessionStarted(elem)
	default:
		break
	}
}

func (s *stream) handleConnecting(elem xml.XElement) {
	// cancel connection timeout timer
	if s.connectTm != nil {
		s.connectTm.Stop()
		s.connectTm = nil
	}

	// validate stream element
	if err := s.validateStreamElement(elem); err != nil {
		s.disconnectWithStreamError(err)
		return
	}
	// assign stream domain
	s.ctx.SetString(elem.To(), domainCtxKey)

	// open stream
	s.openStream()

	features := xml.NewElementName("stream:features")
	features.SetAttribute("xmlns:stream", streamNamespace)
	features.SetAttribute("version", "1.0")

	if !s.IsAuthenticated() {
		features.AppendElements(s.unauthenticatedFeatures())
		s.setState(connected)
	} else {
		features.AppendElements(s.authenticatedFeatures())
		s.setState(authenticated)
	}
	s.writeElement(features)
}

func (s *stream) unauthenticatedFeatures() []xml.XElement {
	var features []xml.XElement

	isSocketTransport := s.tr.Type() == transport.Socket

	if isSocketTransport && !s.IsSecured() {
		startTLS := xml.NewElementName("starttls")
		startTLS.SetNamespace("urn:ietf:params:xml:ns:xmpp-tls")
		startTLS.AppendElement(xml.NewElementName("required"))
		features = append(features, startTLS)
	}

	// attach SASL mechanisms
	shouldOfferSASL := (!isSocketTransport || (isSocketTransport && s.IsSecured()))

	if shouldOfferSASL && len(s.authrs) > 0 {
		mechanisms := xml.NewElementName("mechanisms")
		mechanisms.SetNamespace(saslNamespace)
		for _, athr := range s.authrs {
			mechanism := xml.NewElementName("mechanism")
			mechanism.SetText(athr.Mechanism())
			mechanisms.AppendElement(mechanism)
		}
		features = append(features, mechanisms)
	}

	// allow In-band registration over encrypted stream only
	allowRegistration := s.IsSecured()

	if _, ok := s.cfg.Modules.Enabled["registration"]; ok && allowRegistration {
		registerFeature := xml.NewElementNamespace("register", "http://jabber.org/features/iq-register")
		features = append(features, registerFeature)
	}
	return features
}

func (s *stream) authenticatedFeatures() []xml.XElement {
	var features []xml.XElement

	isSocketTransport := s.tr.Type() == transport.Socket

	// attach compression feature
	compressionAvailable := isSocketTransport && s.cfg.Compression.Level != compress.NoCompression

	if !s.IsCompressed() && compressionAvailable {
		compression := xml.NewElementNamespace("compression", "http://jabber.org/features/compress")
		method := xml.NewElementName("method")
		method.SetText("zlib")
		compression.AppendElement(method)
		features = append(features, compression)
	}
	bind := xml.NewElementNamespace("bind", "urn:ietf:params:xml:ns:xmpp-bind")
	bind.AppendElement(xml.NewElementName("required"))
	features = append(features, bind)

	session := xml.NewElementNamespace("session", "urn:ietf:params:xml:ns:xmpp-session")
	features = append(features, session)

	if s.roster != nil && s.cfg.Modules.Roster.Versioning {
		ver := xml.NewElementNamespace("ver", "urn:xmpp:features:rosterver")
		features = append(features, ver)
	}
	return features
}

func (s *stream) handleConnected(elem xml.XElement) {
	switch elem.Name() {
	case "starttls":
		if len(elem.Namespace()) > 0 && elem.Namespace() != tlsNamespace {
			s.disconnectWithStreamError(streamerror.ErrInvalidNamespace)
			return
		}
		s.proceedStartTLS()

	case "auth":
		if elem.Namespace() != saslNamespace {
			s.disconnectWithStreamError(streamerror.ErrInvalidNamespace)
			return
		}
		s.startAuthentication(elem)

	case "iq":
		stanza, err := s.buildStanza(elem, false)
		if err != nil {
			s.handleElementError(elem, err)
			return
		}
		iq := stanza.(*xml.IQ)

		if s.register != nil && s.register.MatchesIQ(iq) {
			s.register.ProcessIQ(iq)
			return

		} else if iq.Elements().ChildNamespace("query", "jabber:iq:auth") != nil {
			// don't allow non-SASL authentication
			s.writeElement(iq.ServiceUnavailableError())
			return
		}
		fallthrough

	case "message", "presence":
		s.disconnectWithStreamError(streamerror.ErrNotAuthorized)

	default:
		s.disconnectWithStreamError(streamerror.ErrUnsupportedStanzaType)
	}
}

func (s *stream) handleAuthenticating(elem xml.XElement) {
	if elem.Namespace() != saslNamespace {
		s.disconnectWithStreamError(streamerror.ErrInvalidNamespace)
		return
	}
	authr := s.activeAuthr
	s.continueAuthentication(elem, authr)
	if authr.Authenticated() {
		s.finishAuthentication(authr.Username())
	}
}

func (s *stream) handleAuthenticated(elem xml.XElement) {
	switch elem.Name() {
	case "compress":
		if elem.Namespace() != compressProtocolNamespace {
			s.disconnectWithStreamError(streamerror.ErrUnsupportedStanzaType)
			return
		}
		s.compress(elem)

	case "iq":
		stanza, err := s.buildStanza(elem, true)
		if err != nil {
			s.handleElementError(elem, err)
			return
		}
		iq := stanza.(*xml.IQ)

		if len(s.Resource()) == 0 { // expecting bind
			s.bindResource(iq)
		} else { // expecting session
			s.startSession(iq)
		}

	default:
		s.disconnectWithStreamError(streamerror.ErrUnsupportedStanzaType)
	}
}

func (s *stream) handleSessionStarted(elem xml.XElement) {
	// reset ping timer deadline
	if s.ping != nil {
		s.ping.ResetDeadline()
	}

	stanza, err := s.buildStanza(elem, true)
	if err != nil {
		s.handleElementError(elem, err)
		return
	}
	if s.isComponentDomain(stanza.ToJID().Domain()) {
		s.processComponentStanza(stanza)
	} else {
		s.processStanza(stanza)
	}
}

func (s *stream) proceedStartTLS() {
	if s.IsSecured() {
		s.disconnectWithStreamError(streamerror.ErrNotAuthorized)
		return
	}
	s.ctx.SetBool(true, securedCtxKey)

	s.writeElement(xml.NewElementNamespace("proceed", tlsNamespace))

	s.tr.StartTLS(s.tlsCfg)

	log.Infof("secured stream... id: %s", s.id)

	s.restart()
}

func (s *stream) compress(elem xml.XElement) {
	if s.IsCompressed() {
		s.disconnectWithStreamError(streamerror.ErrUnsupportedStanzaType)
		return
	}
	method := elem.Elements().Child("method")
	if method == nil || len(method.Text()) == 0 {
		failure := xml.NewElementNamespace("failure", compressProtocolNamespace)
		failure.AppendElement(xml.NewElementName("setup-failed"))
		s.writeElement(failure)
		return
	}
	if method.Text() != "zlib" {
		failure := xml.NewElementNamespace("failure", compressProtocolNamespace)
		failure.AppendElement(xml.NewElementName("unsupported-method"))
		s.writeElement(failure)
		return
	}
	s.ctx.SetBool(true, compressedCtxKey)

	s.writeElement(xml.NewElementNamespace("compressed", compressProtocolNamespace))

	s.tr.EnableCompression(s.cfg.Compression.Level)

	log.Infof("compressed stream... id: %s", s.id)

	s.restart()
}

func (s *stream) startAuthentication(elem xml.XElement) {
	mechanism := elem.Attributes().Get("mechanism")
	for _, authr := range s.authrs {
		if authr.Mechanism() == mechanism {
			if err := s.continueAuthentication(elem, authr); err != nil {
				return
			}
			if authr.Authenticated() {
				s.finishAuthentication(authr.Username())
			} else {
				s.activeAuthr = authr
				s.setState(authenticating)
			}
			return
		}
	}

	// ...mechanism not found...
	failure := xml.NewElementNamespace("failure", saslNamespace)
	failure.AppendElement(xml.NewElementName("invalid-mechanism"))
	s.writeElement(failure)
}

func (s *stream) continueAuthentication(elem xml.XElement, authr auth.Authenticator) error {
	err := authr.ProcessElement(elem)
	if saslErr, ok := err.(*auth.SASLError); ok {
		s.failAuthentication(saslErr.Element())
	} else if err != nil {
		log.Error(err)
		s.failAuthentication(auth.ErrSASLTemporaryAuthFailure.(*auth.SASLError).Element())
	}
	return err
}

func (s *stream) finishAuthentication(username string) {
	if s.activeAuthr != nil {
		s.activeAuthr.Reset()
		s.activeAuthr = nil
	}
	j, _ := xml.NewJID(username, s.Domain(), "", true)

	s.ctx.SetString(username, usernameCtxKey)
	s.ctx.SetBool(true, authenticatedCtxKey)
	s.ctx.SetObject(j, jidCtxKey)

	s.restart()
}

func (s *stream) failAuthentication(elem xml.XElement) {
	failure := xml.NewElementNamespace("failure", saslNamespace)
	failure.AppendElement(elem)
	s.writeElement(failure)

	if s.activeAuthr != nil {
		s.activeAuthr.Reset()
		s.activeAuthr = nil
	}
	s.setState(connected)
}

func (s *stream) bindResource(iq *xml.IQ) {
	bind := iq.Elements().ChildNamespace("bind", bindNamespace)
	if bind == nil {
		s.writeElement(iq.NotAllowedError())
		return
	}
	var resource string
	if resourceElem := bind.Elements().Child("resource"); resourceElem != nil {
		resource = resourceElem.Text()
	} else {
		resource = uuid.New()
	}
	// try binding...
	var stm router.C2S
	stms := router.Instance().StreamsMatchingJID(s.JID().ToBareJID())
	for _, s := range stms {
		if s.Resource() == resource {
			stm = s
		}
	}

	if stm != nil {
		switch s.cfg.ResourceConflict {
		case Override:
			// override the resource with a server-generated resourcepart...
			h := sha256.New()
			h.Write([]byte(s.ID()))
			resource = hex.EncodeToString(h.Sum(nil))
		case Replace:
			// terminate the session of the currently connected client...
			stm.Disconnect(streamerror.ErrResourceConstraint)
		default:
			// disallow resource binding attempt...
			s.writeElement(iq.ConflictError())
			return
		}
	}
	userJID, err := xml.NewJID(s.Username(), s.Domain(), resource, false)
	if err != nil {
		s.writeElement(iq.BadRequestError())
		return
	}
	s.ctx.SetString(resource, resourceCtxKey)
	s.ctx.SetObject(userJID, jidCtxKey)

	log.Infof("binded resource... (%s/%s)", s.Username(), s.Resource())

	//...notify successful binding
	result := xml.NewIQType(iq.ID(), xml.ResultType)
	result.SetNamespace(iq.Namespace())

	binded := xml.NewElementNamespace("bind", bindNamespace)
	jid := xml.NewElementName("jid")
	jid.SetText(s.Username() + "@" + s.Domain() + "/" + s.Resource())
	binded.AppendElement(jid)
	result.AppendElement(binded)

	s.writeElement(result)

	if err := router.Instance().AuthenticateStream(s); err != nil {
		log.Error(err)
	}
}

func (s *stream) startSession(iq *xml.IQ) {
	if len(s.Resource()) == 0 {
		// not binded yet...
		s.Disconnect(streamerror.ErrNotAuthorized)
		return
	}
	sess := iq.Elements().ChildNamespace("session", sessionNamespace)
	if sess == nil {
		s.writeElement(iq.NotAllowedError())
		return
	}
	s.writeElement(iq.ResultIQ())

	// initialize modules
	s.initializeModules()

	if s.ping != nil {
		s.ping.StartPinging()
	}
	s.setState(sessionStarted)
}

func (s *stream) processStanza(stanza xml.Stanza) {
	toJID := stanza.ToJID()
	if s.isBlockedJID(toJID) { // blocked JID?
		blocked := xml.NewElementNamespace("blocked", blockedErrorNamespace)
		resp := xml.NewErrorElementFromElement(stanza, xml.ErrNotAcceptable.(*xml.StanzaError), []xml.XElement{blocked})
		s.writeElement(resp)
		return
	}
	switch stanza := stanza.(type) {
	case *xml.Presence:
		s.processPresence(stanza)
	case *xml.IQ:
		s.processIQ(stanza)
	case *xml.Message:
		s.processMessage(stanza)
	}
}

func (s *stream) processComponentStanza(stanza xml.Stanza) {
}

func (s *stream) processIQ(iq *xml.IQ) {
	toJID := iq.ToJID()
	if !router.Instance().IsLocalDomain(toJID.Domain()) {
		// TODO(ortuman): Implement XMPP federation
		return
	}
	if node := toJID.Node(); len(node) > 0 && router.Instance().IsBlockedJID(s.JID(), node) {
		// destination user blocked stream JID
		if iq.IsGet() || iq.IsSet() {
			s.writeElement(iq.ServiceUnavailableError())
		}
		return
	}
	if toJID.IsFullWithUser() {
		switch router.Instance().Route(iq) {
		case router.ErrResourceNotFound:
			s.writeElement(iq.ServiceUnavailableError())
		}
		return
	}

	for _, handler := range s.iqHandlers {
		if !handler.MatchesIQ(iq) {
			continue
		}
		handler.ProcessIQ(iq)
		return
	}

	// ...IQ not handled...
	if iq.IsGet() || iq.IsSet() {
		s.writeElement(iq.ServiceUnavailableError())
	}
}

func (s *stream) processPresence(presence *xml.Presence) {
	toJID := presence.ToJID()
	if !router.Instance().IsLocalDomain(toJID.Domain()) {
		// TODO(ortuman): Implement XMPP federation
		return
	}
	if toJID.IsBare() && (toJID.Node() != s.Username() || toJID.Domain() != s.Domain()) {
		if s.roster != nil {
			s.roster.ProcessPresence(presence)
		}
		return
	}
	if toJID.IsFullWithUser() {
		router.Instance().Route(presence)
		return
	}
	// set context presence
	s.ctx.SetObject(presence, presenceCtxKey)

	// deliver pending approval notifications
	if s.roster != nil {
		if !s.ctx.Bool(rosterOnceCtxKey) {
			s.roster.DeliverPendingApprovalNotifications()
			s.roster.ReceivePresences()
			s.ctx.SetBool(true, rosterOnceCtxKey)
		}
		s.roster.BroadcastPresence(presence)
	}

	// deliver offline messages
	if p := s.Presence(); s.offline != nil && p != nil && p.Priority() >= 0 {
		if !s.ctx.Bool(offlineOnceCtxKey) {
			s.offline.DeliverOfflineMessages()
			s.ctx.SetBool(true, offlineOnceCtxKey)
		}
	}
}

func (s *stream) processMessage(message *xml.Message) {
	toJID := message.ToJID()
	if !router.Instance().IsLocalDomain(toJID.Domain()) {
		// TODO(ortuman): Implement XMPP federation
		return
	}

sendMessage:
	err := router.Instance().Route(message)
	switch err {
	case nil:
		break
	case router.ErrNotAuthenticated:
		if s.offline != nil {
			if (message.IsChat() || message.IsGroupChat()) && message.IsMessageWithBody() {
				return
			}
			s.offline.ArchiveMessage(message)
		}
	case router.ErrResourceNotFound:
		// treat the stanza as if it were addressed to <node@domain>
		toJID = toJID.ToBareJID()
		goto sendMessage
	case router.ErrNotExistingAccount, router.ErrBlockedJID:
		s.writeElement(message.ServiceUnavailableError())
	default:
		log.Error(err)
	}
}

func (s *stream) actorLoop() {
	for {
		f := <-s.actorCh
		f()
		if s.getState() == disconnected {
			return
		}
	}
}

func (s *stream) doRead() {
	if elem, err := s.parser.ParseElement(); err == nil {
		s.actorCh <- func() {
			s.readElement(elem)
		}
	} else {
		if s.getState() == disconnected {
			return // already disconnected...
		}

		var discErr error
		switch err {
		case nil, io.EOF, io.ErrUnexpectedEOF:
			break

		case xml.ErrStreamClosedByPeer: // ...received </stream:stream>
			if s.tr.Type() != transport.Socket {
				discErr = streamerror.ErrInvalidXML
			}

		case xml.ErrTooLargeStanza:
			discErr = streamerror.ErrPolicyViolation

		default:
			switch e := err.(type) {
			case net.Error:
				if e.Timeout() {
					discErr = streamerror.ErrConnectionTimeout
				} else {
					discErr = streamerror.ErrInvalidXML
				}

			case *websocket.CloseError:
				break // connection closed by peer...

			default:
				log.Error(err)
				discErr = streamerror.ErrInvalidXML
			}
		}
		s.actorCh <- func() {
			s.disconnect(discErr)
		}
	}
}

func (s *stream) writeElement(element xml.XElement) {
	log.Debugf("SEND: %v", element)
	s.tr.WriteElement(element, true)
}

func (s *stream) readElement(elem xml.XElement) {
	if elem != nil {
		log.Debugf("RECV: %v", elem)
		s.handleElement(elem)
	}
	if s.getState() != disconnected {
		go s.doRead()
	}
}

func (s *stream) disconnect(err error) {
	switch err {
	case nil:
		s.disconnectClosingStream(false)
	default:
		if strmErr, ok := err.(*streamerror.Error); ok {
			s.disconnectWithStreamError(strmErr)
		} else {
			log.Error(err)
			s.disconnectClosingStream(false)
		}
	}
}

func (s *stream) openStream() {
	var ops *xml.Element
	var includeClosing bool

	buf := &bytes.Buffer{}
	switch s.tr.Type() {
	case transport.Socket:
		ops = xml.NewElementName("stream:stream")
		ops.SetAttribute("xmlns", jabberClientNamespace)
		ops.SetAttribute("xmlns:stream", streamNamespace)
		buf.WriteString(`<?xml version="1.0"?>`)

	case transport.WebSocket:
		ops = xml.NewElementName("open")
		ops.SetAttribute("xmlns", framedStreamNamespace)
		includeClosing = true

	default:
		return
	}
	ops.SetAttribute("id", uuid.New())
	ops.SetAttribute("from", s.Domain())
	ops.SetAttribute("version", "1.0")
	ops.ToXML(buf, includeClosing)

	openStr := buf.String()
	log.Debugf("SEND: %s", openStr)

	s.tr.WriteString(buf.String())
}

func (s *stream) buildStanza(elem xml.XElement, validateFrom bool) (xml.Stanza, error) {
	if err := s.validateNamespace(elem); err != nil {
		return nil, err
	}
	fromJID, toJID, err := s.extractAddresses(elem, validateFrom)
	if err != nil {
		return nil, err
	}
	switch elem.Name() {
	case "iq":
		iq, err := xml.NewIQFromElement(elem, fromJID, toJID)
		if err != nil {
			log.Error(err)
			return nil, xml.ErrBadRequest
		}
		return iq, nil

	case "presence":
		presence, err := xml.NewPresenceFromElement(elem, fromJID, toJID)
		if err != nil {
			log.Error(err)
			return nil, xml.ErrBadRequest
		}
		return presence, nil

	case "message":
		message, err := xml.NewMessageFromElement(elem, fromJID, toJID)
		if err != nil {
			log.Error(err)
			return nil, xml.ErrBadRequest
		}
		return message, nil
	}
	return nil, streamerror.ErrUnsupportedStanzaType
}

func (s *stream) handleElementError(elem xml.XElement, err error) {
	if streamErr, ok := err.(*streamerror.Error); ok {
		s.disconnectWithStreamError(streamErr)
	} else if stanzaErr, ok := err.(*xml.StanzaError); ok {
		s.writeElement(xml.NewErrorElementFromElement(elem, stanzaErr, nil))
	} else {
		log.Error(err)
	}
}

func (s *stream) validateStreamElement(elem xml.XElement) *streamerror.Error {
	switch s.tr.Type() {
	case transport.Socket:
		if elem.Name() != "stream:stream" {
			return streamerror.ErrUnsupportedStanzaType
		}
		if elem.Namespace() != jabberClientNamespace || elem.Attributes().Get("xmlns:stream") != streamNamespace {
			return streamerror.ErrInvalidNamespace
		}

	case transport.WebSocket:
		if elem.Name() != "open" {
			return streamerror.ErrUnsupportedStanzaType
		}
		if elem.Namespace() != framedStreamNamespace {
			return streamerror.ErrInvalidNamespace
		}
	}
	to := elem.To()
	if len(to) > 0 && !router.Instance().IsLocalDomain(to) {
		return streamerror.ErrHostUnknown
	}
	if elem.Version() != "1.0" {
		return streamerror.ErrUnsupportedVersion
	}
	return nil
}

func (s *stream) validateNamespace(elem xml.XElement) *streamerror.Error {
	ns := elem.Namespace()
	if len(ns) == 0 || ns == jabberClientNamespace {
		return nil
	}
	return streamerror.ErrInvalidNamespace
}

func (s *stream) extractAddresses(elem xml.XElement, validateFrom bool) (fromJID *xml.JID, toJID *xml.JID, err error) {
	// validate from JID
	from := elem.From()
	if validateFrom && len(from) > 0 && !s.isValidFrom(from) {
		return nil, nil, streamerror.ErrInvalidFrom
	}
	fromJID = s.JID()

	// validate to JID
	to := elem.To()
	if len(to) > 0 {
		toJID, err = xml.NewJIDString(elem.To(), false)
		if err != nil {
			return nil, nil, xml.ErrJidMalformed
		}
	} else {
		toJID = s.JID().ToBareJID() // account's bare JID as default 'to'
	}
	return
}

func (s *stream) isValidFrom(from string) bool {
	validFrom := false
	j, err := xml.NewJIDString(from, false)
	if err == nil && j != nil {
		node := j.Node()
		domain := j.Domain()
		resource := j.Resource()

		userJID := s.JID()
		validFrom = node == userJID.Node() && domain == userJID.Domain()
		if len(resource) > 0 {
			validFrom = validFrom && resource == userJID.Resource()
		}
	}
	return validFrom
}

func (s *stream) isComponentDomain(domain string) bool {
	return false
}

func (s *stream) disconnectWithStreamError(err *streamerror.Error) {
	if s.getState() == connecting {
		s.openStream()
	}
	s.writeElement(err.Element())
	s.disconnectClosingStream(true)
}

func (s *stream) disconnectClosingStream(closeStream bool) {
	if err := s.updateLogoutInfo(); err != nil {
		log.Error(err)
	}
	if presence := s.Presence(); presence != nil && presence.IsAvailable() && s.roster != nil {
		s.roster.BroadcastPresenceAndWait(xml.NewPresence(s.JID(), s.JID(), xml.UnavailableType))
	}
	if closeStream {
		switch s.tr.Type() {
		case transport.Socket:
			s.tr.WriteString("</stream:stream>")
		case transport.WebSocket:
			s.tr.WriteString(fmt.Sprintf(`<close xmlns="%s" />`, framedStreamNamespace))
		}
	}
	// signal termination...
	s.ctx.Terminate()

	// unregister stream
	if err := router.Instance().UnregisterStream(s); err != nil {
		log.Error(err)
	}
	s.setState(disconnected)
	s.tr.Close()
}

func (s *stream) updateLogoutInfo() error {
	var usr *model.User
	var err error
	if presence := s.Presence(); presence != nil {
		if usr, err = storage.Instance().FetchUser(s.Username()); usr != nil && err == nil {
			usr.LoggedOutAt = time.Now()
			if presence.IsUnavailable() {
				usr.LoggedOutStatus = presence.Status()
			}
			return storage.Instance().InsertOrUpdateUser(usr)
		}
	}
	return err
}

func (s *stream) isBlockedJID(jid *xml.JID) bool {
	if jid.IsServer() && router.Instance().IsLocalDomain(jid.Domain()) {
		return false
	}
	return router.Instance().IsBlockedJID(jid, s.Username())
}

func (s *stream) restart() {
	s.parser = xml.NewParser(s.tr, s.cfg.MaxStanzaSize)
	s.setState(connecting)
}

func (s *stream) setState(state uint32) {
	atomic.StoreUint32(&s.state, state)
}

func (s *stream) getState() uint32 {
	return atomic.LoadUint32(&s.state)
}
