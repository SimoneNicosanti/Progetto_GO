package view

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
)

func GetUserInput(hintString string) (string, error) {
	fmt.Println(hintString)
	fmt.Print(">>> ")

	var userInput string
	var scanner *bufio.Scanner = bufio.NewScanner(os.Stdin)
	scanner.Scan()
	if scanner.Err() != nil {
		println("Errore Lettura Input")
		return "", errors.New("errore lettura input")
	}
	userInput = scanner.Text()

	return userInput, nil
}

func GetUserKey(hintString string) (int, error) {
	userInput, err := GetUserInput(hintString)
	if err != nil {
		return -1, err
	}

	userInputKey, err := strconv.Atoi(userInput)
	if err != nil {
		fmt.Println("Errore Conversione Input")
		return -1, err
	}

	return userInputKey, nil
}
