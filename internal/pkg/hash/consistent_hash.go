package hash

import (
	"fmt"
	"sort"
	"strconv"
	"sync"
)

const (
	_topWeight   = 100
	_minReplicas = 100
	_prime       = 16777619
)

type (
	Func func(data []byte) uint64
	// ConsistentHash is a structure that implements consistent hashing.
	ConsistentHash struct {
		lock     sync.RWMutex
		ring     map[uint64][]any
		nodes    map[string]struct{}
		keys     []uint64
		hashFunc Func
		replicas int
	}
)

// ConsistentHashOption defines a function type for configuring ConsistentHash.
type ConsistentHashOption func(c *ConsistentHash)

// WithReplicas sets the number of replicas for the ConsistentHash.
func WithReplicas(replicas int) ConsistentHashOption {
	return func(c *ConsistentHash) {
		c.replicas = replicas
	}
}

// WithHashFunc sets the hash function for the ConsistentHash.
func WithHashFunc(hashFunc Func) ConsistentHashOption {
	return func(c *ConsistentHash) {
		c.hashFunc = hashFunc
	}
}

// NewConsistentHash creates and returns a new instance of ConsistentHash.
// default using hash function murmur3 64-bit and 100 replicas
func NewConsistentHash(opts ...ConsistentHashOption) *ConsistentHash {
	c := &ConsistentHash{
		ring:     make(map[uint64][]any),
		nodes:    make(map[string]struct{}),
		hashFunc: Hash,
		replicas: _minReplicas,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Add adds the given node into h.
func (c *ConsistentHash) Add(node any) {
	c.AddWithReplicas(node, c.replicas)
}

// AddWithReplicas adds the given node into h with the specified number of replicas.
func (h *ConsistentHash) AddWithReplicas(node any, replicas int) {
	h.Remove(node)

	if replicas > h.replicas {
		replicas = h.replicas
	}

	nodeRepresent := represent(node)
	h.lock.Lock()
	defer h.lock.Unlock()
	h.addNode(nodeRepresent)

	for i := 0; i < replicas; i++ {
		hash := h.hashFunc([]byte(nodeRepresent + strconv.Itoa(i)))
		h.keys = append(h.keys, hash)
		h.ring[hash] = append(h.ring[hash], node)
	}

	sort.Slice(h.keys, func(i, j int) bool {
		return h.keys[i] < h.keys[j]
	})
}

// AddWithWeight adds the given node into h with the specified weight.
func (h *ConsistentHash) AddWithWeight(node any, weight int) {
	replicas := h.replicas * weight / _topWeight
	h.AddWithReplicas(node, replicas)
}

// Get returns the node responsible for the given value v.
func (h *ConsistentHash) Get(v any) (any, bool) {
	h.lock.RLock()
	defer h.lock.RUnlock()

	if len(h.ring) == 0 {
		return nil, false
	}

	hash := h.hashFunc([]byte(represent(v)))
	index := sort.Search(len(h.keys), func(i int) bool {
		return h.keys[i] >= hash
	}) % len(h.keys)

	nodes := h.ring[h.keys[index]]
	switch len(nodes) {
	case 0:
		return nil, false
	case 1:
		return nodes[0], true
	default:
		innerIndex := h.hashFunc([]byte(innerRepresent(v)))
		pos := int(innerIndex % uint64(len(nodes)))
		return nodes[pos], true
	}
}

// Remove removes the given node from h.
func (h *ConsistentHash) Remove(node any) {
	nodeRepresent := represent(node)

	h.lock.Lock()
	defer h.lock.Unlock()

	if !h.containsNode(nodeRepresent) {
		return
	}

	for i := 0; i < h.replicas; i++ {
		hash := h.hashFunc([]byte(nodeRepresent + strconv.Itoa(i)))
		index := sort.Search(len(h.keys), func(i int) bool {
			return h.keys[i] >= hash
		})
		if index < len(h.keys) && h.keys[index] == hash {
			h.keys = append(h.keys[:index], h.keys[index+1:]...)
		}
		h.removeRingNode(hash, nodeRepresent)
	}

	h.removeNode(nodeRepresent)
}

// removeRingNode removes the node with the given representation from the ring at the specified hash.
func (h *ConsistentHash) removeRingNode(hash uint64, nodeRepresent string) {
	if nodes, ok := h.ring[hash]; ok {
		newNodes := nodes[:0]
		for _, x := range nodes {
			if represent(x) != nodeRepresent {
				newNodes = append(newNodes, x)
			}
		}
		if len(newNodes) > 0 {
			h.ring[hash] = newNodes
		} else {
			delete(h.ring, hash)
		}
	}
}

// addNode adds the node with the given representation to the set of nodes.
func (h *ConsistentHash) addNode(nodeRepresent string) {
	h.nodes[nodeRepresent] = struct{}{}
}

// containsNode checks if the node with the given representation exists in the set of nodes.
func (h *ConsistentHash) containsNode(nodeRepresent string) bool {
	_, ok := h.nodes[nodeRepresent]
	return ok
}

// removeNode removes the node with the given representation from the set of nodes.
func (h *ConsistentHash) removeNode(nodeRepresent string) {
	delete(h.nodes, nodeRepresent)
}

// innerRepresent generates an inner representation for the given node.
func innerRepresent(node any) string {
	return fmt.Sprintf("%d:%v", _prime, node)
}

// represent generates a string representation for the given node.
func represent(node any) string {
	return fmt.Sprintf("%v", node)
}
