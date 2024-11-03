package main

import (
	"bufio"
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
var listener net.Listener

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

func handleConnection(conn net.Conn, credentials map[string]string, wg *sync.WaitGroup) {
	defer wg.Done()
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
				log.Printf("Connection closed by client: %s\n", clientAddr)
			} else if strings.Contains(err.Error(), "i/o timeout") {
				log.Printf("Connection timed out: %s\n", clientAddr)
			} else {
				log.Printf("Disconnected due to: %v from %s\n", err, clientAddr)
			}
			return
		}

		// Resetting the deadline after every successful read
		conn.SetReadDeadline(time.Now().Add(idleTimeout))

		log.Printf("Received from %s: %s\n", clientAddr, strings.TrimSpace(string(buf[:n])))
		conn.Write([]byte("Data received.\n"))
	}
}

// Authentication function to validate the client credentials
func authenticate(conn net.Conn, credentials map[string]string) bool {
	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		log.Println("Error reading credentials:", err)
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
	// Create a WaitGroup to track active connections
	var wg sync.WaitGroup

	// Set up signal handling
	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, os.Interrupt, syscall.SIGTERM)

	// Read credentials
	credentials, err := readCredentials("id_passwd.txt")
	if err != nil {
		log.Println("Error reading credentials:", err)
		os.Exit(1)
	}

	// Start server
	listener, err = net.Listen("tcp", ":9090")
	if err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
	log.Println("TCP server is listening on port 9090...")

	// Channel to signal the accept loop to stop
	done := make(chan struct{})

	// Accept connections in a separate goroutine
	go func() {
		for {
			select {
			case <-done:
				return
			default:
				conn, err := listener.Accept()
				if err != nil {
					if !strings.Contains(err.Error(), "use of closed network connection") {
						log.Printf("Error accepting connection: %v", err)
					}
					continue
				}
				wg.Add(1)
				go handleConnection(conn, credentials, &wg)
			}
		}
	}()

	// Wait for shutdown signal
	<-signalChannel
	log.Println("Interrupt signal received. Starting graceful shutdown...")

	// Close the listener first to stop accepting new connections
	if err := listener.Close(); err != nil {
		log.Printf("Error closing listener: %v", err)
	}

	// Signal the accept loop to stop
	close(done)

	// Close all existing connections
	mu.Lock()
	for addr, conn := range authenticatedSessions {
		log.Printf("Closing connection from %s\n", addr)
		conn.Close()
	}
	mu.Unlock()

	// Wait for all connections to finish
	log.Println("Waiting for all connections to close...")
	wg.Wait()

	log.Println("Server shutdown completed")
}
