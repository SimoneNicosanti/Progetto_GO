package peer

import (
	"net/rpc"
	"sync"
)

type ChordPeer struct {
	PeerId   int
	PeerAddr string
}

var N int = 64

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
Ho 64 possibili chiavi --> lo spazio di indirizzamento è di log(N) = 6 bit
fingerTable[0] == predecessore
fingerTable[1] == successore
*/

type FingerTable struct {
	mutex sync.Mutex
	table [2](*PeerConnection)
}

var fingerTable FingerTable = FingerTable{sync.Mutex{}, [2](*PeerConnection){nil, nil}}

var registryClientPtr *rpc.Client

//TODO Meccanismo di Caching per velocizzare algoritmo di Lookup
