package main

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"log"
	"os"
	"os/signal"
	"scrp/client/handlers"
	"syscall"

	"golang.org/x/term"
)

func main() {
	// Define own username, password and public key here
	sigChan := make(chan os.Signal, 1)
	// Notify the program to send the SIGINT signal to sigChan
	signal.Notify(sigChan, syscall.SIGINT)
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Print("Enter Server Address or Host (e.g. 192.168.1.1 or localhost): ")
	scanner.Scan()
	host := scanner.Text()

	fmt.Print("Enter username: ")
	scanner.Scan()
	username := scanner.Text()

	fmt.Print("Enter Password: ")
	bytePassword, _ := term.ReadPassword(int(syscall.Stdin))
	password := string(bytePassword)

	client, err := handlers.NewClient(username, password)
	if err != nil {
		log.Fatalf("Failed to authenticate to server: %v", err)
	}

	// Connect to server over TLS
	conf := &tls.Config{
		InsecureSkipVerify: true, // Should be set to false in production
	}

	conn, err := tls.Dial("tcp", host+":8080", conf)
	if err != nil {
		log.Fatalf("Failed to connect to server: %v", err)
	}
	client.Conn = conn

	go func(client *handlers.Client) {
		// Wait for the signal
		<-sigChan

		// Handle the signal (Ctrl+C)
		fmt.Println("^C captured, exiting...")
		client.SendDisconnectRequest()
		os.Exit(0)
	}(client)

	// Handle response from server
	client.HandleServerMessages()

	// Send authentication request to the server
	client.SendAuthRequest()

	// Allow user to select recipient and send messages
	client.StartMessagingUI()
}
