package main

import (
	"flag"
	"fmt"
	"github.com/ubermint/kvnode/server"
	"github.com/ubermint/kvnode/master"
	"log"
	"net"
)

func main() {
	port := flag.Int("port", 8000, "the port number")
	master_flag := flag.Bool("master", false, "master server")

	flag.Parse()

	if !*master_flag && len(flag.Args()) == 0 {
		log.Fatal("IP address and port are required on node server.")
	}

	if *master_flag {
		fmt.Println("Launching as Master...")
		var m master.Master
		m.Port = *port

		m.Run()
	} else {
		fmt.Println("Launching as Node...")

		addr := ""
		addr = flag.Args()[0]
		tcpAddr, err := net.ResolveTCPAddr("tcp", addr)

		if err != nil {
			log.Fatal("Invalid address: ", err)
			return
		}

		var srv server.Server
		srv.Port = *port
		srv.MasterAddr = *tcpAddr

		srv.Run()
	}
}
