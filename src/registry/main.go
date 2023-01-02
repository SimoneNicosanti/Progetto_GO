package main

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/rpc"
	"sync"
	"time"
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

// TODO Valutare come parallelizzare meglio l'accesso al registry --> Il mutex blocca tutti, anche coloro che vogliono solo leggere e non scrivere !!

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

	go verifyActivePeer()

	for {
		http.Serve(listener, nil)
	}
}

func verifyActivePeer() {
	for {
		timer := time.NewTimer(5 * time.Second)
		<-timer.C
		myMap.mutex.Lock()

		for peerId, addrPtr := range myMap.peerMap {
			peerConn, err := rpc.DialHTTP("tcp", *addrPtr)
			if err != nil {
				delete(myMap.peerMap, peerId)
				fmt.Println(err.Error())
				continue
			}

			arg := 0
			err = peerConn.Call("ChordPeer.Ping", &arg, nil)
			if err != nil {
				delete(myMap.peerMap, peerId)
				fmt.Println(err.Error())
			}
			fmt.Printf("%d ", peerId)
		}
		fmt.Println()
		myMap.mutex.Unlock()
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
	findSuccessorAndPredecessor(chordPeerPtr, replyPtr)

	return nil
}

func (t *Registry) PeerLeave(chordPeerPtr *ChordPeer, replyPtr *int) error {
	myMap.mutex.Lock()
	defer myMap.mutex.Unlock()

	delete(myMap.peerMap, chordPeerPtr.PeerId)

	return nil
}

func (t *Registry) GetSuccessorAndPredecessor(peerPtr *ChordPeer, succAndPredPtr *SuccAndPred) error {
	myMap.mutex.Lock()
	defer myMap.mutex.Unlock()

	findSuccessorAndPredecessor(peerPtr, succAndPredPtr)

	return nil
}

func findSuccessorAndPredecessor(peerPtr *ChordPeer, succAndPredPtr *SuccAndPred) {
	keySlice := make([]int, 0, 10)
	for key := range myMap.peerMap {
		if key != peerPtr.PeerId {
			keySlice = append(keySlice, key)
		}
	}

	if len(keySlice) == 0 {
		// è l'unico nodo all'interno della rete: il suo successore e predecessore coincidono con lui stesso
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
		if key > peerPtr.PeerId && key < succKey {
			succKey = key
		}
		if key < peerPtr.PeerId && key > predKey {
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
