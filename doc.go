// Package gxnet provides TCP/UDP based media for Gurux components.
// It implements the common IGXMedia-style contract: open/close a connection,
// send/receive data (optionally framed by an EOP marker), and emit events for
// received data, errors, tracing and state changes.
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
// options (such as timeout, EOP, tracing) can be configured through setters.
//
// Example
//
//	n := gxnet.NewGXNet(gxnet.TCP, "127.0.0.1", 4059)
//	n.SetTimeout(5_000) // 5000 ms; or use time.Duration setters if exposed
//
//	n.SetOnReceived(func(m IGXMedia, e ReceiveEventArgs) {
//	    // handle e.Data(), e.SenderInfo()
//	})
//	n.SetOnError(func(m IGXMedia, err error) {
//	    // log/handle error
//	})
//
//	if err := n.Open(); err != nil {
//	    // handle connect error
//	}
//	defer n.Close()
//
//	// send bytes; receive happens via the callback or via a blocking Receive.
//	_, _ = n.Send([]byte{0x01, 0x02, 0x03})
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
