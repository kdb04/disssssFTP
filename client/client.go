package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"golang.org/x/term"
)

func main() {

	sigchannel := make(chan os.Signal, 1)
	signal.Notify(sigchannel, os.Interrupt, syscall.SIGTERM)

	conn, err := net.Dial("tcp", "localhost:9090")
	if err != nil {
		log.Fatalf("Failed to connect to server: %v", err)
	}
	defer conn.Close()

	go func() {
		<-sigchannel
		log.Println("Ctrl+C detected, closing connection to server.")
		conn.Close()
		os.Exit(0)
	}()

	log.Println("Client is running. Press Ctrl+C to shut down the server.")

	// Asking for username
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter username: ")
	username, _ := reader.ReadString('\n')
	username = strings.TrimSpace(username)

	// Asking for password(made hidden due to golang.org/x/term)
	fmt.Print("Enter password: ")
	bytePassword, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		fmt.Println("\nError reading password:", err)
		return
	}
	password := strings.TrimSpace(string(bytePassword))
	fmt.Println()

	// Combining the username and password to form a single string
	credentials := fmt.Sprintf("%s:%s", username, password)

	// Sending the credentials string to the server for authentication
	_, err = conn.Write([]byte(credentials + "\n"))
	if err != nil {
		fmt.Println("Error sending credentials:", err)
		return
	}

	// Reading the response of the server after authentication
	serverResponse, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		fmt.Println("Error reading server response:", err)
		return
	}

	if serverResponse != "Authentication successful. You are now connected.\n" {
		fmt.Printf("Authentication failed. Server response: %s\n", serverResponse)
		return
	}

	fmt.Println("Authentication successful.")

	// Simulating wait time before sending the message
	time.Sleep(5 * time.Second)

	// Sending a message to the sever after authentication
	message := "Hello from single client!"
	_, err = conn.Write([]byte(message + "\n"))
	if err != nil {
		fmt.Println("Error sending message:", err)
		return
	}

	fmt.Printf("Message sent: %s\n", message)
}
