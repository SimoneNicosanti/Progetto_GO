package peer

import (
	"errors"
)

/*
	TODO
	- AGGIUNTA SEMAFORI SU STRUTTURE CONDIVISE
	- Thread per verifica TTL
	- Cache di risorse recenti : terna chiave, valore, TTL
	- Finger Table
*/

func (t *ChordPeer) Lookup(keyPtr *int, replyPtr *Resource) error {
	/*
		TODO
		1. Consulta Cache
		2. Verifica tu responsabile
		3. Verifica successore responsabile
		4. Ricerca ricorsiva
		5. Aggiunta in cache
	*/
	resourceMap.mutex.Lock()
	defer resourceMap.mutex.Unlock()

	var resourceKey int = *keyPtr
	if resourceMap.resMap[resourceKey] == nil {
		return errors.New("la chiave non ha un valore associato")
	}

	replyPtr.Value = resourceMap.resMap[resourceKey].Value
	return nil
}

func (t *ChordPeer) Remove(keyPtr *int, replyPtr *Resource) error {
	// TODO Aggiungere ricerca del nodo responsabile
	resourceMap.mutex.Lock()
	defer resourceMap.mutex.Unlock()

	var resourceKey int = *keyPtr
	if resourceMap.resMap[resourceKey] == nil {
		return errors.New("la chiave non ha un valore associato")
	}

	replyPtr.Value = resourceMap.resMap[resourceKey].Value
	resourceMap.resMap[resourceKey] = nil

	return nil
}

func (t *ChordPeer) Add(resourcePtr *Resource, replyPtr *int) error {
	// TODO Aggiungere ricerca del nodo responsabile
	resourceMap.mutex.Lock()
	defer resourceMap.mutex.Unlock()

	var resourceKey int = simpleHash(*resourcePtr)

	if resourceMap.resMap[resourceKey] != nil {
		return errors.New("la chiave ha già un valore associato")
	}
	resourceMap.resMap[resourceKey] = resourcePtr

	*replyPtr = resourceKey

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

func (t *ChordPeer) ChangeSuccessor(newSuccPtr *ChordPeer, replyPtr *int) error {
	fingerTable.mutex.Lock()
	defer fingerTable.mutex.Unlock()

	table := fingerTable.table

	newSuccConn, err := connectToPeer(newSuccPtr)
	if err != nil {
		return err
	}

	if table[1] != nil {
		/*if newSuccPtr.PeerId > table[1].Peer.PeerId {
			return errors.New("il nuovo successore non può essere maggiore dell'attuale")
		}*/
		table[1].ClientPtr.Close()
	}

	table[1] = &PeerConnection{*newSuccPtr, newSuccConn}

	return nil

}

func (t *ChordPeer) ChangePredecessor(newPredPtr *ChordPeer, replyPtr *map[int](*Resource)) error {
	fingerTable.mutex.Lock()
	resourceMap.mutex.Lock()
	defer resourceMap.mutex.Unlock()
	defer fingerTable.mutex.Unlock()

	table := fingerTable.table

	newPredConn, err := connectToPeer(newPredPtr)
	if err != nil {
		return err
	}

	if table[0] != nil {
		/*if newPredPtr.PeerId < table[0].Peer.PeerId {
			return errors.New("il nuovo predecessore non può essere minore dell'attuale")
		}*/
		table[0].ClientPtr.Close()
	}

	for key, value := range resourceMap.resMap {
		if key > table[0].Peer.PeerId && key <= newPredPtr.PeerId {
			(*replyPtr)[key] = value
			delete(resourceMap.resMap, key)
		}
	}

	table[0] = &PeerConnection{*newPredPtr, newPredConn}

	return nil

}
