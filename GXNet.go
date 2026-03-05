package gxnet

// --------------------------------------------------------------------------
//
//	Gurux Ltd
//
// Filename:        $HeadURL$
//
// Version:         $Revision$,
//
//	$Date$
//	$Author$
//
// # Copyright (c) Gurux Ltd
//
// ---------------------------------------------------------------------------
//
//	DESCRIPTION
//
// This file is a part of Gurux Device Framework.
//
// Gurux Device Framework is Open Source software; you can redistribute it
// and/or modify it under the terms of the GNU General Public License
// as published by the Free Software Foundation; version 2 of the License.
// Gurux Device Framework is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.
// See the GNU General Public License for more details.
//
// More information of Gurux products: https://www.gurux.org
//
// This code is licensed under the GNU General Public License v2.
// Full text may be retrieved at http://www.gnu.org/licenses/gpl-2.0.txt
// ---------------------------------------------------------------------------

import (
	"bytes"
	"encoding/binary"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Gurux/gxcommon-go"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

// GXNet implements gxcommon.IGXMedia over TCP or UDP sockets.
//
// Configure connection settings (host, port, protocol, timeouts), then call
// Open. Data can be sent with Send and received either asynchronously via
// callbacks or synchronously via Receive.
type GXNet struct {
	// Protocol selects TCP or UDP transport.
	Protocol NetworkType
	// HostName is the remote DNS name or IP address.
	HostName string
	// Port is the remote TCP/UDP port number.
	Port int

	// Connection timeout in milliseconds.
	timeout time.Duration
	eop     any
	// UseIPv6 enables IPv6 sockets. The default is false (IPv4).
	UseIPv6 bool

	// The trace level specifies which types of trace messages are emitted.
	traceLevel gxcommon.TraceLevel
	// OnReceived: Media component notifies asynchronously received data through this method.
	mu   sync.RWMutex
	conn net.Conn
	wg   sync.WaitGroup

	stop        chan struct{}
	synchronous bool

	bytesSent     uint64
	bytesReceived uint64

	//Called when the Media state is changed.
	onState gxcommon.MediaStateHandler

	//Called when the new data is received.
	onReceive gxcommon.ReceivedEventHandler

	//Called when the Media is sending or receiving data.
	onTrace gxcommon.TraceEventHandler

	//Called when the Media is sending or receiving data.
	onErr gxcommon.ErrorEventHandler

	//Sync settings.
	receivedSize int
	received     synchronousMediaBase

	// Printer for localized messages.
	p *message.Printer
}

// NewGXNet returns a GXNet initialized with the given protocol, host, and port.
//
// The default timeout is 10 seconds.
func NewGXNet(protocol NetworkType, hostName string, port int) *GXNet {
	g := &GXNet{Protocol: protocol, HostName: hostName, Port: port,
		stop: make(chan struct{}), timeout: time.Duration(10000) * time.Millisecond}
	g.received = *newGXSynchronousMediaBase()
	g.p = message.NewPrinter(gxcommon.Language())
	return g
}

// String returns the endpoint in "host:port" format.
func (g *GXNet) String() string {
	return fmt.Sprintf("%s:%s", g.HostName, strconv.Itoa(g.Port))
}

// GetName returns the endpoint in "host:port" format.
func (g *GXNet) GetName() string {
	return fmt.Sprintf("%s:%s", g.HostName, strconv.Itoa(g.Port))
}

// IsOpen reports whether the network connection is currently open.
func (g *GXNet) IsOpen() bool {
	return g.conn != nil
}

// Copy copies configurable connection settings to another media instance.
//
// The target must be *GXNet.
func (g *GXNet) Copy(target gxcommon.IGXMedia) error {
	switch dst := target.(type) {
	case *GXNet:
		dst.timeout = g.timeout
		dst.Protocol = g.Protocol
		dst.HostName = g.HostName
		dst.Port = g.Port
		dst.traceLevel = g.traceLevel
		dst.eop = g.eop
	default:
		return fmt.Errorf("copy: target is %T; want *GXNet", target)
	}
	return nil
}

// GetMediaType returns the media type identifier used by Gurux.
func (g *GXNet) GetMediaType() string {
	return "Net"
}

func xmlEscape(s string) string {
	var buf bytes.Buffer
	if err := xml.EscapeText(&buf, []byte(s)); err != nil {
		return s
	}
	return buf.String()
}

// GetSettings serializes current connection settings into XML fragments.
func (g *GXNet) GetSettings() string {
	var b strings.Builder
	if g.HostName != "" {
		fmt.Fprintf(&b, "<IP>%s</IP>\n", xmlEscape(g.HostName))
	}
	if g.Port != 0 {
		fmt.Fprintf(&b, "<Port>%v</Port>\n", g.Port)
	}
	if g.Protocol != NetworkTypeTCP {
		fmt.Fprintf(&b, "<Protocol>%d</Protocol>\n", int(g.Protocol))
	}
	if g.UseIPv6 {
		b.WriteString("<IPv6>1</IPv6>\n")
	}
	return b.String()
}

// SetSettings loads connection settings from XML fragments.
//
// Supported elements are IP, Port, Protocol, and IPv6.
func (g *GXNet) SetSettings(value string) error {

	if strings.TrimSpace(value) == "" {
		return nil
	}
	dec := xml.NewDecoder(strings.NewReader("<root>" + value + "</root>"))
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		se, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}

		switch se.Name.Local {
		case "Protocol":
			var v string
			if err := dec.DecodeElement(&v, &se); err != nil {
				return err
			}
			if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
				g.Protocol = NetworkType(n)
			}
		case "Port":
			var v string
			if err := dec.DecodeElement(&v, &se); err != nil {
				return err
			}
			if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
				g.Port = n
			}
		case "Server":
			var v string
			if err := dec.DecodeElement(&v, &se); err != nil {
				return err
			}
		case "IP":
			var v string
			if err := dec.DecodeElement(&v, &se); err != nil {
				return err
			}
			g.HostName = v
		case "IPv6":
			var v string
			if err := dec.DecodeElement(&v, &se); err != nil {
				return err
			}
			g.UseIPv6 = strings.TrimSpace(v) == "1"
		}
	}
	return nil
}

