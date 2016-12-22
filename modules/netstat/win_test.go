// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Rob Murtha robmurtha@gmail.com [:robmurtha]
package netstat

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNetstatWinOutput(t *testing.T) {
	res := &NetstatWinOutput{}
	require.NoError(t, res.UnmarshalText(testWinNetstatOutput))
}

func TestNetstatWinEntry_parseEndpoint(t *testing.T) {
	entry := &netstatWinEntry{}
	ip, port, err := entry.parseEndpointString(`131.253.34.248:443`)
	require.NoError(t, err)
	require.Equal(t, `131.253.34.248`, ip.String())
	require.Equal(t, 443, port)
}

func TestNetstatWinOutputTCP4(t *testing.T) {
	// TCP data has 4 fields
	data := []byte(`
  TCP    0.0.0.0:49673          0.0.0.0:0              LISTENING
  TCP    10.0.2.15:139          0.0.0.0:0              LISTENING
  TCP    10.0.2.15:49671        131.253.34.248:443     ESTABLISHED
`)
	res := &NetstatWinOutput{}
	require.NoError(t, res.UnmarshalText(data))
	require.Len(t, res.Entries, 3)
	for _, e := range res.Entries {
		require.NotNil(t, e.LocalIP)
		require.NotEqual(t, 0, e.LocalPort)
		require.NotNil(t, e.RemoteIP)
	}
	require.Equal(t, "0.0.0.0", res.Entries[0].LocalIP.String())
	require.Equal(t, 49673, res.Entries[0].LocalPort)
	require.Equal(t, "0.0.0.0", res.Entries[1].RemoteIP.String())
	require.Equal(t, 139, res.Entries[1].LocalPort)

	require.Equal(t, "10.0.2.15", res.Entries[2].LocalIP.String())
	require.Equal(t, 49671, res.Entries[2].LocalPort)
	require.Equal(t, "131.253.34.248", res.Entries[2].RemoteIP.String())
	require.Equal(t, 443, res.Entries[2].RemotePort)
}

func TestNetstatWinOutputTCP6(t *testing.T) {
	// TCP data has 4 fields
	data := []byte(`
  TCP    [::]:135               [::]:0                 LISTENING
`)
	res := &NetstatWinOutput{}
	require.NoError(t, res.UnmarshalText(data))
	require.Len(t, res.Entries, 1)
	entry := res.Entries[0]
	require.Nil(t, entry.LocalIP.To4())
	require.Equal(t, "<nil>", entry.LocalIP.String())
	require.Equal(t, 135, entry.LocalPort)
	require.Equal(t, "<nil>", entry.RemoteIP.String())
	require.Equal(t, 0, entry.RemotePort)
}

func TestNetstatWinowsParserWildcard(t *testing.T) {
	data := []byte(`
	UDP    0.0.0.0:5355           *:*
	`)

	res := &NetstatWinOutput{}
	require.NoError(t, res.UnmarshalText(data))
	require.Len(t, res.Entries, 1)
	entry := res.Entries[0]
	require.Nil(t, entry.RemoteIP)
}

func TestNetstatWinOutputUDP4(t *testing.T) {
	// UDP data has 3 fields
	data := []byte(`
  UDP    0.0.0.0:5355           *:*
  UDP    10.0.2.15:137          *:*
`)
	res := &NetstatWinOutput{}
	require.NoError(t, res.UnmarshalText(data))
	require.Len(t, res.Entries, 2)
	for _, e := range res.Entries {
		require.NotNil(t, e.LocalIP)
		require.NotEqual(t, 0, e.LocalPort)
		require.Nil(t, e.RemoteIP)
	}
}

func TestNetstatWinOutputUDP6(t *testing.T) {
	// UDP data has 3 fields
	data := []byte(`
  UDP    [::]:5353              *:*
  UDP    [::1]:1900             *:*
  UDP    [fe80::1c5a:2c75:6ce3:9f0%3]:1900  *:*
`)
	res := &NetstatWinOutput{}
	require.NoError(t, res.UnmarshalText(data))
	require.Len(t, res.Entries, 3)
	for _, e := range res.Entries {
		require.NotEqual(t, 0, e.LocalPort)
	}
	entry := res.Entries[2]
	require.Nil(t, entry.LocalIP.To4())
}

