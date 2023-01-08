package peer

import (
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/rpc"
	"time"
)

type SuccAndPred struct {
	SuccPeerPtr *ChordPeer
	PredPeerPtr *ChordPeer
}

var registryPort string = "1234"
var registryIP string = "127.0.0.1"

func InitializeChord() (*ChordPeer, error) {

	//peerId := int(time.Now().Unix() % int64(N))
	rand.Seed(time.Now().UnixNano())
	peerId := rand.Intn(N)

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

	// Messo prima perché ho il Ping fatto dal Registry
	go serveChord(listener)

	err = enterInRing(peer, succAndPredPtr.SuccPeerPtr, succAndPredPtr.PredPeerPtr)
	if err != nil {
		return nil, err
	}

	go stabilizeRing(peer)
	go manageCache()

	log.Default().Println("Inizializzazione avvenuta con succsso")

	return peer, nil
}

func stabilizeRing(peerPtr *ChordPeer) {
	/*
		Periodicamente chiedo al registry chi sono successore e predecessore.
		Questo mi permette di stabilizzare l'anello anche in caso di accessi concorrenti.
		Infatti ogni nodo periodicamente stabilizza la sua tabella e quindi ognuno ha almeno
		il suo successore e predecessore corretti.
		Nel caso di fingerTable standard, la ricerca del nodo successore potrebbe essere fatta
		chiedendo solo al successore in modo ricorsivo, così da avere garanzia di correttezza
	*/
	for {
		timer := time.NewTimer(STABILIZE_TIMER * time.Second)
		<-timer.C
		fingerTable.mutex.Lock()
		resourceMap.mutex.Lock()

		//succConn := fingerTable.table[1]
		//predConn := fingerTable.table[0]

		/*if succConn == nil || predConn == nil {
			resourceMap.mutex.Unlock()
			fingerTable.mutex.Unlock()
			continue
		}

		//arg := 0
		//err_1 := succConn.ClientPtr.Call("ChordPeer.Ping", &arg, nil)
		//err_2 := predConn.ClientPtr.Call("ChordPeer.Ping", &arg, nil)

		//if err_1 != nil || err_2 != nil {
		//log.Default().Println("Anello Non Stabile")*/
		succAndPredPtr := new(SuccAndPred)
		err := registryClientPtr.Call("Registry.GetSuccessorAndPredecessor", peerPtr, succAndPredPtr)
		if err != nil {
			log.Default().Println("Impossibile Stabilizzare l'anello")
			resourceMap.mutex.Unlock()
			fingerTable.mutex.Unlock()
			//os.Exit(-1)
			continue
		}
		err = initializeFingerTable(succAndPredPtr.SuccPeerPtr, succAndPredPtr.PredPeerPtr, peerPtr)
		if err != nil {
			log.Default().Println("Impossibile Inizializzare Finger Table")
			resourceMap.mutex.Unlock()
			fingerTable.mutex.Unlock()
			continue
		}
		log.Default().Println("Anello Stabilizzato")
		//}
		resourceMap.mutex.Unlock()
		fingerTable.mutex.Unlock()
	}
}

func serveChord(listenerPtr *net.Listener) {
	for {
		http.Serve(*listenerPtr, nil)
	}
}

func registerService(peerPtr *ChordPeer) (*net.Listener, error) {
	err := rpc.Register(peerPtr)
	if err != nil {
		log.Default().Println("Impossibile registrare il servizio")
		log.Default().Printf("Errore: '%s'", err.Error())
		return nil, err
	}

	rpc.HandleHTTP()
	chordListener, err := net.Listen("tcp", ":0")
	if err != nil {
		log.Default().Println("Impossibile eseguire listen")
		log.Default().Printf("Errore: '%s'", err.Error())
		return nil, err
	}

	return &chordListener, nil
}

func registerPeer(peerPtr *ChordPeer) (*SuccAndPred, error) {

	client, err := rpc.DialHTTP("tcp", registryIP+":"+registryPort)
	if err != nil {
		log.Default().Println("Impossibile collegarsi al registry")
		log.Default().Printf("Errore: '%s'\n", err.Error())
		return nil, err
	}
	registryClientPtr = client

	succAndPredPtr := new(SuccAndPred)
	err = client.Call("Registry.PeerJoin", peerPtr, succAndPredPtr)
	if err != nil {
		log.Default().Println("Impossibile registrare il peer")
		log.Default().Printf("Errore: '%s'\n", err.Error())
		return nil, err
	}

	return succAndPredPtr, nil
}

func enterInRing(peer *ChordPeer, succPeerPtr *ChordPeer, predPeerPtr *ChordPeer) error {
	fingerTable.mutex.Lock()
	resourceMap.mutex.Lock()
	defer fingerTable.mutex.Unlock()
	defer resourceMap.mutex.Unlock()

	err := initializeFingerTable(succPeerPtr, predPeerPtr, peer)
	if err != nil {
		log.Default().Println("Impossibile inizializzare la finger table")
		return err
	}

	if succPeerPtr != nil && predPeerPtr != nil {
		// Contatto successore e predecessore per modificare i puntatori
		// Il successore mi risponde anche con le risorse da prendere in carico
		// Il predecessore mi risponde con null

		predConn := fingerTable.table[0].ClientPtr
		err = predConn.Call("ChordPeer.ChangeSuccessor", peer, nil)
		if err != nil {
			log.Default().Println("Impossibile cambiare il successore del predecessore")
			return err
		}

		var tempMap map[int](*Resource) = make(map[int]*Resource)
		succConn := fingerTable.table[1].ClientPtr
		err = succConn.Call("ChordPeer.ChangePredecessorByJoin", peer, &tempMap)
		if err != nil {
			log.Default().Println("Impossibile cambiare il predecessore del successore")
			return err
		}
		for key, value := range tempMap {
			resourceMap.resMap[key] = value
			log.Default().Printf("Ricevuto. Key: %d, Value: %s\n", key, value.Value)
		}

	}

	return nil
}

func initializeFingerTable(succPeerPtr *ChordPeer, predPeerPtr *ChordPeer, peerPtr *ChordPeer) error {
	// Lock da prendere prima della chiamata

	if succPeerPtr == nil && predPeerPtr == nil {
		fingerTable.table[0] = nil
		fingerTable.table[1] = nil
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
		log.Default().Println("Impossibile collegarsi al successore")
		return err
	}
	fingerTable.table[1] = &PeerConnection{*succPeerPtr, succConnPtr}

	predConnPtr, err := connectToPeer(predPeerPtr)
	if err != nil {
		log.Default().Println("Impossibile collegarsi al predecessore")
		return err
	}
	fingerTable.table[0] = &PeerConnection{*predPeerPtr, predConnPtr}

	return nil
}

func connectToPeer(peerPtr *ChordPeer) (*rpc.Client, error) {
	clientPtr, err := rpc.DialHTTP("tcp", peerPtr.PeerAddr)
	if err != nil {
		log.Default().Println("Impossibile collegarsi al peer")
		return nil, err
	}

	return clientPtr, nil
}
