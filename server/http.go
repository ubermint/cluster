package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/rpc"
	"strings"
)

// StatusBadRequest                   = 400 // RFC 9110, 15.5.1
// StatusNotFound                     = 404 // RFC 9110, 15.5.5

type KeyValue struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func (m *Master) LogClients(clients [3]NodeID) {
	for _, nd := range clients {
		s := fmt.Sprintf("[%s](%s)  ", m.GetNodeInfo(*m.GetNodeByID(nd)), string(nd))
		fmt.Println(s)
	}
}

func (m *Master) handleGet(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")

	log.Println(fmt.Sprintf("GET(%s)", key))

	var response KeyValue
	var clients, ok = m.KeyMap[hash(key)]

	m.LogClients(clients)

	if !ok {
		log.Println("No such key in KeyMap")
		w.WriteHeader(http.StatusNotFound)
		return
	}

	for _, client := range clients {
		if client != NodeID("") {
			rpcHost := m.GetNodeByID(client)

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
				Key: []byte(key),
			}
			var getResult GetResult

			err = conn.Call("Srv.Get", getArgs, &getResult)
			if err != nil {
				//rpcHost.status = "Failed"
				continue
			}

			response = KeyValue{
				Key:   key,
				Value: string(getResult.Value),
			}

			log.Println("Get from ", m.GetNodeInfo(*rpcHost))

			err = conn.Close()
			break
		}
	}

	blank := KeyValue{}

	if response == blank {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	jsonResponse, err := json.Marshal(response)
	if err != nil {
		log.Println("Error marshaling JSON:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonResponse)
}

func (m *Master) handleSet(w http.ResponseWriter, r *http.Request) {
	var kv KeyValue
	err := json.NewDecoder(r.Body).Decode(&kv)
	if err != nil {
		log.Println("Error decoding JSON:", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	log.Println(fmt.Sprintf("SET(%s,%s)", kv.Key, kv.Value))
	var clients = m.Ring.GetReplicationNodes(kv.Key)
	m.LogClients(clients)
	succ := 0

	for _, client := range clients {
		if client != NodeID("") {
			rpcHost := m.GetNodeByID(client)

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
				succ += 1
				log.Println("Replicated to ", m.GetNodeInfo(*rpcHost))
			}

			err = conn.Close()
		}
	}

	if succ == 0 {
		log.Println("SET Failed")
		w.WriteHeader(http.StatusNotFound)
		return
	}

	m.KeyMap[hash(kv.Key)] = clients
	w.WriteHeader(http.StatusOK)
}

func (m *Master) handleUpdate(w http.ResponseWriter, r *http.Request) {
	var kv KeyValue
	err := json.NewDecoder(r.Body).Decode(&kv)
	if err != nil {
		log.Println("Error decoding JSON:", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	log.Println(fmt.Sprintf("UPDATE(%s,%s)", kv.Key, kv.Value))
	var clients, ok = m.KeyMap[hash(kv.Key)]

	if !ok {
		log.Println("No such key in KeyMap")
		w.WriteHeader(http.StatusNotFound)
		return
	}

	m.LogClients(clients)
	succ := 0

	for _, client := range clients {
		if client != NodeID("") {
			rpcHost := m.GetNodeByID(client)

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
				log.Println("Failed to call Del method:", m.GetNodeInfo(*rpcHost), err)
				continue
			}

			if updateResult.Success {
				succ += 1
				log.Println("Update to ", m.GetNodeInfo(*rpcHost))
			}

			err = conn.Close()
		}
	}

	if succ == 0 {
		log.Println("UPDATE Failed")
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (m *Master) handleDelete(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")

	log.Println(fmt.Sprintf("DELETE(%s)", key))
	var clients, ok = m.KeyMap[hash(key)]

	m.LogClients(clients)

	if !ok {
		log.Println("No such key in KeyMap")
		w.WriteHeader(http.StatusNotFound)
		return
	}

	succ := 0
	for _, client := range clients {
		if client != NodeID("") {
			rpcHost := m.GetNodeByID(client)

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
				Key: []byte(key),
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
				succ += 1
			}

			err = conn.Close()
		}
	}

	if succ == 0 {
		log.Println("DEL Failed")
		w.WriteHeader(http.StatusNotFound)
		return
	}

	delete(m.KeyMap, hash(key))
	w.WriteHeader(http.StatusOK)
}

func (m *Master) handleJoin(w http.ResponseWriter, r *http.Request) {
	id := NodeID(r.URL.Query().Get("id"))
	ip := strings.Split(r.RemoteAddr, ":")[0]
	port := r.URL.Query().Get("port")

	m.NodesLock.Lock()
	defer m.NodesLock.Unlock()

	for i, node := range m.Nodes {
		if node.ID == id {
			m.Nodes[i].status = "Active"
			w.WriteHeader(http.StatusOK)
			return
		}
	}

	m.Nodes = append(m.Nodes, Node{id, ip, port, "Active"})
	s := fmt.Sprintf("Joined to the cluster: %s:%s (%s)  ", ip, port, id)
	log.Println(s)
	m.Ring.AddNode(id)

	if len(m.Nodes) == 6 {
		log.Println("Replication is enabled.")
	}

	w.WriteHeader(http.StatusOK)
}

func (m *Master) handleLeave(w http.ResponseWriter, r *http.Request) {
	id := NodeID(r.URL.Query().Get("id"))
	ip := strings.Split(r.RemoteAddr, ":")[0]

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
		log.Println("Replication is disabled.")
	}

	w.WriteHeader(http.StatusOK)
}
