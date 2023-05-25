package master

import (
	"encoding/binary"
	"fmt"
	"golang.org/x/crypto/blake2b"
	"sort"
)

type NodeID string

type Node struct {
	ID     NodeID
	IP     string
	port   string
	status string
}

func (m *Master) GetNodeByID(nodeID NodeID) *Node {
	for i, item := range m.Nodes {
		if item.ID == nodeID {
			return &m.Nodes[i]
		}
	}
	return &Node{}
}

type HashRing struct {
	sortedHash   []uint32
	hashMap      map[uint32]NodeID
	isReplicated bool
}

func NewHashRing() *HashRing {
	return &HashRing{
		sortedHash: make([]uint32, 0, 0),
		hashMap:    make(map[uint32]NodeID),
	}
}

func (m *Master) GetNodeInfo(node Node) string {
	return fmt.Sprintf("%s:%s", node.IP, node.port)
}

func (hr *HashRing) GetReplicationNodes(key string) [3]NodeID {
	var hosts = [3]NodeID{NodeID(""), NodeID(""), NodeID("")}

	keyHash := hash(key)

	if len(hr.sortedHash) == 0 {
		return hosts
	}

	if hr.isReplicated {
		for i, item := range hr.sortedHash {
			if keyHash >= item {
				hosts[0] = hr.hashMap[hr.sortedHash[i]]
				hosts[1] = hr.hashMap[hr.sortedHash[(i+1)%len(hr.sortedHash)]]
				hosts[2] = hr.hashMap[hr.sortedHash[(i+2)%len(hr.sortedHash)]]
				return hosts
			}
		}

		hosts[0] = hr.hashMap[hr.sortedHash[0]]
		hosts[1] = hr.hashMap[hr.sortedHash[1]]
		hosts[2] = hr.hashMap[hr.sortedHash[2]]
		return hosts
	}

	for _, item := range hr.sortedHash {
		if keyHash >= item {
			hosts[0] = hr.hashMap[item]
			return hosts
		}
	}
	hosts[0] = hr.hashMap[hr.sortedHash[0]]
	return hosts
}

func (hr *HashRing) AddNode(nodeID NodeID) {
	nodeHash := hash(string(nodeID))
	hr.hashMap[nodeHash] = nodeID

	hr.sortedHash = append(hr.sortedHash, nodeHash)

	sort.Slice(hr.sortedHash, func(i, j int) bool {
		return hr.sortedHash[i] > hr.sortedHash[j]
	})
}

func (hr *HashRing) RemoveNode(nodeID NodeID) {
	nodeHash := hash(string(nodeID))
	delete(hr.hashMap, nodeHash)

	for i, item := range hr.sortedHash {
		if item == nodeHash {
			hr.sortedHash = append(hr.sortedHash[:i], hr.sortedHash[i+1:]...)
			break
		}
	}
}

func (hr *HashRing) GetNode(key string) NodeID {
	keyHash := hash(key)

	for _, item := range hr.sortedHash {
		if keyHash >= item {
			return hr.hashMap[item]
		}
	}

	return hr.hashMap[hr.sortedHash[0]]
}

func hash(data string) uint32 {
	hash := blake2b.Sum256([]byte(data))
	hashUint32 := binary.BigEndian.Uint32(hash[:4])
	return hashUint32
}

/*

func main() {
	hr := NewHashRing()

	for i := 0; i < 10; i++ {
		node := NodeID(fmt.Sprintf("node%d", i))
		hr.AddNode(node)
	}

	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("key%d", i)
		nodeid := hr.GetNode(key)
		//hr.AddNode(nodeid)
		fmt.Println("NodeID for ", key, hash(key), string(nodeid), hash(string(nodeid)))
	}
}

func main() {
	hr := NewHashRing()

	for i := 0; i < 10; i++ {
		node := NodeID(fmt.Sprintf("node%d", i))
		hr.AddNode(node)
	}

	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("key%d", i)
		nodeid := hr.GetNode(key)
		//hr.AddNode(nodeid)
		fmt.Println("NodeID for ", key, hash(key), string(nodeid), hash(string(nodeid)))
	}
}
*/
