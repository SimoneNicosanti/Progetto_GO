package peer

import (
	"fmt"
	"net"
	"net/http"
	"net/rpc"
	"time"
)

type ResourceRange struct {
	FirstKey int //esclusa
	LastKey  int //inclusa
}

type SuccAndPred struct {
	SuccPeerPtr *ChordPeer
	PredPeerPtr *ChordPeer
}

var registryPort string = "1234"
var registryIP string = ""

func InitializeChord() (*ChordPeer, error) {

	peerId := int(time.Now().Unix() % int64(N))

	var peer *ChordPeer = new(ChordPeer)
	peer.PeerId = peerId
	listener, err := registerService(peer)
	if err != nil {
		return nil, err
	}

	peer.PeerAddr = (*listener).Addr().String()
	succAndPredPtr, err := registerPeer(peer)
	if err != nil {
		return nil, err
	}

	go serveChord(listener)

	err = enterInRing(peer, succAndPredPtr.SuccPeerPtr, succAndPredPtr.PredPeerPtr)
	if err != nil {
		return nil, err
	}

	println("Inizializzazione avvenuta con succsso")

	return peer, nil
}

func serveChord(listenerPtr *net.Listener) {
	for {
		http.Serve(*listenerPtr, nil)
	}
}

func registerService(peerPtr *ChordPeer) (*net.Listener, error) {
	err := rpc.Register(peerPtr)
	if err != nil {
		fmt.Println("Impossibile registrare il servizio")
		fmt.Printf("Errore: '%s'", err.Error())
		return nil, err
	}

	rpc.HandleHTTP()
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		fmt.Println("Impossibile eseguire listen")
		fmt.Printf("Errore: '%s'", err.Error())
		return nil, err
	}

	return &listener, nil
}

func registerPeer(peerPtr *ChordPeer) (*SuccAndPred, error) {
	client, err := rpc.DialHTTP("tcp", registryIP+":"+registryPort)
	if err != nil {
		fmt.Println("Impossibile collegarsi al registry")
		fmt.Printf("Errore: '%s'\n", err.Error())
		return nil, err
	}
	registryClientPtr = client

	succAndPredPtr := new(SuccAndPred)
	err = client.Call("Registry.PeerJoin", peerPtr, succAndPredPtr)
	if err != nil {
		fmt.Println("Impossibile registrare il peer")
		fmt.Printf("Errore: '%s'\n", err.Error())
		return nil, err
	}

	return succAndPredPtr, nil
}

func enterInRing(peer *ChordPeer, succPeerPtr *ChordPeer, predPeerPtr *ChordPeer) error {

	err := initializeFingerTable(succPeerPtr, predPeerPtr)
	if err != nil {
		fmt.Println("Impossibile inizializzare la finger table")
		return err
	}

	if succPeerPtr != nil && predPeerPtr != nil {
		// Contatto successore e predecessore per modificare i puntatori
		// Il successore mi risponde anche con le risorse da prendere in carico
		// Il predecessore mi risponde con null
		fingerTable.mutex.Lock()
		resourceMap.mutex.Lock()
		defer resourceMap.mutex.Unlock()
		defer fingerTable.mutex.Unlock()

		predConn := fingerTable.table[0].ClientPtr
		err = predConn.Call("ChordPeer.ChangeSuccessor", peer, nil)
		if err != nil {
			fmt.Println("Impossibile cambiare il successore del predecessore")
			return err
		}

		var tempMap map[int](*Resource) = make(map[int]*Resource)
		succConn := fingerTable.table[1].ClientPtr
		err = succConn.Call("ChordPeer.ChangePredecessor", peer, &tempMap)
		if err != nil {
			fmt.Println("Impossibile cambiare il predecessore del successore")
			return err
		}
		for key, value := range tempMap {
			resourceMap.resMap[key] = value
		}

	}

	return nil
}

func initializeFingerTable(succPeerPtr *ChordPeer, predPeerPtr *ChordPeer) error {
	fingerTable.mutex.Lock()
	defer fingerTable.mutex.Unlock()

	if succPeerPtr == nil && predPeerPtr == nil {
		return nil
	}

	// Chiudo tutte le connessioni precedenti
	for i := range fingerTable.table {
		entry := fingerTable.table[i]
		if entry != nil {
			entry.ClientPtr.Close()
		}
	}

	// Apro le connessioni ai nuovi peer
	// In Chord di base dovrei ricalcolare tutte le entry contattando altri peer
	succConnPtr, err := connectToPeer(succPeerPtr)
	if err != nil {
		fmt.Println("Impossibile collegarsi al successore")
		return err
	}
	fingerTable.table[1] = &PeerConnection{*succPeerPtr, succConnPtr}

	predConnPtr, err := connectToPeer(predPeerPtr)
	if err != nil {
		fmt.Println("Impossibile collegarsi al predecessore")
		return err
	}
	fingerTable.table[0] = &PeerConnection{*predPeerPtr, predConnPtr}

	return nil
}

func connectToPeer(peerPtr *ChordPeer) (*rpc.Client, error) {

	println("Ciao 1")
	clientPtr, err := rpc.DialHTTP("tcp", peerPtr.PeerAddr)
	if err != nil {
		fmt.Println("Impossibile collegarsi al peer")
		return nil, err
	}
	println("Ciao 1")

	return clientPtr, nil
}
