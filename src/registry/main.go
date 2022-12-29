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
	t.FindSuccessorAndPredecessor(&chordPeerPtr.PeerId, replyPtr)

	return nil
}

func (t *Registry) PeerLeave(chordPeerPtr *ChordPeer, replyPtr *int) error {
	myMap.mutex.Lock()
	defer myMap.mutex.Unlock()

	delete(myMap.peerMap, chordPeerPtr.PeerId)

	return nil
}

func (t *Registry) FindSuccessorAndPredecessor(peerIdPtr *int, succAndPredPtr *SuccAndPred) error {

	keySlice := make([]int, 0, 10)
	for key := range myMap.peerMap {
		if key != *peerIdPtr {
			keySlice = append(keySlice, key)
		}
	}

	if len(keySlice) == 0 {
		// è il primo nodo che entra nella rete
		succAndPredPtr.PredPeerPtr = nil
		succAndPredPtr.SuccPeerPtr = nil
		return nil
	}

	succKey := N
	predKey := -1
	minKey := N
	maxKey := -1

	for index := range keySlice {
		// Cerco successore
		key := keySlice[index]
		if key > *peerIdPtr && key < succKey {
			succKey = key
		}
		if key < *peerIdPtr && key > predKey {
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

	return nil
}
