package main

import (
	"flag"
	"fmt"
	"github.com/ubermint/kvnode/server"
	"github.com/ubermint/kvnode/master"
	"log"
	"net"
	"os"
	"io/ioutil"
)

type logWriter struct { }

func (writer logWriter) Write(bytes []byte) (int, error) {
    return fmt.Print("[DEBUG] " + string(bytes))
}

func main() {
	log.SetFlags(0)
    debug := os.Getenv("DEBUG")
	if debug == "1" {
		log.SetOutput(new(logWriter))
		log.Println("DEBUG log enabled")
	} else {
		log.SetOutput(ioutil.Discard) 
	}


	port := flag.Int("port", 8000, "the port number")
	master_flag := flag.Bool("master", false, "master server flag")

	flag.Parse()

	if !*master_flag && len(flag.Args()) == 0 {
		log.Fatal("IP address and port are required on node server.")
	}

	if *master_flag {
		log.Println("Launching as Master...")
		var m master.Master
		m.Port = *port

		m.Run()
	} else {
		log.Println("Launching as Node...")

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
