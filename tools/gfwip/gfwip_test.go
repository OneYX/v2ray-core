package gfwip

import (
	"github.com/golang/protobuf/proto"
	"math"
	"net"
	"strconv"
	"strings"
	"testing"
	"v2ray.com/core/app/router"
)

func Test(t *testing.T) {

	var gfwIPs CountryIPRange
	if err := proto.Unmarshal(GfwIPs, &gfwIPs); err != nil {

	}
	for i := range gfwIPs.Ips {
		println(gfwIPs.Ips[i].String())
	}

}

func Test1(t *testing.T) {
	ips := &CountryIPRange{
		Ips: make([]*router.CIDR, 0, 8192),
	}
	arr := []string{"apnic|CN|ipv4|91.108.4.0|1024|20110414|allocated", "apnic|CN|ipv4|91.108.8.0|1024|20110412|allocated"}
	for i := range arr {
		line := arr[i]
		line = strings.TrimSpace(line)
		parts := strings.Split(line, "|")
		if len(parts) < 5 {
			continue
		}
		if strings.ToLower(parts[1]) != "cn" || strings.ToLower(parts[2]) != "ipv4" {
			continue
		}
		ip := parts[3]
		count, err := strconv.Atoi(parts[4])
		if err != nil {
			continue
		}
		mask := uint32(math.Floor(math.Log2(float64(count)) + 0.5))
		ipBytes := net.ParseIP(ip)
		if len(ipBytes) == 0 {
			panic("Invalid IP " + ip)
		}
		ips.Ips = append(ips.Ips, &router.CIDR{
			Ip:     []byte(ipBytes)[12:16],
			Prefix: 32 - mask,
		})
	}
}

