package view

import (
	"chord/peer"
	"fmt"
	"strconv"
)

func GetUserOption() int {
	printMenu()
	userOptionString, err := GetUserInput("")
	if err != nil {
		return -1
	}
	userOption, err := strconv.Atoi(userOptionString)
	if err != nil {
		fmt.Println("Errore Conversione Input")
	}
	return userOption
}

func PrintInvalidOption() {
	fmt.Println("Opzione Non Valida")
}

func printMenu() {
	fmt.Print("\n")
	fmt.Println("Scegli Opzione")
	fmt.Println("1. Ricerca Risorsa")
	fmt.Println("2. Aggiungi Risorsa")
	fmt.Println("3. Rimuovi Risora")
	fmt.Print("4. Esci")
}

func GetInfoToAdd() (string, error) {
	fmt.Print("\n")

	userInput, err := GetUserInput("Inserire stringa da aggiungere")
	if err != nil {
		return "", err
	}
	return userInput, nil
}

func GetInfoToRemove() (int, error) {
	fmt.Print("\n")
	return GetUserKey("Inserire chiave da rimuovere")
}

func GetInfoToSearch() (int, error) {
	fmt.Print("\n")
	return GetUserKey("Inserire chiave da cercare")
}

func PrintNewResourceKey(newResourceKey int) {
	fmt.Printf("La chiave della risorsa è %d\n", newResourceKey)
}

func PrintError(textString string, err error) {
	fmt.Println(textString)
	fmt.Printf("Errore: '%s'\n", err.Error())
}

func PrintResult(resourcePtr *peer.Resource) {
	fmt.Printf("Valore: '%s'\n", resourcePtr.Value)
}