// GetSynchronous enables synchronous receive mode.
//
// It returns a restore function that disables synchronous mode when called.
func (g *GXNet) GetSynchronous() func() {
	g.mu.Lock()
	g.synchronous = true
	g.mu.Unlock()
	return func() {
		g.mu.Lock()
		g.synchronous = false
		g.mu.Unlock()
	}
}

// IsSynchronous reports whether synchronous receive mode is enabled.
func (g *GXNet) IsSynchronous() bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.synchronous
}

// ResetSynchronousBuffer resets the synchronous buffer state.
//
// For GXNet this is currently a no-op.
func (g *GXNet) ResetSynchronousBuffer() {
}

// GetBytesSent returns the number of payload bytes sent.
func (g *GXNet) GetBytesSent() uint64 {
	return g.bytesSent
}

// GetBytesReceived returns the number of payload bytes received.
func (g *GXNet) GetBytesReceived() uint64 {
	return g.bytesReceived
}

// ResetByteCounters resets sent and received byte counters to zero.
func (g *GXNet) ResetByteCounters() {
	g.bytesSent = 0
	g.bytesReceived = 0
}

// Validate validates current settings before opening the connection.
//
// For GXNet this currently returns nil.
func (g *GXNet) Validate() error {
	return nil
}

// SetEop sets the end-of-packet marker used by Receive.
//
// The value can be a byte, string, or []byte.
func (g *GXNet) SetEop(eop any) {
	g.eop = eop
}

// GetEop returns the configured end-of-packet marker.
func (g *GXNet) GetEop() any {
	return g.eop
}

// GetTimeout returns the connection timeout in milliseconds.
func (g *GXNet) GetTimeout() uint32 {
	return uint32(g.timeout / time.Millisecond)
}

// SetTimeout sets the connection timeout in milliseconds.
func (g *GXNet) SetTimeout(value uint32) error {
	g.timeout = time.Duration(value) * time.Millisecond
	return nil
}

// GetTrace returns the current trace verbosity level.
func (g *GXNet) GetTrace() gxcommon.TraceLevel {
	return g.traceLevel
}

// SetTrace sets the trace verbosity level.
func (g *GXNet) SetTrace(traceLevel gxcommon.TraceLevel) error {
	g.traceLevel = traceLevel
	return nil
}

// SetOnReceived sets the callback for asynchronously received data.
func (g *GXNet) SetOnReceived(value gxcommon.ReceivedEventHandler) {
	g.mu.Lock()
	g.onReceive = value
	g.mu.Unlock()
}

// SetOnError sets the callback for asynchronous media errors.
func (g *GXNet) SetOnError(value gxcommon.ErrorEventHandler) {
	g.mu.Lock()
	g.onErr = value
	g.mu.Unlock()
}

// SetOnMediaStateChange sets the callback for media state transitions.
func (g *GXNet) SetOnMediaStateChange(value gxcommon.MediaStateHandler) {
	g.mu.Lock()
	g.onState = value
	g.mu.Unlock()
}

