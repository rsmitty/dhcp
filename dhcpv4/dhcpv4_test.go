package dhcpv4

import (
	"net"
	"testing"

	"github.com/rsmitty/dhcp/iana"
	"github.com/stretchr/testify/require"
)

func TestGetExternalIPv4Addrs(t *testing.T) {
	addrs4and6 := []net.Addr{
		&net.IPAddr{IP: net.IP{1, 2, 3, 4}},
		&net.IPAddr{IP: net.IP{4, 3, 2, 1}},
		&net.IPNet{IP: net.IP{4, 3, 2, 0}},
		&net.IPAddr{IP: net.IP{1, 2, 3, 4, 1, 1, 1, 1}},
		&net.IPAddr{IP: net.IP{4, 3, 2, 1, 1, 1, 1, 1}},
		&net.IPAddr{},                         // nil IP
		&net.IPAddr{IP: net.IP{127, 0, 0, 1}}, // loopback IP
	}

	expected := []net.IP{
		net.IP{1, 2, 3, 4},
		net.IP{4, 3, 2, 1},
		net.IP{4, 3, 2, 0},
	}
	actual, err := GetExternalIPv4Addrs(addrs4and6)
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}

func TestFromBytes(t *testing.T) {
	data := []byte{
		1,                      // dhcp request
		1,                      // ethernet hw type
		6,                      // hw addr length
		3,                      // hop count
		0xaa, 0xbb, 0xcc, 0xdd, // transaction ID, big endian (network)
		0, 3, // number of seconds
		0, 1, // broadcast
		0, 0, 0, 0, // client IP address
		0, 0, 0, 0, // your IP address
		0, 0, 0, 0, // server IP address
		0, 0, 0, 0, // gateway IP address
		0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, // client MAC address + padding
	}

	// server host name
	expectedHostname := []byte{}
	for i := 0; i < 64; i++ {
		expectedHostname = append(expectedHostname, 0)
	}
	data = append(data, expectedHostname...)
	// boot file name
	expectedBootfilename := []byte{}
	for i := 0; i < 128; i++ {
		expectedBootfilename = append(expectedBootfilename, 0)
	}
	data = append(data, expectedBootfilename...)
	// magic cookie, then no options
	data = append(data, magicCookie[:]...)

	d, err := FromBytes(data)
	require.NoError(t, err)
	require.Equal(t, d.OpCode, OpcodeBootRequest)
	require.Equal(t, d.HWType, iana.HWTypeEthernet)
	require.Equal(t, d.HopCount, byte(3))
	require.Equal(t, d.TransactionID, TransactionID{0xaa, 0xbb, 0xcc, 0xdd})
	require.Equal(t, d.NumSeconds, uint16(3))
	require.Equal(t, d.Flags, uint16(1))
	require.True(t, d.ClientIPAddr.Equal(net.IPv4zero))
	require.True(t, d.YourIPAddr.Equal(net.IPv4zero))
	require.True(t, d.GatewayIPAddr.Equal(net.IPv4zero))
	require.Equal(t, d.ClientHWAddr, net.HardwareAddr{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff})
	require.Equal(t, d.ServerHostName, "")
	require.Equal(t, d.BootFileName, "")
	// no need to check Magic Cookie as it is already validated in FromBytes
	// above
}

func TestFromBytesZeroLength(t *testing.T) {
	data := []byte{}
	_, err := FromBytes(data)
	require.Error(t, err)
}

func TestFromBytesShortLength(t *testing.T) {
	data := []byte{1, 1, 6, 0}
	_, err := FromBytes(data)
	require.Error(t, err)
}

func TestFromBytesInvalidOptions(t *testing.T) {
	data := []byte{
		1,                      // dhcp request
		1,                      // ethernet hw type
		6,                      // hw addr length
		0,                      // hop count
		0xaa, 0xbb, 0xcc, 0xdd, // transaction ID
		3, 0, // number of seconds
		1, 0, // broadcast
		0, 0, 0, 0, // client IP address
		0, 0, 0, 0, // your IP address
		0, 0, 0, 0, // server IP address
		0, 0, 0, 0, // gateway IP address
		0xff, 0xee, 0xdd, 0xcc, 0xbb, 0xaa, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, // client MAC address + padding
	}
	// server host name
	for i := 0; i < 64; i++ {
		data = append(data, 0)
	}
	// boot file name
	for i := 0; i < 128; i++ {
		data = append(data, 0)
	}
	// invalid magic cookie, forcing option parsing to fail
	data = append(data, []byte{99, 130, 83, 98}...)
	_, err := FromBytes(data)
	require.Error(t, err)
}

