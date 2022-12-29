package peer

import (
	"math"
	"net/rpc"
	"sync"
	"time"
)

type ChordPeer struct {
	PeerId   int
	PeerAddr string
}

const bitNumber int = 6

var N int = int(math.Pow(2, 6))

type Resource struct {
	Value string
}

type ResourceMap struct {
	mutex  sync.Mutex
	resMap map[int](*Resource)
}

var resourceMap ResourceMap = ResourceMap{sync.Mutex{}, make((map[int](*Resource)))}

type PeerConnection struct {
	Peer      ChordPeer
	ClientPtr *rpc.Client
}

/*
fingerTable[0] == predecessore
fingerTable[1] == successore
*/

type FingerTable struct {
	mutex sync.Mutex
	table [bitNumber + 1](*PeerConnection)
}

// TODO verificare che inizializzazione sia fatta in modo corretto
var fingerTable FingerTable = FingerTable{sync.Mutex{}, [bitNumber + 1](*PeerConnection){}}

var registryClientPtr *rpc.Client

type CacheTuple struct {
	resource  *Resource
	entryTime time.Time
}

type Cache struct {
	mutex    sync.Mutex
	cacheMap map[int](CacheTuple)
}

var peerCache Cache = Cache{sync.Mutex{}, make((map[int](CacheTuple)))}