// SetOnTrace sets the callback for trace events.
func (g *GXNet) SetOnTrace(value gxcommon.TraceEventHandler) {
	g.mu.Lock()
	g.onTrace = value
	g.mu.Unlock()
}

// Open establishes the configured TCP or UDP connection.
//
// If the connection is already open, Open returns nil.
func (g *GXNet) Open() error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.conn != nil {
		return nil
	}
	g.statef(false, gxcommon.MediaStateOpening)
	g.trace(false, gxcommon.TraceTypesInfo, g.p.Sprintf("msg.connecting_to", g.Protocol.String(), g.HostName, strconv.Itoa(g.Port), g.timeout.Milliseconds()))
	var p string
	if g.Protocol == NetworkTypeTCP {
		if g.UseIPv6 {
			p = "tcp6"
		} else {
			p = "tcp4"
		}
	} else {
		if g.UseIPv6 {
			p = "udp6"
		} else {
			p = "udp"
		}
	}
	c, err := net.DialTimeout(p, g.HostName+":"+strconv.Itoa(g.Port), g.timeout)
	if err != nil {
		g.trace(false, gxcommon.TraceTypesError, g.p.Sprintf("msg.connect_failed", g.HostName, strconv.Itoa(g.Port), err))
		g.errorf(false, err)
		return err
	}
	g.conn = c
	g.wg.Add(1)
	go g.reader()

	g.trace(false, gxcommon.TraceTypesInfo, g.p.Sprintf("msg.connected_to", g.HostName, strconv.Itoa(g.Port)))
	g.statef(false, gxcommon.MediaStateOpen)
	return nil
}

// Send writes data to the open connection.
//
// The receiver parameter is ignored for network media.
func (g *GXNet) Send(data any, receiver string) error {
	tmp, err := gxcommon.ToBytes(data, binary.BigEndian)
	if err != nil {
		return err
	}
	g.mu.RLock()
	c := g.conn
	g.mu.RUnlock()
	if c == nil {
		return gxcommon.ErrConnectionClosed
	}
	g.bytesSent += uint64(len(tmp))
	//Trace data.
	str, err := gxcommon.ToString(data)
	if err != nil {
		return err
	}
	g.tracef(true, gxcommon.TraceTypesSent, "TX: %s", str)

	if g.timeout > 0 {
		_ = c.SetWriteDeadline(time.Now().Add(g.timeout))
	}
	_, ret := c.Write(tmp)
	return ret
}

// Receive waits until requested data is available in the synchronous buffer.
//
// Matching can be based on EOP, Count, or AllData in args.
func (g *GXNet) Receive(args *gxcommon.ReceiveParameters) (bool, error) {
	if args.EOP == nil && args.Count == 0 && !args.AllData {
		return false, errors.New(g.p.Sprintf("msg.count_or_eop"))
	}
	terminator, err := gxcommon.ToBytes(args.EOP, binary.BigEndian)
	if err != nil {
		return false, err
	}

	var waitTime time.Duration
	if args.WaitTime <= 0 {
		waitTime = 0
	} else {
		waitTime = time.Duration(args.WaitTime) * time.Millisecond
	}
	index := g.received.Search(terminator, args.Count, waitTime)
	if index == -1 {
		return false, nil
	}

	if args.AllData {
		//Read all data.
		index = -1
	}
	args.Reply, err = gxcommon.BytesToAny2(g.received.Get(index), args.ReplyType, binary.ByteOrder(binary.BigEndian))
	if err != nil {
		return false, err
	}
	return true, nil
}

func (g *GXNet) handleData(data []byte) {
	str, err := gxcommon.ToString(data)
	if err != nil {
		g.tracef(true, gxcommon.TraceTypesError, "RX failed: %v", err)
		g.errorf(true, err)
	} else {
		g.tracef(true, gxcommon.TraceTypesReceived, "RX: %s", str)
	}
	if g.synchronous {
		g.appendData(data)
	} else {
		g.receivef(true, data)
	}
}

func (g *GXNet) reader() {
	defer g.wg.Done()
	//Ethernet maximum frame size is 1518 bytes.
	buf := make([]byte, 1518)

	for {
		_ = g.conn.SetReadDeadline(time.Now().Add(time.Second))
		n, err := g.conn.Read(buf)
		if err != nil {
			// timeout
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				select {
				case <-g.stop:
					return
				default:
					continue
				}
			}
			if errors.Is(err, net.ErrClosed) {
				return
			}
			if (g.stop) != nil {
				g.trace(false, gxcommon.TraceTypesError, g.p.Sprintf("msg.connection_failed", err))
				g.errorf(false, err)
			}
			return
		}

		if n > 0 {
			g.bytesReceived += uint64(n)
			g.handleData(buf[:n])
		}
		select {
		case <-g.stop:
			return
		default:
		}
	}
}

