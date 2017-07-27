package util

import (
	"barrage/logger"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
)

func InternalIp() string {
	addrs, err := net.InterfaceAddrs()

	if err != nil {
		logger.Error("net.InterfaceAddrs error (%v)", err)
		panic(err)
	}

	for _, address := range addrs {

		// 检查ip地址判断是否回环地址
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}

		}
	}
	return ""
}

func ExternalIp() string {
	resp, err := http.Get("http://myexternalip.com/raw")
	if err != nil {
		logger.Error("http.Get external ip error %v", err)
		panic(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logger.Error("ioutil.ReadAll error %v", err)
		panic(err)
	}
	return strings.TrimRight(string(body), "\n")
}
