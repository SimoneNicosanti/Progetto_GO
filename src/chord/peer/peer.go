package peer

import (
	"errors"
	"log"
	"time"
)

/*
	TODO
	- Finger Table normale di Chord??
	-
*/

func (t *ChordPeer) Lookup(keyPtr *int, replyPtr *Resource) error {

	resourceMap.mutex.Lock()
	fingerTable.mutex.Lock()
	peerCache.mutex.Lock()
	defer peerCache.mutex.Unlock()
	defer fingerTable.mutex.Unlock()
	defer resourceMap.mutex.Unlock()

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
		err := succConn.Call("ChordPeer.Lookup", keyPtr, replyPtr)
		if err != nil {
			return err
		}
		peerCache.cacheMap[*keyPtr] = CacheTuple{replyPtr, time.Now()}
	}

	return nil
}

func (t *ChordPeer) Remove(keyPtr *int, replyPtr *Resource) error {
	// Dovrei pulire la cache in teoria, ma la pulizia viene fatta in automatico
	resourceMap.mutex.Lock()
	fingerTable.mutex.Lock()
	defer fingerTable.mutex.Unlock()
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
		err := succConn.Call("ChordPeer.Remove", keyPtr, replyPtr)

		if err != nil {
			return err
		}
	}

	return nil
}

func (t *ChordPeer) Add(resourcePtr *Resource, replyPtr *int) error {
	resourceMap.mutex.Lock()
	fingerTable.mutex.Lock()
	peerCache.mutex.Lock()
	defer peerCache.mutex.Unlock()
	defer fingerTable.mutex.Unlock()
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
		err := succConn.Call("ChordPeer.Add", resourcePtr, replyPtr)
		if err != nil {
			return err
		}
		peerCache.cacheMap[addKey] = CacheTuple{resourcePtr, time.Now()}
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
		// Unico nodo della rete
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
		timer := time.NewTimer(30 * time.Second)
		<-timer.C
		peerCache.mutex.Lock()

		for key, tuple := range peerCache.cacheMap {
			now := time.Now()
			if now.Sub(tuple.entryTime) > (10 * time.Second) {
				delete(peerCache.cacheMap, key)
			}
		}

		peerCache.mutex.Unlock()
	}
}

func (t *ChordPeer) ChangeSuccessor(newSuccPtr *ChordPeer, replyPtr *int) error {
	fingerTable.mutex.Lock()
	defer fingerTable.mutex.Unlock()

	newSuccConn, err := connectToPeer(newSuccPtr)
	if err != nil {
		return err
	}

	if fingerTable.table[1] != nil {
		fingerTable.table[1].ClientPtr.Close()
	}

	fingerTable.table[1] = &PeerConnection{*newSuccPtr, newSuccConn}

	log.Default().Printf("New Successor: %d\n", newSuccPtr.PeerId)

	return nil

}

func (t *ChordPeer) ChangePredecessor(newPredPtr *ChordPeer, replyPtr *map[int](*Resource)) error {
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
				log.Default().Printf("Passato 1: %d, %s", key, *value)
			}
		} else if newPredPtr.PeerId < t.PeerId && newPredPtr.PeerId > currPredId {
			if key <= newPredPtr.PeerId && key > currPredId {
				(*replyPtr)[key] = value
				delete(resourceMap.resMap, key)
				log.Default().Printf("Passato 2: %d, %s", key, *value)
			}
		} else {
			if (key > currPredId && key < N) || (key >= 0 && key <= newPredPtr.PeerId) {
				(*replyPtr)[key] = value
				delete(resourceMap.resMap, key)
				log.Default().Printf("Passato 3: %d, %s", key, *value)
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

func (t *ChordPeer) Ping(argsPtr *int, replyPtr *int) error {
	return nil
}
