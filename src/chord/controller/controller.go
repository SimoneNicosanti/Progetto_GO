package controller

import (
	"chord/peer"
	"chord/view"
)

var nodePtr *peer.ChordPeer

func Controller(peerPtr *peer.ChordPeer) {
	nodePtr = peerPtr
	var exit bool = false
	for !exit {
		var userOption int = view.GetUserOption()
		switch userOption {
		case 1:
			//Search
			lookupResource()
		case 2:
			//Add
			addResource()
		case 3:
			//Remove
			removeResource()
		case 4:
			//Exit
			err := peer.LeaveRing(nodePtr)
			if err != nil {
				view.PrintError("Impossibile Uscire dall' anello; Attendere per non perdere le risorse salvate", err)
			} else {
				exit = true
			}
		default:
			view.PrintInvalidOption()
		}
	}
}

func lookupResource() {
	key, err := view.GetInfoToSearch()
	if err != nil {
		return
	}
	resourcePtr := new(peer.Resource)
	err = nodePtr.LookupResource(&key, resourcePtr)
	if err != nil {
		view.PrintError("Impossibile trovare risorsa", err)
		return
	}
	view.PrintResult(resourcePtr)
}

func addResource() {
	resourceValue, err := view.GetInfoToAdd()
	if err != nil {
		return
	}

	resourcePtr := new(peer.Resource)
	resourcePtr.Value = resourceValue
	var newResourceKey int
	err = nodePtr.AddResource(resourcePtr, &newResourceKey)
	if err != nil {
		view.PrintError("Impossibile aggiungere risorsa", err)
		return
	}
	view.PrintNewResourceKey(newResourceKey)
}

func removeResource() {
	key, err := view.GetInfoToRemove()
	if err != nil {
		return
	}

	resourcePtr := new(peer.Resource)
	err = nodePtr.RemoveResource(&key, resourcePtr)

	if err != nil {
		view.PrintError("Impossibile rimuovere risorsa", err)
		return
	}
	view.PrintResult(resourcePtr)
}
