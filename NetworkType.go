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
	"fmt"
	"strings"

	"github.com/Gurux/gxcommon-go"
)

// NetworkType identifies which transport protocol GXNet uses.
type NetworkType int

const (
	// NetworkTypeUDP selects UDP datagram transport.
	NetworkTypeUDP NetworkType = iota
	// NetworkTypeTCP selects TCP stream transport.
	NetworkTypeTCP
)

// NetworkTypeParse parses a protocol name into a NetworkType value.
//
// Accepted values are "TCP" and "UDP" (case-insensitive). If the string
// does not match one of those names an error wrapping gxcommon.ErrUnknownEnum
// is returned.
func NetworkTypeParse(value string) (NetworkType, error) {
	var ret NetworkType
	var err error
	switch strings.ToUpper(value) {
	case "UDP":
		ret = NetworkTypeUDP
	case "TCP":
		ret = NetworkTypeTCP
	default:
		err = fmt.Errorf("%w: %q", gxcommon.ErrUnknownEnum, value)
	}
	return ret, err
}

// ExampleNetworkTypeParse demonstrates parsing valid and invalid values.
func ExampleNetworkTypeParse() {
	if _, err := NetworkTypeParse("tcp"); err != nil {
		return
	}
	if _, err := NetworkTypeParse("bogus"); err == nil {
		return
	}
	// Output:
}

// String returns the canonical name of the network type.
// It satisfies fmt.Stringer.
func (g NetworkType) String() string {
	var ret string
	switch g {
	case NetworkTypeUDP:
		ret = "UDP"
	case NetworkTypeTCP:
		ret = "TCP"
	}
	return ret
}