func (g *GXNet) receivef(lock bool, data []byte) {
	var cb gxcommon.ReceivedEventHandler
	if lock {
		g.mu.RLock()
		cb = g.onReceive
		g.mu.RUnlock()
	} else {
		cb = g.onReceive
	}
	if cb != nil {
		cb(g, *gxcommon.NewReceiveEventArgs(data, g.HostName+":"+strconv.Itoa(g.Port)))
	}
}

func (g *GXNet) errorf(lock bool, err error) {
	var cb gxcommon.ErrorEventHandler
	if lock {
		g.mu.RLock()
		cb = g.onErr
		g.mu.RUnlock()
	} else {
		cb = g.onErr
	}
	if cb != nil {
		cb(g, err)
	}
}

func (g *GXNet) tracef(lock bool, traceType gxcommon.TraceTypes, fmtStr string, a ...any) {
	var cb gxcommon.TraceEventHandler
	trace := false
	if lock {
		g.mu.RLock()
		trace = !(int(g.traceLevel) < int(traceType))
		cb = g.onTrace
		g.mu.RUnlock()
	} else {
		trace = !(int(g.traceLevel) < int(traceType))
		cb = g.onTrace
	}
	if cb != nil && trace {
		p := gxcommon.NewTraceEventArgs(traceType, fmt.Sprintf(fmtStr, a...), "")
		var m gxcommon.IGXMedia = g
		cb(m, *p)
	}
}

func useTrace(level gxcommon.TraceLevel, traceType gxcommon.TraceTypes) bool {
	var ret bool
	switch level {
	case gxcommon.TraceLevelOff:
		ret = false
	case gxcommon.TraceLevelError:
		ret = traceType == gxcommon.TraceTypesError
	case gxcommon.TraceLevelVerbose:
		ret = true
	default:
		ret = traceType&gxcommon.TraceTypesError != 0 || traceType&gxcommon.TraceTypesWarning != 0
	}
	return ret
}

func (g *GXNet) trace(lock bool, traceType gxcommon.TraceTypes, message string) {
	var cb gxcommon.TraceEventHandler
	trace := false
	if lock {
		g.mu.RLock()
		trace = useTrace(g.traceLevel, traceType)
		cb = g.onTrace
		g.mu.RUnlock()
	} else {
		trace = !(int(g.traceLevel) < int(traceType))
		cb = g.onTrace
	}
	if cb != nil && trace {
		p := gxcommon.NewTraceEventArgs(traceType, message, "")
		var m gxcommon.IGXMedia = g
		cb(m, *p)
	}
}

func (g *GXNet) statef(lock bool, state gxcommon.MediaState) {
	var cb gxcommon.MediaStateHandler
	if lock {
		g.mu.RLock()
		cb = g.onState
		g.mu.RUnlock()
	} else {
		cb = g.onState
	}
	if cb != nil {
		cb(g, *gxcommon.NewMediaStateEventArgs(state))
	}
}

func (g *GXNet) appendData(data []byte) {
	if len(data) == 0 {
		return
	}
	g.received.Append(data)
	g.mu.Lock()
	g.receivedSize += len(data)
	g.mu.Unlock()
}

// Close stops the reader goroutine and closes the active connection.
//
// Close is safe to call multiple times.
func (g *GXNet) Close() error {
	g.mu.Lock()
	defer g.mu.Unlock()
	select {
	case <-g.stop:
		// already closed
	default:
		if g.conn != nil {
			g.trace(false, gxcommon.TraceTypesInfo, g.p.Sprintf("msg.closing_connection", g.HostName, strconv.Itoa(g.Port)))
			g.statef(false, gxcommon.MediaStateClosing)
		}
		close(g.stop)
	}
	var err error
	if g.conn != nil {
		// Make sure reader goroutine is not blocked on read.
		_ = g.conn.SetReadDeadline(time.Now())
		err = g.conn.Close()
		g.conn = nil
		g.trace(false, gxcommon.TraceTypesInfo, g.p.Sprintf("msg.connection_closed", g.HostName, strconv.Itoa(g.Port)))
		g.statef(false, gxcommon.MediaStateClosed)
	}
	g.wg.Wait()
	return err
}

