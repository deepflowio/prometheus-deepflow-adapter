package utils

import (
	"net"
	"os"
	"path/filepath"
)

func GetProcessName() string {
	path, err := os.Executable()
	if err != nil {
		return ""
	}
	return filepath.Base(path)
}

func GetLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, addr := range addrs {
		ipAddr, ok := addr.(*net.IPNet)
		if !ok {
			continue
		}
		if ipAddr.IP.IsLoopback() {
			continue
		}

		if !ipAddr.IP.IsGlobalUnicast() {
			continue
		}
		return ipAddr.IP.String()
	}
	return ""
}

func GetLocalIPWithDial() string {
	conn, err := net.Dial("udp", "8.8.8.8:53")
	if err != nil {
		return ""
	}
	lAddr := conn.LocalAddr().(*net.UDPAddr)
	return lAddr.IP.String()
}