func TestNetstatWinOutputHasIPConnected(t *testing.T) {
	res := &NetstatWinOutput{}
	require.NoError(t, res.UnmarshalText(testWinNetstatOutput))

	ipnet, _ := res.getIPNet("131.253.34.248")
	elements := res.HasIPConnected(ipnet)
	require.Len(t, elements, 1)
	el := elements[0]
	require.Equal(t, "10.0.2.15", el.LocalAddr)
	require.Equal(t, float64(49671), el.LocalPort)
	require.Equal(t, "131.253.34.248", el.RemoteAddr)
	require.Equal(t, float64(443), el.RemotePort)

	ipnet, _ = res.getIPNet("131.253.34.0/8")
	require.Len(t, res.HasIPConnected(ipnet), 1)

	ipnet, _ = res.getIPNet("131.253.34.99")
	require.Len(t, res.HasIPConnected(ipnet), 0)
}

func TestNetstatWinOutputHasListeningPort(t *testing.T) {
	res := &NetstatWinOutput{}
	require.NoError(t, res.UnmarshalText(testWinNetstatOutput))
	require.Len(t, res.HasListeningPort(99), 0)
	require.Len(t, res.HasListeningPort(135), 2)
	require.Len(t, res.HasListeningPort(49671), 1)
	require.Len(t, res.HasListeningPort(1900), 4)
	require.Len(t, res.HasListeningPort(54109), 1)
}

var testWinNetstatOutput = []byte(`

Active Connections

  Proto  Local Address          Foreign Address        State
  TCP    0.0.0.0:135            0.0.0.0:0              LISTENING
  TCP    0.0.0.0:445            0.0.0.0:0              LISTENING
  TCP    0.0.0.0:7680           0.0.0.0:0              LISTENING
  TCP    0.0.0.0:49664          0.0.0.0:0              LISTENING
  TCP    0.0.0.0:49665          0.0.0.0:0              LISTENING
  TCP    0.0.0.0:49666          0.0.0.0:0              LISTENING
  TCP    0.0.0.0:49667          0.0.0.0:0              LISTENING
  TCP    0.0.0.0:49668          0.0.0.0:0              LISTENING
  TCP    0.0.0.0:49673          0.0.0.0:0              LISTENING
  TCP    10.0.2.15:139          0.0.0.0:0              LISTENING
  TCP    10.0.2.15:49671        131.253.34.248:443     ESTABLISHED
  TCP    10.0.2.15:49672        64.4.54.253:443        TIME_WAIT
  TCP    10.0.2.15:49676        93.184.216.146:443     TIME_WAIT
  TCP    10.0.2.15:49677        23.222.161.245:80      TIME_WAIT
  TCP    10.0.2.15:49679        65.52.108.205:443      ESTABLISHED
  TCP    10.0.2.15:49680        185.48.81.162:80       TIME_WAIT
  TCP    10.0.2.15:49685        40.112.223.14:443      ESTABLISHED
  TCP    10.0.2.15:49686        13.90.208.215:443      ESTABLISHED
  TCP    10.0.2.15:49691        23.212.189.103:443     ESTABLISHED
  TCP    10.0.2.15:49693        23.222.173.76:80       ESTABLISHED
  TCP    10.0.2.15:49695        23.222.161.245:80      ESTABLISHED
  TCP    10.0.2.15:49696        23.213.5.176:443       ESTABLISHED
  TCP    10.0.2.15:49697        23.212.250.233:443     ESTABLISHED
  TCP    10.0.2.15:49698        204.79.197.200:443     TIME_WAIT
  TCP    10.0.2.15:49699        204.79.197.200:443     ESTABLISHED
  TCP    [::]:135               [::]:0                 LISTENING
  TCP    [::]:445               [::]:0                 LISTENING
  TCP    [::]:7680              [::]:0                 LISTENING
  TCP    [::]:49664             [::]:0                 LISTENING
  TCP    [::]:49665             [::]:0                 LISTENING
  TCP    [::]:49666             [::]:0                 LISTENING
  TCP    [::]:49667             [::]:0                 LISTENING
  TCP    [::]:49668             [::]:0                 LISTENING
  TCP    [::]:49673             [::]:0                 LISTENING
  UDP    0.0.0.0:3544           *:*
  UDP    0.0.0.0:5050           *:*
  UDP    0.0.0.0:5353           *:*
  UDP    0.0.0.0:5355           *:*
  UDP    10.0.2.15:137          *:*
  UDP    10.0.2.15:138          *:*
  UDP    10.0.2.15:1900         *:*
  UDP    10.0.2.15:52273        *:*
  UDP    10.0.2.15:54111        *:*
  UDP    127.0.0.1:1900         *:*
  UDP    127.0.0.1:54112        *:*
  UDP    [::]:5353              *:*
  UDP    [::]:5355              *:*
  UDP    [::1]:1900             *:*
  UDP    [::1]:54110            *:*
  UDP    [fe80::1c5a:2c75:6ce3:9f0%3]:1900  *:*
  UDP    [fe80::1c5a:2c75:6ce3:9f0%3]:54109  *:*

`)
