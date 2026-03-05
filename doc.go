// Package gxnet provides TCP/UDP-based media for Gurux components.
// It implements the common IGXMedia-style contract: open/close a connection,
// send/receive data (optionally framed by an EOP marker), and emit events for
// received data, errors, tracing and state changes.
//
// The transport is fully concurrent-safe, supports optional end-of-packet
// (EOP) framing, adjustable timeouts, and event callbacks that allow the
// caller to handle data and state changes asynchronously. Synchronous
// receive mode is also provided for callers that prefer a blocking API.
//
// Features
//
//   - Protocols: TCP and UDP (see NetworkType).
//   - Framing: optional EOP (End Of Packet) marker (byte, string or []byte).
//   - Timeouts: connection and I/O timeouts via time.Duration.
//   - Tracing: configurable trace level/mask for sent/received/error/info.
//   - Events: Received, Error, Trace and MediaState callbacks.
//   - Concurrency: safe for concurrent reads/writes; Close unblocks pending I/O.
//
// # Construction
//
// Use NewGXNet to create a connection with protocol, host and port. Additional
// options (such as timeouts, EOP, tracing, IPv6, etc.) can be configured
// through setter methods before calling Open.
//
// The following example demonstrates basic usage:
//
//	enm := gxnet.NewGXNet(gxnet.NetworkTypeTCP, "127.0.0.1", 4059)
//	enm.SetTimeout(5000) // milliseconds
//
//	enm.SetOnReceived(func(m IGXMedia, e ReceiveEventArgs) {
//		// handle incoming data
//	})
//	enm.SetOnError(func(m IGXMedia, err error) {
//		// handle errors
//	})
//
//	if err := enm.Open(); err != nil {
//		// handle connect error
//	}
//	defer enm.Close()
//
//	// send bytes; receive happens via the callback or via a blocking Receive.
//	_, _ = enm.Send([]byte{0x01, 0x02, 0x03})
//
// # EOP framing
//
// When an EOP is configured, incoming bytes are buffered until the marker is
// observed. The marker can be a single byte (e.g. 0x7E), a string (e.g. "OK"),
// or an arbitrary byte slice. Disable EOP to read raw stream data.
//
// # Errors and timeouts
//
// Network and protocol errors are returned from calls or routed to Error
// handlers. Timeouts follow Go conventions (context/deadline or Duration-based
// configuration). Error messages are lowercased per Go style guidelines.
//
// # Notes
//
// The zero value of GXNet is not ready for use; always construct via NewGXNet.
// Long-running work in event handlers should be offloaded to a separate
// goroutine to avoid blocking I/O paths.
package gxnet

import (
	"github.com/Gurux/gxcommon-go"
)

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

// ExampleNewGXNet demonstrates a minimal client that opens a connection,
// sends a few bytes, and then closes the media. The example is suitable for
// inclusion in godoc output and can be run with `go test`.
func ExampleNewGXNet() {
	m := NewGXNet(NetworkTypeTCP, "127.0.0.1", 4059)
	m.SetTimeout(5000)
	m.SetOnError(func(gxcommon.IGXMedia, error) {})
	m.SetOnReceived(func(gxcommon.IGXMedia, gxcommon.ReceiveEventArgs) {})
	if err := m.Open(); err != nil {
		return
	}
	defer m.Close()
	if err := m.Send([]byte("hello"), ""); err != nil {
		return
	}
	// Output:
}
