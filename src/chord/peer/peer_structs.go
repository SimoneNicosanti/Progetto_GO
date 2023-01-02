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

const CACHE_RES_TTL time.Duration = 30

const CACHE_TIMER time.Duration = 30

const STABILIZE_TIMER time.Duration = 30

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

// Struttura che mi serve per dire chi è il nuovo predecessore del peer e quali chiavi il peer deve predere in carico a seguito del cambio predecessore
type PredecessorInfo struct {
	Peer           ChordPeer
	PredecessorMap map[int](*Resource)
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
