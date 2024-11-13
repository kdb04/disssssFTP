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

	// Authenticate the user
	username := authenticate(conn, credentials)
	if username == "" {
		conn.Write([]byte("Incorrect username or password. Disconnecting.\n"))
		return
	}

	log.Printf("User '%s' authenticated successfully from %s", username, conn.RemoteAddr().String())

	// Create a unique directory for the authenticated user
	clientDir := filepath.Join(baseDir, username)
	if err := os.MkdirAll(clientDir, 0755); err != nil {
		log.Printf("Error creating directory for user '%s': %v", username, err)
		return
	}

	clientAddr := conn.RemoteAddr().String()
	mu.Lock()
	authenticatedSessions[clientAddr] = conn
	mu.Unlock()

	defer func() {
		mu.Lock()
		delete(authenticatedSessions, clientAddr)
		mu.Unlock()
		log.Printf("User '%s' disconnected", username)
	}()

	conn.Write([]byte("Authentication successful. You are now connected.\n"))
	conn.SetReadDeadline(time.Now().Add(idleTimeout))

	for {
		// Read full file path (assuming it's sent as a string)
		var filePathLen int32
		if err := binary.Read(conn, binary.LittleEndian, &filePathLen); err != nil {
			//log.Printf("Error reading file path length from user '%s': %v", username, err)
			return
		}

		filePathBuf := make([]byte, filePathLen)
		_, err := conn.Read(filePathBuf)
		if err != nil {
			log.Printf("Error reading file path from user '%s': %v", username, err)
			return
		}
		filePath := string(filePathBuf)

		// Extract the filename from the full file path
		fileName := filepath.Base(filePath) // Get only the file name (not the full path)

		// Create the full path where the file will be saved on the server
		fileSavePath := filepath.Join(clientDir, fileName)
		file, err := os.Create(fileSavePath)
		if err != nil {
			log.Printf("Error creating file %s for user '%s': %v", fileSavePath, username, err)
			return
		}
		defer file.Close()

		// Read the file content and write it to the file
		// Read the file content and write it to the file
		// Read the file content and write it to the file
		buf := make([]byte, 1024) // Buffer to read chunks of the file
		for {
			n, err := conn.Read(buf)
			if err == io.EOF {
				// When the file content has been completely received
				log.Printf("User '%s' upload complete for file: %s", username, fileName)

				// Send a success message to the client
				if _, err := conn.Write([]byte("File uploaded successfully.\n")); err != nil {
					log.Printf("Error sending upload confirmation to user '%s': %v", username, err)
				}
				break // Break out of the loop to proceed to the next file (if any)
			}
			if err != nil {
				//log.Printf("Error receiving file content from user '%s': %v", username, err)
				return
			}

			// Write received data to file
			if _, err := file.Write(buf[:n]); err != nil {
				log.Printf("Error writing to file %s for user '%s': %v", fileName, username, err)
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
