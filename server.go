package main

import (
	"bufio"
	"encoding/binary"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"path/filepath"
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
const baseDir = "./uploads" // Base directory for all client uploads

// Function to read credentials from the file
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
	username := authenticate(conn, credentials)
	if username == "" {
		conn.Write([]byte("Incorrect username or password. Disconnecting.\n"))
		return
	}

	// Create a unique directory for the authenticated user
	clientDir := filepath.Join(baseDir, username)
	if err := os.MkdirAll(clientDir, 0755); err != nil {
		log.Printf("Error creating directory for %s: %v", username, err)
		return
	}

	// Persist session in authenticatedSessions map
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

	// Set read deadline for idle timeout
	conn.SetReadDeadline(time.Now().Add(idleTimeout))

	// Receive file from the client
	for {
		// Read file name length and name
		var fileNameLen int32
		if err := binary.Read(conn, binary.LittleEndian, &fileNameLen); err != nil {
			log.Printf("Error reading filename length from %s: %v", username, err)
			return
		}

		fileNameBuf := make([]byte, fileNameLen)
		_, err := conn.Read(fileNameBuf)
		if err != nil {
			log.Printf("Error reading filename from %s: %v", username, err)
			return
		}
		fileName := string(fileNameBuf)

		// Create file path in the client's directory
		filePath := filepath.Join(clientDir, fileName)
		file, err := os.Create(filePath)
		if err != nil {
			log.Printf("Error creating file %s for %s: %v", filePath, username, err)
			return
		}
		defer file.Close()

		// Read file content and write to the file
		buf := make([]byte, 1024)
		for {
			n, err := conn.Read(buf)
			if err == io.EOF {
				log.Printf("Client: %s has successfully shared file: %s\n", username, fileName)
				conn.Write([]byte("File upload complete.\n"))
				break
			}
			if err != nil {
				log.Printf("Error receiving file from %s: %v", username, err)
				return
			}

			// Write received data to file
			if _, err := file.Write(buf[:n]); err != nil {
				log.Printf("Error writing to file %s for %s: %v", fileName, username, err)
				return
			}
		}
	}
}

// Authentication function to validate the client credentials
func authenticate(conn net.Conn, credentials map[string]string) string {
	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		log.Println("Error reading credentials:", err)
		return ""
	}

	input := strings.TrimSpace(string(buf[:n]))
	parts := strings.Split(input, ":")
	if len(parts) != 2 {
		conn.Write([]byte("Invalid format. Use username:password format.\n"))
		return ""
	}

	username, password := parts[0], parts[1]
	if storedPassword, ok := credentials[username]; ok && storedPassword == password {
		return username
	}
	return ""
}

func main() {
	// Ensure base upload directory exists
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		log.Fatalf("Error creating base upload directory: %v", err)
	}

	var wg sync.WaitGroup
	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, os.Interrupt, syscall.SIGTERM)

	credentials, err := readCredentials("id_passwd.txt")
	if err != nil {
		log.Fatalf("Error reading credentials: %v", err)
	}

	listener, err = net.Listen("tcp", ":9090")
	if err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
	log.Println("TCP server is listening on port 9090...")

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

	<-signalChannel
	log.Println("Interrupt signal received. Starting graceful shutdown...")

	if err := listener.Close(); err != nil {
		log.Printf("Error closing listener: %v", err)
	}

	close(done)

	mu.Lock()
	for addr, conn := range authenticatedSessions {
		log.Printf("Closing connection from %s\n", addr)
		conn.Close()
	}
	mu.Unlock()

	log.Println("Waiting for all connections to close...")
	wg.Wait()
	log.Println("Server shutdown completed")
}
