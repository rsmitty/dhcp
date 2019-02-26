package dhcpv4

import (
	"fmt"
	"net"

	"github.com/u-root/u-root/pkg/uio"
)

type ClasslessRoutes []ClasslessRoute

type ClasslessRoute struct {
	Destination *net.IPNet
	Router      net.IP
}

// FromBytes implements the option decoder interface
func (c *ClasslessRoutes) FromBytes(data []byte) error {
	buf := uio.NewBigEndianBuffer(data)
	// minimum length is 5 bytes
	if buf.Len() < 5 {
		return fmt.Errorf("Classless Route DHCP option must always list at least one route")
	}

	var cr ClasslessRoute
	var mask net.IPMask
	var sigbits int
	var maskCIDR int
	var router net.IP
	var dst net.IP

	for {
		if !buf.Has(1) {
			break
		}

		maskCIDR = int(buf.CopyN(1)[0])
		mask = net.CIDRMask(maskCIDR, 32)

		switch {
		case maskCIDR == 0:
			sigbits = 0
		case maskCIDR > 0 && maskCIDR <= 8:
			sigbits = 1
		case maskCIDR > 8 && maskCIDR <= 16:
			sigbits = 2
		case maskCIDR > 16 && maskCIDR <= 24:
			sigbits = 3
		case maskCIDR > 24 && maskCIDR <= 32:
			sigbits = 4
		}

		if !buf.Has(sigbits) {
			break
		}

		d := buf.CopyN(sigbits)

		for i := len(d); i < net.IPv4len; i++ {
			d = append(d, 0)
		}

		if !buf.Has(net.IPv4len) {
			break
		}
		r := buf.CopyN(net.IPv4len)

		cr = ClasslessRoute{}
		router = net.IP(r)
		if router.String() == net.IPv4zero.String() {
			router = nil
		}
		cr.Router = router

		dst = net.IP(d)
		if dst.String() != net.IPv4zero.String() {
			cr.Destination = &net.IPNet{
				IP:   dst,
				Mask: mask,
			}
		}

		*c = append(*c, cr)
	}
	return buf.FinError()
}

// String returns a human-readable IP.
func (c ClasslessRoutes) String() string {
	str := "["
	for i, r := range c {
		str += r.Destination.String()
		str += " "
		str += r.Router.String()
		if i < len(c)-1 {
			str += ", "
		}
	}
	str += "]"
	return str
}

// GetClasslessRoutes returrns a slice of ClasslessRoute structs
func GetClasslessRoutes(code OptionCode, o Options) ClasslessRoutes {
	v := o.Get(code)
	if v == nil {
		return nil
	}

	var crs ClasslessRoutes
	if err := crs.FromBytes(v); err != nil {
		return nil
	}
	return crs
}
