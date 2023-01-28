package peer

import (
	"errors"
	"log"
	"time"
)

/*
	TODO
	- Studiare bene i lock delle risorse: vedere se si può parallelizzare meglio senza usare le defer
		se uso le defer, quando ho la ricerca ricorsiva, le risorse possono essere bloccate fino alla fine della ricorsione!!
*/

func (t *ChordPeer) LookupResource(keyPtr *int, replyPtr *Resource) error {

	resourceMap.mutex.RLock()
	fingerTable.mutex.RLock()
	peerCache.mutex.Lock()
	defer peerCache.mutex.Unlock()
	defer fingerTable.mutex.RUnlock()
	defer resourceMap.mutex.RUnlock()

	lookupKey := *keyPtr
	if isResponsible(t, lookupKey) {
		// Il nodo attuale è il responsabile della risorsa
		if resourceMap.resMap[lookupKey] == nil {
			return errors.New("la chiave non ha un valore associato")
		}
		replyPtr.Value = resourceMap.resMap[lookupKey].Value
		log.Default().Printf("Richiesta Lookup key: %d\n", *keyPtr)
	} else {
		// Ricerca in cache
		cacheValue, inCache := peerCache.cacheMap[*keyPtr]
		if inCache {
			*replyPtr = *cacheValue.resource
			return nil
		}

		// Ricerca in rete ed inserimento in cache
		succConn := fingerTable.table[1].ClientPtr
		err := succConn.Call("ChordPeer.LookupResource", keyPtr, replyPtr)
		if err != nil {
			return err
		}
		peerCache.cacheMap[*keyPtr] = CacheTuple{replyPtr, time.Now()}
	}

	return nil
}

func (t *ChordPeer) RemoveResource(keyPtr *int, replyPtr *Resource) error {
	// Dovrei pulire la cache in teoria, ma la pulizia viene fatta in automatico
	resourceMap.mutex.Lock()
	fingerTable.mutex.RLock()
	defer fingerTable.mutex.RUnlock()
	defer resourceMap.mutex.Unlock()

	removeKey := *keyPtr
	if isResponsible(t, removeKey) {
		if resourceMap.resMap[removeKey] == nil {
			return errors.New("la chiave non ha un valore associato")
		}
		replyPtr.Value = resourceMap.resMap[removeKey].Value
		resourceMap.resMap[removeKey] = nil
	} else {
		succConn := fingerTable.table[1].ClientPtr
		err := succConn.Call("ChordPeer.RemoveResource", keyPtr, replyPtr)

		if err != nil {
			return err
		}
	}

	return nil
}

func (t *ChordPeer) AddResource(resourcePtr *Resource, replyPtr *int) error {
	resourceMap.mutex.Lock()
	fingerTable.mutex.RLock()
	peerCache.mutex.Lock()
	defer peerCache.mutex.Unlock()
	defer fingerTable.mutex.RUnlock()
	defer resourceMap.mutex.Unlock()

	var addKey int = simpleHash(*resourcePtr)

	if isResponsible(t, addKey) {
		if resourceMap.resMap[addKey] != nil {
			return errors.New("la chiave ha già un valore associato")
		}
		resourceMap.resMap[addKey] = resourcePtr
		*replyPtr = addKey
		log.Default().Printf("Richiesta Aggiunta key: %d\n", addKey)
	} else {
		succConn := fingerTable.table[1].ClientPtr
		err := succConn.Call("ChordPeer.AddResource", resourcePtr, replyPtr)
		if err != nil {
			return err
		}
		peerCache.cacheMap[addKey] = CacheTuple{resourcePtr, time.Now()}
	}

	return nil
}

func LeaveRing(peerPtr *ChordPeer) error {
	fingerTable.mutex.Lock()
	resourceMap.mutex.Lock()
	defer resourceMap.mutex.Unlock()
	defer fingerTable.mutex.Unlock()

	err := registryClientPtr.Call("Registry.PeerLeave", peerPtr, nil)
	if err != nil {
		return err
	}

	if fingerTable.table[1] != nil {
		// Il nodo non è solo nell'anello --> devo passare le risorse

		succNode := fingerTable.table[1]
		predNode := fingerTable.table[0]

		err = predNode.ClientPtr.Call("ChordPeer.ChangeSuccessor", &(succNode.Peer), nil)
		if err != nil {
			log.Default().Println("Impossibile cambiare il successore del predecessore")
			return errors.New(err.Error() + "\nImpossibile cambiare il successore del predecessore")
		}

		predInfo := PredecessorInfo{predNode.Peer, resourceMap.resMap}

		err = succNode.ClientPtr.Call("ChordPeer.ChangePredecessorByLeave", &predInfo, nil)
		if err != nil {
			log.Default().Println("Impossibile cambiare il predecessore del successore")
			return errors.New(err.Error() + "\nImpossibile cambiare il predecessore del successore")
		}
	}

	err = registryClientPtr.Call("Registry.PeerLeave", peerPtr, nil)
	if err != nil {
		return err
	}

	return nil
}

func simpleHash(resource Resource) int {
	var hashValue int = 0
	for _, char := range resource.Value {
		hashValue += int(char - '0')
	}
	hashValue = hashValue % N
	return hashValue
}

func isResponsible(peer *ChordPeer, key int) bool {
	if fingerTable.table[0] == nil {
		// Unico nodo della rete --> Unico responsabile possibile
		return true
	}
	predId := fingerTable.table[0].Peer.PeerId
	peerId := peer.PeerId

	var isResp bool
	if predId > peerId {
		isResp = (key > predId && key < N) || (key >= 0 && key <= peerId)
	} else {
		isResp = (key > predId && key <= peerId)
	}

	return isResp
}