func TestToStringMethods(t *testing.T) {
	d, err := New()
	if err != nil {
		t.Fatal(err)
	}

	// FlagsToString
	d.SetUnicast()
	require.Equal(t, "Unicast", d.FlagsToString())
	d.SetBroadcast()
	require.Equal(t, "Broadcast", d.FlagsToString())
	d.Flags = 0xffff
	require.Equal(t, "Broadcast (reserved bits not zeroed)", d.FlagsToString())
}

func TestNewToBytes(t *testing.T) {
	// the following bytes match what dhcpv4.New would create. Keep them in
	// sync!
	expected := []byte{
		1,                      // Opcode BootRequest
		1,                      // HwType Ethernet
		6,                      // HwAddrLen
		0,                      // HopCount
		0x11, 0x22, 0x33, 0x44, // TransactionID
		0, 0, // NumSeconds
		0, 0, // Flags
		0, 0, 0, 0, // ClientIPAddr
		0, 0, 0, 0, // YourIPAddr
		0, 0, 0, 0, // ServerIPAddr
		0, 0, 0, 0, // GatewayIPAddr
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, // ClientHwAddr
	}
	// ServerHostName
	for i := 0; i < 64; i++ {
		expected = append(expected, 0)
	}
	// BootFileName
	for i := 0; i < 128; i++ {
		expected = append(expected, 0)
	}
	// Magic Cookie
	expected = append(expected, magicCookie[:]...)
	// End
	expected = append(expected, 0xff)

	d, err := New()
	require.NoError(t, err)
	// fix TransactionID to match the expected one, since it's randomly
	// generated in New()
	d.TransactionID = TransactionID{0x11, 0x22, 0x33, 0x44}
	got := d.ToBytes()
	require.Equal(t, expected, got)
}

func TestGetOption(t *testing.T) {
	d, err := New()
	if err != nil {
		t.Fatal(err)
	}

	hostnameOpt := OptGeneric(OptionHostName, []byte("darkstar"))
	bootFileOpt := OptBootFileName("boot.img")
	d.UpdateOption(hostnameOpt)
	d.UpdateOption(bootFileOpt)

	require.Equal(t, d.GetOneOption(OptionHostName), []byte("darkstar"))
	require.Equal(t, d.GetOneOption(OptionBootfileName), []byte("boot.img"))
	require.Equal(t, d.GetOneOption(OptionRouter), []byte(nil))
}

func TestUpdateOption(t *testing.T) {
	d, err := New()
	require.NoError(t, err)

	hostnameOpt := OptGeneric(OptionHostName, []byte("darkstar"))
	bootFileOpt1 := OptBootFileName("boot.img")
	bootFileOpt2 := OptBootFileName("boot2.img")
	d.UpdateOption(hostnameOpt)
	d.UpdateOption(bootFileOpt1)
	d.UpdateOption(bootFileOpt2)

	options := d.Options
	require.Equal(t, len(options), 2)
	require.Equal(t, d.GetOneOption(OptionHostName), []byte("darkstar"))
	require.Equal(t, d.GetOneOption(OptionBootfileName), []byte("boot2.img"))
}

func TestDHCPv4NewRequestFromOffer(t *testing.T) {
	offer, err := New()
	require.NoError(t, err)
	offer.SetBroadcast()
	offer.UpdateOption(OptMessageType(MessageTypeOffer))
	req, err := NewRequestFromOffer(offer)
	require.Error(t, err)

	// Now add the option so it doesn't error out.
	offer.UpdateOption(OptServerIdentifier(net.IPv4(192, 168, 0, 1)))

	// Broadcast request
	req, err = NewRequestFromOffer(offer)
	require.NoError(t, err)
	require.Equal(t, MessageTypeRequest, req.MessageType())
	require.False(t, req.IsUnicast())
	require.True(t, req.IsBroadcast())

	// Unicast request
	offer.SetUnicast()
	req, err = NewRequestFromOffer(offer)
	require.NoError(t, err)
	require.True(t, req.IsUnicast())
	require.False(t, req.IsBroadcast())
}

