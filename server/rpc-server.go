package server

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/ubermint/kv/storage"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"net/url"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type Server struct {
	nodeID     string
	Port       int
	MasterAddr net.TCPAddr
	db         *storage.Storage
	isPersist  bool
	mutex      sync.RWMutex
}

func (srv *Server) JoinCluster() error {
	params := url.Values{}
	params.Set("id", srv.nodeID)
	params.Set("port", fmt.Sprintf("%d", srv.Port))

	to_url := fmt.Sprintf("http://%s:%d/join?%s",
		srv.MasterAddr.IP.String(), srv.MasterAddr.Port, params.Encode())

	resp, err := http.Get(to_url)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Error: Request failed ")
	}

	return nil
}

func (srv *Server) LeaveCluster() error {
	params := url.Values{}
	params.Set("id", srv.nodeID)

	url := fmt.Sprintf("http://%s:%d/leave?%s",
		srv.MasterAddr.IP.String(), srv.MasterAddr.Port, params.Encode())

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Error: Request failed ")
	}

	return nil
}

func (srv *Server) Setup() error {
	nodeID_env := os.Getenv("NODE_ID")
	if nodeID_env == "" {
		srv.isPersist = false
		uuidObj, err := uuid.NewRandom()
		if err != nil {
			log.Fatal("Failed to generate UUID: ", err)
		}
		nodeID_env = uuidObj.String()
		if err != nil {
			return err
		}
	} else {
		srv.isPersist = true
	}

	srv.nodeID = nodeID_env
	work_dir := fmt.Sprintf("data/%s/", srv.nodeID)
	log.Println("Storage: ", work_dir)

	srv.db = &storage.Storage{}
	err := srv.db.New(work_dir)
	if err != nil {
		log.Fatal(err)
	}

    err = srv.JoinCluster()
    if err != nil {
        log.Fatal(err)
    }


	return nil
}

func (srv *Server) Run() {
	err := srv.Setup()
	if err != nil {
		log.Fatal(err)
	}

	err = rpc.RegisterName("Srv", srv)

	if err != nil {
		log.Fatal("Failed to register RPC service:", err)
	}

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		listener, err := net.Listen("tcp", fmt.Sprintf(":%d", srv.Port))
		if err != nil {
            //err = srv.LeaveCluster()
           // if err != nil {
             //   log.Fatal(err)
           // }
			log.Fatal("Failed to start RPC server:", err)
		}

		log.Println("RPC server started on", listener.Addr())
		rpc.Accept(listener)
		log.Println(fmt.Sprintf("Joined master at %s:%d.", srv.MasterAddr.IP.String(), srv.MasterAddr.Port))
	}()

	<-shutdown

	if srv.isPersist {
		srv.db.Close()
	} else {
		srv.db.Destroy()
	}

	err = srv.LeaveCluster()
	if err != nil {
		log.Fatal(err)
	}

	log.Println(fmt.Sprintf("Leaved master at %s:%d.", srv.MasterAddr.IP.String(), srv.MasterAddr.Port))

	_, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	log.Println("RPC server stopped.")
}