func manageCache() {
	for {
		timer := time.NewTimer(CACHE_TIMER * time.Second)
		<-timer.C
		peerCache.mutex.Lock()

		for key, tuple := range peerCache.cacheMap {
			now := time.Now()
			if now.Sub(tuple.entryTime) > (CACHE_RES_TTL * time.Second) {
				delete(peerCache.cacheMap, key)
			}
		}

		peerCache.mutex.Unlock()
	}
}

func (t *ChordPeer) ChangeSuccessor(newSuccPtr *ChordPeer, replyPtr *int) error {
	fingerTable.mutex.Lock()
	defer fingerTable.mutex.Unlock()

	var newTableValue *PeerConnection
	if newSuccPtr.PeerId == t.PeerId {
		// Nodo unico nella rete --> Successore diventa nil
		newTableValue = nil
	} else {
		newSuccConn, err := connectToPeer(newSuccPtr)
		if err != nil {
			return err
		}
		newTableValue = &PeerConnection{*newSuccPtr, newSuccConn}
	}

	if fingerTable.table[1] != nil {
		fingerTable.table[1].ClientPtr.Close()
	}

	fingerTable.table[1] = newTableValue

	log.Default().Printf("New Successor: %d\n", newSuccPtr.PeerId)

	return nil

}

func (t *ChordPeer) ChangePredecessorByJoin(newPredPtr *ChordPeer, replyPtr *map[int](*Resource)) error {
	fingerTable.mutex.Lock()
	resourceMap.mutex.Lock()
	defer resourceMap.mutex.Unlock()
	defer fingerTable.mutex.Unlock()

	newPredConn, err := connectToPeer(newPredPtr)
	if err != nil {
		return err
	}

	var currPredId int
	if fingerTable.table[0] == nil {
		currPredId = t.PeerId
	} else {
		currPredId = fingerTable.table[0].Peer.PeerId
		log.Default().Printf("Current Predecessor : %d", currPredId)
	}

	for key, value := range resourceMap.resMap {
		if newPredPtr.PeerId > t.PeerId {
			if key > currPredId && key <= newPredPtr.PeerId {
				(*replyPtr)[key] = value
				delete(resourceMap.resMap, key)
				//log.Default().Printf("Passato 1: %d, %s", key, *value)
			}
		} else if newPredPtr.PeerId < t.PeerId && newPredPtr.PeerId > currPredId {
			if key <= newPredPtr.PeerId && key > currPredId {
				(*replyPtr)[key] = value
				delete(resourceMap.resMap, key)
				//log.Default().Printf("Passato 2: %d, %s", key, *value)
			}
		} else {
			if (key > currPredId && key < N) || (key >= 0 && key <= newPredPtr.PeerId) {
				(*replyPtr)[key] = value
				delete(resourceMap.resMap, key)
				//log.Default().Printf("Passato 3: %d, %s", key, *value)
			}
		}
	}

	if fingerTable.table[0] != nil {
		fingerTable.table[0].ClientPtr.Close()
	}
	fingerTable.table[0] = &PeerConnection{*newPredPtr, newPredConn}

	log.Default().Printf("New Predecessor: %d\n", newPredPtr.PeerId)

	return nil

}

func (t *ChordPeer) ChangePredecessorByLeave(predInfoPtr *PredecessorInfo, replyPtr *int) error {
	fingerTable.mutex.Lock()
	resourceMap.mutex.Lock()
	defer resourceMap.mutex.Unlock()
	defer fingerTable.mutex.Unlock()

	var newTableValue *PeerConnection
	if predInfoPtr.Peer.PeerId == t.PeerId {
		// Il nodo diventa unico all'interno della rete --> Predecessore diventa nil
		newTableValue = nil
	} else {
		// Ho un altro nodo all'interno della rete che diventa il predecessore
		newPredConn, err := connectToPeer(&predInfoPtr.Peer)
		if err != nil {
			return err
		}
		newTableValue = &PeerConnection{predInfoPtr.Peer, newPredConn}
	}

	for key, value := range predInfoPtr.PredecessorMap {
		resourceMap.resMap[key] = value
		log.Default().Printf("Passata Coppia: (%d, %s)\n", key, value.Value)
	}

	if fingerTable.table[0] != nil {
		fingerTable.table[0].ClientPtr.Close()
	}

	fingerTable.table[0] = newTableValue

	log.Default().Printf("New Predecessor: %d\n", predInfoPtr.Peer.PeerId)

	return nil
}

func (t *ChordPeer) Ping(argsPtr *int, replyPtr *int) error {
	return nil
}

func (t *ChordPeer) FindSuccessor(peerPtr *ChordPeer, successorPtr *ChordPeer) error {
	fingerTable.mutex.RLock()
	defer fingerTable.mutex.RUnlock()

	err := findResourceSuccessor(peerPtr.PeerId, t, successorPtr)
	if err != nil {
		return err
	}

	return nil
}

func findResourceSuccessor(resKey int, peerPtr *ChordPeer, successorPtr *ChordPeer) error {
	if isResponsible(peerPtr, resKey) {
		successorPtr.PeerId = peerPtr.PeerId
		successorPtr.PeerAddr = peerPtr.PeerAddr
	} else {
		// Problema!! Come faccio a capire di aver percorso tutto l'anello?? Potrei rischiare il blocco dei nodi per presa multipla dei lock
		succConn := fingerTable.table[1].ClientPtr
		arg := 0
		err := succConn.Call("ChordPeer.FindSuccessor", &arg, successorPtr)
		if err != nil {
			return err
		}
	}
	return nil
}
