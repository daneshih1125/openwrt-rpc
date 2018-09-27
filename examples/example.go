package main

import (
	"fmt"
	"os"

	log "github.com/Sirupsen/logrus"
	wrpc "github.com/daneshih1125/openwrt-rpc"
)

// exmaple for openwrt json rpc client.
// default is http port 80.
func main() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "%s [ip] [username] [password]\n", os.Args[0])
		os.Exit(1)
	}

	ip := os.Args[1]
	username := os.Args[2]
	password := os.Args[3]

	s := &wrpc.RpcServer{
		Hostname: ip,
	}

	auth := &wrpc.Auth{
		Username: username,
		Password: password,
		Timeout:  30,
	}

	w, err := wrpc.New(s, auth)
	if err != nil {
		log.Fatal(err)
	}
	uptime, _ := w.SysRPC("uptime", nil)
	fmt.Println(uptime)
	devList, _ := w.SysRPC("net.devices", nil)
	fmt.Println(devList)

	// Test set UCI value
	enabled, _ := w.UciRPC("get", []string{"system", "ntp", "enabled"})
	fmt.Println(enabled)
	var setValue string
	if enabled == "1" {
		setValue = "0"
	} else {
		setValue = "1"
	}
	w.UciRPC("set", []string{"system", "ntp", "enabled", setValue})
	w.UciRPC("commit", []string{"system"})
	enabled, _ = w.UciRPC("get", []string{"system", "ntp", "enabled"})
	fmt.Println(enabled)
}
