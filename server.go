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

var (
	authenticatedSessions = make(map[string]net.Conn)
	mu                    sync.Mutex
	listener              net.Listener
)

const (
	idleTimeout = 5 * time.Minute
	baseDir     = "./uploads"
)

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

	if _, err := conn.Write([]byte("Authentication successful. You are now connected.\n")); err != nil {
		log.Printf("Error writing authentication success message: %v", err)
		return
	}

	// Receive file from the client
	for {
		if err := conn.SetReadDeadline(time.Now().Add(idleTimeout)); err != nil {
			log.Printf("Error setting read deadline: %v", err)
		}

		// Read file name length and name
		var fileNameLen int32
		if err := binary.Read(conn, binary.LittleEndian, &fileNameLen); err != nil {
			if err == io.EOF || strings.Contains(err.Error(), "connection reset by peer") {
				log.Printf("Client %s disconnected", username)
				return
			}
			log.Printf("Error reading filename length from %s: %v", username, err)
			return
		}

		fileNameBuf := make([]byte, fileNameLen)
		_, err := io.ReadFull(conn, fileNameBuf)
		if err != nil {
			log.Printf("Error reading filename from %s: %v", username, err)
			return
		}
		fileName := string(fileNameBuf)

		var fileSize int64
		if err := binary.Read(conn, binary.LittleEndian, &fileSize); err != nil {
			log.Printf("Error reading file size from %s: %v", username, err)
			return
		}

		// Create file path in the client's directory
		filePath := filepath.Join(clientDir, fileName)
		file, err := os.Create(filePath)
		if err != nil {
			log.Printf("Error creating file %s for %s: %v", filePath, username, err)
			conn.Write([]byte("Error: Failed to create file\n"))
			continue
		}

		// Read file content
		bytesReceived := int64(0)
		buf := make([]byte, 32*1024) // 32KB buffer (prolly split the large file later on!)
		for bytesReceived < fileSize {
			n, err := conn.Read(buf)
			if err != nil {
				if err != io.EOF {
					log.Printf("Error receiving file from %s: %v", username, err)
					file.Close()
					os.Remove(filePath)
					return
				}
				break
			}

			if _, err := file.Write(buf[:n]); err != nil {
				log.Printf("Error writing to file %s for %s: %v", fileName, username, err)
				file.Close()
				os.Remove(filePath)
				return
			}
			bytesReceived += int64(n)
		}

		file.Close()
		log.Printf("File %s received from %s (%d bytes)", fileName, username, bytesReceived)

		// Send ack
		if _, err := conn.Write([]byte("Done\n")); err != nil {
			log.Printf("Error sending acknowledgment to %s: %v", username, err)
			return
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

	credentials, err := readCredentials("id_passwd.txt")
	if err != nil {
		log.Fatalf("Error reading credentials: %v", err)
	}

	listener, err = net.Listen("tcp", ":9090")
	if err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
	defer listener.Close()

	log.Println("TCP server is listening on port 9090...")

	var wg sync.WaitGroup
	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, os.Interrupt, syscall.SIGTERM)

	// Handle shutdown gracefully
	go func() {
		<-signalChannel
		log.Println("Shutting down server...")
		listener.Close()

		mu.Lock()
		for _, conn := range authenticatedSessions {
			conn.Close()
		}
		mu.Unlock()

		wg.Wait()
		log.Println("Server shutdown complete")
		os.Exit(0)
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			if strings.Contains(err.Error(), "use of closed network connection") {
				break
			}
			log.Printf("Error accepting connection: %v", err)
			continue
		}

		wg.Add(1)
		go handleConnection(conn, credentials, &wg)
	}
}
