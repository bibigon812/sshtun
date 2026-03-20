package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/bibigon812/sshtun"
)

func main() {
	index := flag.Int("index", 0, "tunnel index")
	hostname := flag.String("hostname", "", "remote hostname to connect with")
	localAddress := flag.String("local-address", "", "local ip addres of the tunnel interface")
	remoteAddress := flag.String("remote-address", "", "remote ip addres of the tunnel interface")

	flag.Parse()

	if *hostname == "" {
		fmt.Println("Error: The -hostname flag is required.")
		flag.PrintDefaults()
		os.Exit(1)
	}

	if *localAddress == "" {
		fmt.Println("Error: The -local-address flag is required.")
		flag.PrintDefaults()
		os.Exit(1)
	}

	if *remoteAddress == "" {
		fmt.Println("Error: The -remote-address flag is required.")
		flag.PrintDefaults()
		os.Exit(1)
	}

	sshtun.Create(
		*hostname,
		"root",
		22,
		*index,
		[]string{*remoteAddress},
		[]string{*localAddress},
	)
}
