package master

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type Master struct {
	Port int
	*http.Server
	Ring      *HashRing
	Nodes     []Node
	NodesLock sync.Mutex
	ReqLock sync.Mutex
	//KeyMap    map[uint32][3]NodeID
	KeyMap    sync.Map
}

func (m *Master) Routes() http.Handler {
	router := http.NewServeMux()

	router.HandleFunc("/set", m.handleSet)
	router.HandleFunc("/get", m.handleGet)
	router.HandleFunc("/update", m.handleUpdate)
	router.HandleFunc("/delete", m.handleDelete)
	router.HandleFunc("/join", m.handleJoin)
	router.HandleFunc("/leave", m.handleLeave)

	return router
}

func (m *Master) Setup() error {
	m.Nodes = []Node{}
	m.Ring = NewHashRing()
	//m.KeyMap = make(map[uint32][3]NodeID)

	return nil
}

func (m *Master) Run() {
	err := m.Setup()
	if err != nil {
		log.Fatal(err)
	}

	m.Server = &http.Server{
		Addr:    fmt.Sprintf(":%d", m.Port),
		Handler: m.Routes(),
	}

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Println(fmt.Sprintf("HTTP server listening on port %d.", m.Port))
		if err := m.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Error: %s\n", err)
		}
	}()

	<-shutdown

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := m.Shutdown(ctx); err != nil {
		log.Fatal(err)
	}

	log.Println("Server stopped")
}