//nolint:errcheck
func init() {
	// --- English (default) ---
	message.SetString(language.AmericanEnglish, "msg.closing_connection", "Closing connection to %s:%s")
	message.SetString(language.AmericanEnglish, "msg.connection_closed", "Connection closed to %s:%s")
	message.SetString(language.AmericanEnglish, "msg.connection_failed", "Connection failed: %v")
	message.SetString(language.AmericanEnglish, "msg.count_or_eop", "Either Count or EOP must be set")
	message.SetString(language.AmericanEnglish, "msg.connected_to", "Connected to %s:%s")
	message.SetString(language.AmericanEnglish, "msg.connect_failed", "connect to %s:%s failed: %v")
	message.SetString(language.AmericanEnglish, "msg.connecting_to", "%s connecting to %s:%s timeout %d ms")

	// --- German (de) ---
	message.SetString(language.German, "msg.closing_connection", "Verbindung zu %s:%s wird geschlossen")
	message.SetString(language.German, "msg.connection_closed", "Verbindung zu %s:%s wurde geschlossen")
	message.SetString(language.German, "msg.connection_failed", "Verbindung fehlgeschlagen: %v")
	message.SetString(language.German, "msg.count_or_eop", "Entweder Count oder EOP muss gesetzt sein")
	message.SetString(language.German, "msg.connected_to", "Verbunden mit %s:%s")
	message.SetString(language.German, "msg.connect_failed", "Verbindung zu %s:%s fehlgeschlagen: %v")
	message.SetString(language.German, "msg.connecting_to", "%s verbindet sich mit %s:%s timeout %d ms")

	// --- Finnish (fi) ---
	message.SetString(language.Finnish, "msg.closing_connection", "Suljetaan yhteys kohteeseen %s:%s")
	message.SetString(language.Finnish, "msg.connection_closed", "Yhteys suljettu kohteeseen %s:%s")
	message.SetString(language.Finnish, "msg.connection_failed", "Yhteyden muodostus epäonnistui: %v")
	message.SetString(language.Finnish, "msg.count_or_eop", "Joko Count tai EOP on asetettava")
	message.SetString(language.Finnish, "msg.connected_to", "Yhdistetty kohteeseen %s:%s")
	message.SetString(language.Finnish, "msg.connect_failed", "Yhteyden muodostus kohteeseen %s:%s epäonnistui: %v")
	message.SetString(language.Finnish, "msg.connecting_to", "%s yhdistetään kohteeseen %s:%s timeout %d ms")

	// --- Swedish (sv) ---
	message.SetString(language.Swedish, "msg.closing_connection", "Stänger anslutning till %s:%s")
	message.SetString(language.Swedish, "msg.connection_closed", "Anslutning stängd till %s:%s")
	message.SetString(language.Swedish, "msg.connection_failed", "Anslutningen misslyckades: %v")
	message.SetString(language.Swedish, "msg.count_or_eop", "Antingen Count eller EOP måste anges")
	message.SetString(language.Swedish, "msg.connected_to", "Ansluten till %s:%s")
	message.SetString(language.Swedish, "msg.connect_failed", "Anslutning till %s:%s misslyckades: %v")
	message.SetString(language.Swedish, "msg.connecting_to", "%s ansluter till %s:%s timeout %d ms")

	// --- Spanish (es) ---
	message.SetString(language.Spanish, "msg.closing_connection", "Cerrando conexión con %s:%s")
	message.SetString(language.Spanish, "msg.connection_closed", "Conexión cerrada con %s:%s")
	message.SetString(language.Spanish, "msg.connection_failed", "Error de conexión: %v")
	message.SetString(language.Spanish, "msg.count_or_eop", "Se debe establecer Count o EOP")
	message.SetString(language.Spanish, "msg.connected_to", "Conectado a %s:%s")
	message.SetString(language.Spanish, "msg.connect_failed", "Error al conectar con %s:%s: %v")
	message.SetString(language.Spanish, "msg.connecting_to", "%s conectando a %s:%s timeout %d ms")

	// --- Estonian (et) ---
	message.SetString(language.Estonian, "msg.closing_connection", "Suletakse ühendus sihtkohta %s:%s")
	message.SetString(language.Estonian, "msg.connection_closed", "Ühendus suleti sihtkohta %s:%s")
	message.SetString(language.Estonian, "msg.connection_failed", "Ühendus ebaõnnestus: %v")
	message.SetString(language.Estonian, "msg.count_or_eop", "Count või EOP peab olema määratud")
	message.SetString(language.Estonian, "msg.connected_to", "Ühendatud sihtkohta %s:%s")
	message.SetString(language.Estonian, "msg.connect_failed", "Ühendamine sihtkohta %s:%s ebaõnnestus: %v")
	message.SetString(language.Estonian, "msg.connecting_to", "%s ühendatakse sihtkohta %s:%s timeout %d ms")
}
