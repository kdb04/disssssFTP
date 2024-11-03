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
	// Set up signal handling
	sigChannel := make(chan os.Signal, 1)
	signal.Notify(sigChannel, os.Interrupt, syscall.SIGTERM)

	// Connect to server
	conn, err := net.Dial("tcp", "localhost:9090")
	if err != nil {
		log.Fatalf("Failed to connect to server: %v", err)
	}

	// Create a done channel for cleanup coordination
	done := make(chan struct{})

	// Handle signals in a separate goroutine
	go func() {
		<-sigChannel
		log.Println("\nCtrl+C detected, closing connection to server...")
		close(done)
		conn.Close()
	}()

	// Create a channel for authentication result
	authDone := make(chan bool)

	go func() {
		defer close(authDone)

		// Asking for username
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Enter username: ")
		username, _ := reader.ReadString('\n')
		username = strings.TrimSpace(username)

		// Asking for password
		fmt.Print("Enter password: ")
		bytePassword, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			fmt.Println("\nError reading password:", err)
			authDone <- false
			return
		}
		password := strings.TrimSpace(string(bytePassword))
		fmt.Println()

		// Send credentials
		credentials := fmt.Sprintf("%s:%s", username, password)
		_, err = conn.Write([]byte(credentials + "\n"))
		if err != nil {
			fmt.Println("Error sending credentials:", err)
			authDone <- false
			return
		}

		// Read server response
		serverResponse, err := bufio.NewReader(conn).ReadString('\n')
		if err != nil {
			fmt.Println("Error reading server response:", err)
			authDone <- false
			return
		}

		if serverResponse != "Authentication successful. You are now connected.\n" {
			fmt.Printf("Authentication failed. Server response: %s\n", serverResponse)
			authDone <- false
			return
		}

		fmt.Println("Authentication successful.")
		authDone <- true
	}()

	// Wait for authentication or interruption
	select {
	case <-done:
		return
	case success := <-authDone:
		if !success {
			return
		}
	}

	// Simulate wait time before sending the message
	select {
	case <-done:
		return
	case <-time.After(5 * time.Second):
	}

	// Send message
	message := "Hello from single client!"
	_, err = conn.Write([]byte(message + "\n"))
	if err != nil {
		fmt.Println("Error sending message:", err)
		return
	}

	fmt.Printf("Message sent: %s\n", message)

	// Wait for server response or interruption
	responseReader := bufio.NewReader(conn)
	go func() {
		response, err := responseReader.ReadString('\n')
		if err != nil {
			if !strings.Contains(err.Error(), "use of closed network connection") {
				fmt.Printf("Error reading server response: %v\n", err)
			}
			return
		}
		fmt.Printf("Server response: %s", response)
	}()

	// Wait for done signal
	<-done
	time.Sleep(100 * time.Millisecond) // Give a short time for cleanup
}
