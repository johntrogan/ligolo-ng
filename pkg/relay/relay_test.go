// Ligolo-ng
// Copyright (C) 2025 Nicolas Chatelain (nicocha30)

// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package relay

import (
	"bytes"
	"net"
	"testing"
	"time"
)

// TestStartUDPListenerRelayReturnTraffic reproduces
// https://github.com/nicocha30/ligolo-ng/issues/147: a reverse UDP listener
// must keep relaying after the redirect target replies. The agent listens on
// an unconnected UDP socket (net.ListenUDP) and relays datagrams over the
// tunnel. The previous implementation pumped raw bytes with io.Copy, so the
// return path called Write on the unconnected socket, which always fails and
// tore the whole listener down on the first reply.
func TestStartUDPListenerRelayReturnTraffic(t *testing.T) {
	// Agent-side UDP listener, exactly as NewUDPListener creates it.
	udpConn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatalf("ListenUDP: %v", err)
	}
	defer udpConn.Close()
	listenerAddr := udpConn.LocalAddr().(*net.UDPAddr)

	// net.Pipe stands in for the yamux tunnel back to the proxy.
	proxySide, tunnelSide := net.Pipe()
	defer proxySide.Close()
	defer tunnelSide.Close()

	go StartUDPListenerRelay(tunnelSide, udpConn)

	// A UDP client reaches the agent listener and sends a datagram.
	client, err := net.DialUDP("udp", nil, listenerAddr)
	if err != nil {
		t.Fatalf("DialUDP: %v", err)
	}
	defer client.Close()

	if _, err := client.Write([]byte("ping")); err != nil {
		t.Fatalf("client write: %v", err)
	}

	// Forward direction: the datagram must reach the proxy side of the tunnel.
	if err := proxySide.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("set deadline: %v", err)
	}
	buf := make([]byte, 1024)
	n, err := proxySide.Read(buf)
	if err != nil {
		t.Fatalf("reading forwarded datagram from tunnel: %v", err)
	}
	if got := buf[:n]; !bytes.Equal(got, []byte("ping")) {
		t.Fatalf("forwarded datagram = %q, want %q", got, "ping")
	}

	// Return direction: a reply written back through the tunnel must reach the
	// original UDP client. This is the behavior that issue #147 reports broken.
	if _, err := proxySide.Write([]byte("pong")); err != nil {
		t.Fatalf("proxy write: %v", err)
	}

	if err := client.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("set deadline: %v", err)
	}
	n, err = client.Read(buf)
	if err != nil {
		t.Fatalf("client did not receive return traffic (issue #147): %v", err)
	}
	if got := buf[:n]; !bytes.Equal(got, []byte("pong")) {
		t.Fatalf("client received %q, want %q", got, "pong")
	}
}
