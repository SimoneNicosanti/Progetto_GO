package main

import (
	"chord/controller"
	"chord/peer"
	"fmt"
	"io/ioutil"
	"log"
)

func main() {
	fmt.Println("------------------ Chord ------------------")
	//log.SetOutput(ioutil.Discard)
	peer, err := peer.InitializeChord()
	log.SetOutput(ioutil.Discard)
	if err != nil {
		fmt.Println("Errore Inizializzazione Chord")
		fmt.Printf("Errore '%s'\n", err.Error())
		return
	}
	fmt.Printf("PeerId: %d\n", peer.PeerId)
	controller.Controller(peer)
	fmt.Println("------------------- Bye -------------------")

}
