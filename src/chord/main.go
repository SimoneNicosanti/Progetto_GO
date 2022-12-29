package main

import (
	"chord/controller"
	"chord/peer"
	"fmt"
)

func main() {
	fmt.Println("------------------ Chord ------------------")
	peer, err := peer.InitializeChord()
	if err != nil {
		fmt.Println("Errore Inizializzazione Chord")
		fmt.Printf("Errore '%s'\n", err.Error())
		return
	}
	fmt.Printf("PeerId: %d\n", peer.PeerId)
	controller.Controller(peer)
	fmt.Println("------------------- Bye -------------------")
}
