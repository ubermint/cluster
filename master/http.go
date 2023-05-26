package master

import (
	"encoding/json"
	"fmt"
	. "github.com/ubermint/cluster/server"
	"github.com/valyala/fasthttp"
	"log"
	"net/rpc"
	"strings"
)

// StatusOK                           = 200 // RFC 7231, 6.3.1
// StatusBadRequest                   = 400 // RFC 9110, 15.5.1
// StatusNotFound                     = 404 // RFC 9110, 15.5.5
// StatusInternalServerError          = 500 // RFC 7231, 6.6.1

type KeyValue struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func (m *Master) LogClients(hosts [3]NodeID) {
	for _, nd := range hosts {
		if string(nd) != "" {
			info := m.GetNodeInfo(*m.GetNodeByID(nd))
			s := fmt.Sprintf("[%s](%s)", info, string(nd))
			log.Println(s)
		}
	}
}

func (m *Master) HTTPGet(ctx *fasthttp.RequestCtx) {
	key := string(ctx.QueryArgs().Peek("key"))
	log.Println(fmt.Sprintf("GET(%s)", key))

	kv := &KeyValue{
		Key:   key,
		Value: "",
	}

	var hosts [3]NodeID
	hh, ok := m.KeyMap.Load(hash(key))
	if ok {
		hosts = hh.([3]NodeID)
		m.LogClients(hosts)
	} else {
		log.Println("No such key in KeyMap")
		ctx.SetStatusCode(fasthttp.StatusNotFound)
		return
	}

	ok = m.handleGet(kv, hosts)
	if !ok {
		ctx.SetStatusCode(fasthttp.StatusNotFound)
		return
	}

	responseJSON, err := json.Marshal(kv)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		return
	}

	ctx.SetContentType("application/json")
	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.Write(responseJSON)
}

func (m *Master) handleGet(kv *KeyValue, hosts [3]NodeID) bool {
	for _, host := range hosts {
		if host != NodeID("") {
			rpcHost := m.GetNodeByID(host)

			if rpcHost.status == "Failed" {
				continue
			}
			conn, err := rpc.Dial("tcp", m.GetNodeInfo(*rpcHost))

			if err != nil {
				log.Println("Failed to connect to RPC server:", m.GetNodeInfo(*rpcHost))
				rpcHost.status = "Failed"
				continue
			}

			getArgs := GetArgs{
				Key: []byte(kv.Key),
			}
			var getResult GetResult

			err = conn.Call("Srv.Get", getArgs, &getResult)
			if err != nil {
				//rpcHost.status = "Failed"
				continue
			}

			log.Println("Get from ", m.GetNodeInfo(*rpcHost))
			err = conn.Close()
			kv.Value = string(getResult.Value)

			return true
		}
	}

	return false
}

func (m *Master) HTTPSet(ctx *fasthttp.RequestCtx) {
	ctx.Response.Header.Del("Server")
	ctx.Response.Header.Del("Date")
	var kv KeyValue
	err := json.Unmarshal(ctx.PostBody(), &kv)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		log.Println("Invalid JSON payload")
		return
	}

	log.Println(fmt.Sprintf("SET(%s,%s)", kv.Key, kv.Value))
	var hosts = m.Ring.GetReplicationNodes(kv.Key)
	m.LogClients(hosts)

	ok := m.handleSet(&kv, hosts)
	if !ok {
		ctx.SetStatusCode(fasthttp.StatusNotFound)
		return
	}

	m.KeyMap.Store(hash(kv.Key), hosts)
	ctx.SetStatusCode(fasthttp.StatusOK)
}

func (m *Master) handleSet(kv *KeyValue, hosts [3]NodeID) bool {
	res := 0
	for _, host := range hosts {
		if host != NodeID("") {
			rpcHost := m.GetNodeByID(host)

			if rpcHost.status == "Failed" {
				continue
			}

			conn, err := rpc.Dial("tcp", m.GetNodeInfo(*rpcHost))

			if err != nil {
				log.Println("Failed to connect to RPC server: ", m.GetNodeInfo(*rpcHost))
				rpcHost.status = "Failed"
				continue
			}

			setArgs := SetArgs{
				Key:   []byte(kv.Key),
				Value: []byte(kv.Value),
			}
			var setResult SetResult

			err = conn.Call("Srv.Set", setArgs, &setResult)
			if err != nil {
				continue
			}

			if setResult.Success {
				res += 1
				log.Println("Set to ", m.GetNodeInfo(*rpcHost))
			}

			err = conn.Close()
		}
	}

	if res == 0 {
		log.Println("SET Failed")
		return false
	}
	return true
}

func (m *Master) HTTPUpdate(ctx *fasthttp.RequestCtx) {
	var kv KeyValue
	err := json.Unmarshal(ctx.PostBody(), &kv)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		log.Println("Invalid JSON payload")
		return
	}

	log.Println(fmt.Sprintf("UPDATE(%s,%s)", kv.Key, kv.Value))

	var hosts [3]NodeID
	hh, ok := m.KeyMap.Load(hash(kv.Key))
	if ok {
		hosts = hh.([3]NodeID)
		m.LogClients(hosts)
	} else {
		log.Println("No such key in KeyMap")
		ctx.SetStatusCode(fasthttp.StatusNotFound)
		return
	}

	ok = m.handleUpdate(&kv, hosts)
	if !ok {
		ctx.SetStatusCode(fasthttp.StatusNotFound)
		return
	}

	ctx.SetStatusCode(fasthttp.StatusOK)
}

