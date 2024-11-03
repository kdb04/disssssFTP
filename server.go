package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

// In-memory store for authenticated sessions
var authenticatedSessions = make(map[string]net.Conn)
var mu sync.Mutex

// Timeout duration for idle connections
const idleTimeout = 5 * time.Minute

// Function to read the credentials from the file
func readCredentials(filePath string) (map[string]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	credentials := make(map[string]string)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, ":")
		if len(parts) == 2 {
			credentials[parts[0]] = parts[1]
		}
	}
	return credentials, scanner.Err()
}

func handleConnection(conn net.Conn, credentials map[string]string) {
	defer conn.Close()

	// Authentication process
	authenticated := authenticate(conn, credentials)
	if !authenticated {
		conn.Write([]byte("Incorrect username or password. Disconnecting.\n"))
		return
	}

	// After authentication, persist session
	clientAddr := conn.RemoteAddr().String()
	mu.Lock()
	authenticatedSessions[clientAddr] = conn
	mu.Unlock()

	defer func() {
		mu.Lock()
		delete(authenticatedSessions, clientAddr)
		mu.Unlock()
	}()

	conn.Write([]byte("Authentication successful. You are now connected.\n"))

	// Reading the data from the client within the idle timeout limit
	buf := make([]byte, 1024)
	conn.SetReadDeadline(time.Now().Add(idleTimeout))

	for {
		n, err := conn.Read(buf)
		if err != nil {
			if strings.Contains(err.Error(), "connection reset by peer") {
				fmt.Println("Connection closed by client.")
			} else if strings.Contains(err.Error(), "i/o timeout") {
				fmt.Println("Connection timed out.")
			} else {
				fmt.Println("Disconnected due to:", err)
			}
			return
		}

		// Resetting the deadline after every successful read to handle long-duration sessions
		conn.SetReadDeadline(time.Now().Add(idleTimeout))

		fmt.Printf("Received from %s: %s\n", clientAddr, buf[:n])
		conn.Write([]byte("Data received.\n"))
	}
}

// Authentication function to validate the client credentials
func authenticate(conn net.Conn, credentials map[string]string) bool {
	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		fmt.Println("Error reading credentials:", err)
		return false
	}

	input := strings.TrimSpace(string(buf[:n]))
	parts := strings.Split(input, ":")
	if len(parts) != 2 {
		conn.Write([]byte("Invalid format. Use username:password format.\n"))
		return false
	}

	username, password := parts[0], parts[1]
	if storedPassword, ok := credentials[username]; ok && storedPassword == password {
		return true
	}
	return false
}

func main() {
	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, os.Interrupt, syscall.SIGTERM)

	listener, err := net.Listen("tcp", ":9090")
	if err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
	defer listener.Close()
	log.Println("TCP server is listening on port 9090...")

	go func() {
		for {
			connection, err := listener.Accept()
			if err != nil {
				log.Println("Error accepting connection: %v", err)
				continue
			}
			go handleConnection(connection)
		}
	}()

	<-signalChannel
	log.Println("Interrupt signal received. Shutting down gracefully...")

	mu.Lock()
	for _, conn := range authenticatedSessions {
		conn.Close()
	}
	mu.Unlock()
	log.Println("Server shutdown completed")

}
