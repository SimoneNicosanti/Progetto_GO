package main

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/rpc"
	"sync"
)

type PeerMap struct {
	mutex   sync.Mutex
	peerMap map[int](*string)
}

var N int = 64

var myMap PeerMap = PeerMap{sync.Mutex{}, make(map[int](*string))}

type Registry int

type ChordPeer struct {
	PeerId   int
	PeerAddr string
}

type SuccAndPred struct {
	SuccPeerPtr *ChordPeer
	PredPeerPtr *ChordPeer
}

//Aggiungere PING ai vari peer per verificare che siano ancora attivi in modo da pulire il registry periodicamente

func main() {

	registry := new(Registry)
	err := rpc.Register(registry)

	if err != nil {
		fmt.Println("Impossibile registrare il servizio")
		fmt.Printf("Errore: '%s'", err.Error())
		return
	}

	rpc.HandleHTTP()
	listener, err := net.Listen("tcp", ":1234")
	if err != nil {
		fmt.Println("Impossibile eseguire listen")
		fmt.Printf("Errore: '%s'", err.Error())
		return
	}

	for {
		http.Serve(listener, nil)
	}
}

func (t *Registry) PeerJoin(chordPeerPtr *ChordPeer, replyPtr *SuccAndPred) error {
	peer := chordPeerPtr.PeerId

	myMap.mutex.Lock()
	defer myMap.mutex.Unlock()

	if myMap.peerMap[peer] != nil {
		return errors.New("esiste già un peer con stesso id")
	}
	myMap.peerMap[peer] = &chordPeerPtr.PeerAddr

	for key := range myMap.peerMap {
		fmt.Printf("%d ", key)
	}
	println()

	// Trovo il successore e il predecessore del nodo e rispondo con quelli
	// Se non faccio così posso rispondere con un nodo casuale della rete a cui poi viene chiesto di trovare successore e predecessore
	findSuccessorAndPredecessor(chordPeerPtr.PeerId, replyPtr)

	return nil
}

func findSuccessorAndPredecessor(peerId int, succAndPredPtr *SuccAndPred) {

	keySlice := make([]int, 0, 10)
	for key := range myMap.peerMap {
		if key != peerId {
			keySlice = append(keySlice, key)
		}
	}

	if len(keySlice) == 0 {
		// è il primo nodo che entra nella rete
		succAndPredPtr.PredPeerPtr = nil
		succAndPredPtr.SuccPeerPtr = nil
		return
	}

	succKey := N
	predKey := -1
	minKey := N
	maxKey := -1

	for index := range keySlice {
		// Cerco successore
		key := keySlice[index]
		if key > peerId && key < succKey {
			succKey = key
		}
		if key < peerId && key > predKey {
			predKey = key
		}
		if key < minKey {
			minKey = key
		}
		if key > maxKey {
			maxKey = key
		}
	}

	if succKey == N {
		succKey = minKey
	}
	if predKey == -1 {
		predKey = maxKey
	}

	succAndPredPtr.PredPeerPtr = &ChordPeer{predKey, *myMap.peerMap[predKey]}
	succAndPredPtr.SuccPeerPtr = &ChordPeer{succKey, *myMap.peerMap[succKey]}
}