func (m *Master) handleUpdate(kv *KeyValue, hosts [3]NodeID) bool {
	res := 0
	for _, host := range hosts {
		if host != NodeID("") {
			rpcHost := m.GetNodeByID(host)

			if rpcHost.status == "Failed" {
				continue
			}

			conn, err := rpc.Dial("tcp", m.GetNodeInfo(*rpcHost))

			if err != nil {
				log.Println("Failed to connect to RPC server: ", m.GetNodeInfo(*rpcHost))
				rpcHost.status = "Failed"
				continue
			}

			updateArgs := UpdateArgs{
				Key:   []byte(kv.Key),
				Value: []byte(kv.Value),
			}
			var updateResult UpdateResult

			err = conn.Call("Srv.Update", updateArgs, &updateResult)
			if err != nil {
				log.Println("Failed to call Update method:", m.GetNodeInfo(*rpcHost), err)
				continue
			}

			if updateResult.Success {
				res += 1
				log.Println("Update to ", m.GetNodeInfo(*rpcHost))
			}

			err = conn.Close()
		}
	}

	if res == 0 {
		log.Println("UPDATE Failed")
		return false
	}

	return true
}

func (m *Master) HTTPDelete(ctx *fasthttp.RequestCtx) {
	key := string(ctx.QueryArgs().Peek("key"))
	log.Println(fmt.Sprintf("DELETE(%s)", key))

	kv := &KeyValue{
		Key:   key,
		Value: "",
	}

	var hosts [3]NodeID
	hh, ok := m.KeyMap.Load(hash(kv.Key))
	if ok {
		hosts = hh.([3]NodeID)
		m.LogClients(hosts)
	} else {
		log.Println("No such key in KeyMap")
		ctx.SetStatusCode(fasthttp.StatusNotFound)
		return
	}

	ok = m.handleDel(kv, hosts)
	if !ok {
		log.Println("DEL Failed")
		ctx.SetStatusCode(fasthttp.StatusNotFound)
		return
	}

	m.KeyMap.Delete(hash(key))
	ctx.SetStatusCode(fasthttp.StatusOK)
}

func (m *Master) handleDel(kv *KeyValue, hosts [3]NodeID) bool {
	res := 0
	for _, host := range hosts {
		if host != NodeID("") {
			rpcHost := m.GetNodeByID(host)

			if rpcHost.status == "Failed" {
				continue
			}

			conn, err := rpc.Dial("tcp", m.GetNodeInfo(*rpcHost))

			if err != nil {
				log.Println("Failed to connect to RPC server:", m.GetNodeInfo(*rpcHost))
				rpcHost.status = "Failed"
				continue
			}

			delArgs := DelArgs{
				Key: []byte(kv.Key),
			}
			var delResult DelResult

			err = conn.Call("Srv.Del", delArgs, &delResult)
			if err != nil {
				log.Println("Failed to call Del method:", m.GetNodeInfo(*rpcHost), err)
				//rpcHost.status = "Failed"
				continue
			}

			if delResult.Success {
				log.Println("Deleted at ", m.GetNodeInfo(*rpcHost))
				res += 1
			}

			err = conn.Close()
		}
	}

	if res == 0 {
		log.Println("DELETE Failed")
		return false
	}
	return true
}

func (m *Master) HTTPJoin(ctx *fasthttp.RequestCtx) {
	id := NodeID(string(ctx.QueryArgs().Peek("id")))
	ip := strings.Split(ctx.RemoteAddr().String(), ":")[0]
	port := string(ctx.QueryArgs().Peek("port"))

	m.NodesLock.Lock()
	defer m.NodesLock.Unlock()

	for i, node := range m.Nodes {
		if node.ID == id {
			m.Nodes[i].status = "Active"
			ctx.SetStatusCode(fasthttp.StatusOK)
			return
		}
	}

	m.Nodes = append(m.Nodes, Node{id, ip, port, "Active"})
	s := fmt.Sprintf("Joined to the cluster: %s:%s (%s)  ", ip, port, id)
	log.Println(s)
	m.Ring.AddNode(id)

	if len(m.Nodes) == 6 {
		m.Ring.isReplicated = true
		log.Println("Replication is enabled.")
	}

	ctx.SetStatusCode(fasthttp.StatusOK)
}

func (m *Master) HTTPLeave(ctx *fasthttp.RequestCtx) {
	id := NodeID(string(ctx.QueryArgs().Peek("id")))
	ip := strings.Split(ctx.RemoteAddr().String(), ":")[0]

	m.NodesLock.Lock()
	defer m.NodesLock.Unlock()

	var port string
	var nid string

	for i, node := range m.Nodes {
		if node.ID == id {
			port = node.port
			nid = string(node.ID)
			m.Nodes = append(m.Nodes[:i], m.Nodes[i+1:]...)
			break
		}
	}

	m.Ring.RemoveNode(id)

	s := fmt.Sprintf("Left the cluster: %s:%s (%s)  ", ip, port, nid)
	log.Println(s)

	if len(m.Nodes) == 5 {
		m.Ring.isReplicated = false
		log.Println("Replication is disabled.")
	}

	ctx.SetStatusCode(fasthttp.StatusOK)
}