func TestDHCPv4NewRequestFromOfferWithModifier(t *testing.T) {
	offer, err := New()
	require.NoError(t, err)
	offer.UpdateOption(OptMessageType(MessageTypeOffer))
	offer.UpdateOption(OptServerIdentifier(net.IPv4(192, 168, 0, 1)))
	userClass := WithUserClass([]byte("linuxboot"), false)
	req, err := NewRequestFromOffer(offer, userClass)
	require.NoError(t, err)
	require.Equal(t, MessageTypeRequest, req.MessageType())
}

func TestNewReplyFromRequest(t *testing.T) {
	discover, err := New()
	require.NoError(t, err)
	discover.GatewayIPAddr = net.IPv4(192, 168, 0, 1)
	reply, err := NewReplyFromRequest(discover)
	require.NoError(t, err)
	require.Equal(t, discover.TransactionID, reply.TransactionID)
	require.Equal(t, discover.GatewayIPAddr, reply.GatewayIPAddr)
}

func TestNewReplyFromRequestWithModifier(t *testing.T) {
	discover, err := New()
	require.NoError(t, err)
	discover.GatewayIPAddr = net.IPv4(192, 168, 0, 1)
	userClass := WithUserClass([]byte("linuxboot"), false)
	reply, err := NewReplyFromRequest(discover, userClass)
	require.NoError(t, err)
	require.Equal(t, discover.TransactionID, reply.TransactionID)
	require.Equal(t, discover.GatewayIPAddr, reply.GatewayIPAddr)
}

func TestDHCPv4MessageTypeNil(t *testing.T) {
	m, err := New()
	require.NoError(t, err)
	require.Equal(t, MessageTypeNone, m.MessageType())
}

func TestNewDiscovery(t *testing.T) {
	hwAddr := net.HardwareAddr{1, 2, 3, 4, 5, 6}
	m, err := NewDiscovery(hwAddr)
	require.NoError(t, err)
	require.Equal(t, MessageTypeDiscover, m.MessageType())

	// Validate fields of DISCOVER packet.
	require.Equal(t, OpcodeBootRequest, m.OpCode)
	require.Equal(t, iana.HWTypeEthernet, m.HWType)
	require.Equal(t, hwAddr, m.ClientHWAddr)
	require.True(t, m.IsBroadcast())
	require.True(t, m.Options.Has(OptionParameterRequestList))
}

func TestNewInform(t *testing.T) {
	hwAddr := net.HardwareAddr{1, 2, 3, 4, 5, 6}
	localIP := net.IPv4(10, 10, 11, 11)
	m, err := NewInform(hwAddr, localIP)

	require.NoError(t, err)
	require.Equal(t, OpcodeBootRequest, m.OpCode)
	require.Equal(t, iana.HWTypeEthernet, m.HWType)
	require.Equal(t, hwAddr, m.ClientHWAddr)
	require.Equal(t, MessageTypeInform, m.MessageType())
	require.True(t, m.ClientIPAddr.Equal(localIP))
}

func TestIsOptionRequested(t *testing.T) {
	pkt, err := New()
	require.NoError(t, err)
	require.False(t, pkt.IsOptionRequested(OptionDomainNameServer))

	optprl := OptParameterRequestList(OptionDomainNameServer)
	pkt.UpdateOption(optprl)
	require.True(t, pkt.IsOptionRequested(OptionDomainNameServer))
}

// TODO
//      test Summary() and String()
func TestSummary(t *testing.T) {
	packet, err := New(WithMessageType(MessageTypeInform))
	packet.TransactionID = [4]byte{1, 1, 1, 1}
	require.NoError(t, err)

	want := "DHCPv4 Message\n" +
		"  opcode: BootRequest\n" +
		"  hwtype: Ethernet\n" +
		"  hopcount: 0\n" +
		"  transaction ID: 0x01010101\n" +
		"  num seconds: 0\n" +
		"  flags: Unicast (0x00)\n" +
		"  client IP: 0.0.0.0\n" +
		"  your IP: 0.0.0.0\n" +
		"  server IP: 0.0.0.0\n" +
		"  gateway IP: 0.0.0.0\n" +
		"  client MAC: \n" +
		"  server hostname: \n" +
		"  bootfile name: \n" +
		"  options:\n" +
		"    DHCP Message Type: INFORM\n"
	require.Equal(t, want, packet.Summary())
}
