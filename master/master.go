package master

import (
	"context"
	"fmt"
	"github.com/valyala/fasthttp"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type Master struct {
	Port      int
	Ring      *HashRing
	Nodes     []Node
	NodesLock sync.Mutex
	KeyMap    sync.Map
}

func (m *Master) requestRouter(ctx *fasthttp.RequestCtx) {
	switch string(ctx.Path()) {
	case "/get":
		m.HTTPGet(ctx)
	case "/set":
		m.HTTPSet(ctx)
	case "/update":
		m.HTTPUpdate(ctx)
	case "/delete":
		m.HTTPDelete(ctx)
	case "/join":
		m.HTTPJoin(ctx)
	case "/leave":
		m.HTTPLeave(ctx)
	default:
		ctx.Error("Not Found", fasthttp.StatusNotFound)
	}
}

func (m *Master) Run() {
	m.Nodes = []Node{}
	m.Ring = NewHashRing()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	httpsrv := &fasthttp.Server{
		Handler:      m.requestRouter,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		log.Println(fmt.Sprintf("HTTP server listening on port %d.", m.Port))
		if err := httpsrv.ListenAndServe(fmt.Sprintf(":%d", m.Port)); err != nil {
			log.Fatalf("Error: %s\n", err)
		}
	}()

	<-shutdown

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := httpsrv.ShutdownWithContext(ctx); err != nil {
		log.Fatal(err)
	}

	log.Println("Server stopped.")
}
