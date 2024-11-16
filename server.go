package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
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
	credentials = "id_passwd.txt"
)

func main() {
	// Ensure base upload directory exists
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		log.Fatalf("Error creating base upload directory: %v", err)
	}

	credentials, err := readCredentials(credentials)
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
	// it is now a seperate func
	go handleShutdown(signalChannel, &wg)

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

	handleClientOperations(conn, username, clientDir)
}

// Authentication function to validate the client credentials
func authenticate(conn net.Conn, credentials map[string]string) string {
	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		log.Println("Error reading credentials:", err)
		conn.Write([]byte("Authentication failed: Error reading credentials\n"))
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

	conn.Write([]byte("Authentication failed: Invalid credentials\n"))
	log.Printf("Failed authentication attempt for user: %s", username)
	return ""
}

func handleClientOperations(conn net.Conn, username, clientDir string) {
	reader := bufio.NewReader(conn)

	for {
		if err := conn.SetReadDeadline(time.Now().Add(idleTimeout)); err != nil {
			log.Printf("Error setting read deadline: %v", err)
			return
		}

		opType, err := reader.ReadByte()
		if err != nil {
			if err == io.EOF || strings.Contains(err.Error(), "connection reset by peer") {
				log.Printf("Client %s disconnected", username)
				return
			}
			log.Printf("Error reading operation type from %s: %v", username, err)
			return
		}

		switch opType {
		case 1: // File upload
			// Read filename length
			var fileNameLen int32
			if err := binary.Read(reader, binary.LittleEndian, &fileNameLen); err != nil {
				log.Printf("Error reading filename length: %v", err)
				return
			}

			// Read filename
			fileNameBuf := make([]byte, fileNameLen)
			if _, err := io.ReadFull(reader, fileNameBuf); err != nil {
				log.Printf("Error reading filename: %v", err)
				return
			}
			fileName := string(fileNameBuf)

			// Read file size
			var fileSize int64
			if err := binary.Read(reader, binary.LittleEndian, &fileSize); err != nil {
				log.Printf("Error reading file size: %v", err)
				return
			}

			if err := handleFileUpload(conn, filepath.Join(clientDir, fileName), fileSize, username); err != nil {
				log.Printf("Error handling file upload: %v", err)
				return
			}
			reader.Reset(conn)

		case 2: // File download
			if err := handleFileDownload(conn, username); err != nil {
				log.Printf("Error handling file download for %s: %v", username, err)
				return
			}

		case 5: // List files
			if err := handleListFiles(conn, clientDir); err != nil {
				log.Printf("Error handling list files for %s: %v", username, err)
				return
			}
			reader.Reset(conn) // Reset reader after operation

		default:
			log.Printf("Unknown operation type %d from %s", opType, username)
			return
		}
	}
}

func handleFileUpload(conn net.Conn, filePath string, fileSize int64, username string) error {
	file, err := os.Create(filePath)
	if err != nil {
		conn.Write([]byte("Error: Failed to create file\n"))
		return err
	}
	defer file.Close()

	// Read file content with a buffered reader
	reader := bufio.NewReader(conn)
	bytesReceived := int64(0)
	buf := make([]byte, 1024)

	for bytesReceived < fileSize {
		n, err := reader.Read(buf)
		if err != nil && err != io.EOF {
			conn.Write([]byte("Error: Failed to receive file\n"))
			os.Remove(filePath)
			return err
		}

		if n > 0 {
			if _, err := file.Write(buf[:n]); err != nil {
				conn.Write([]byte("Error: Failed to write file\n"))
				os.Remove(filePath)
				return err
			}
			bytesReceived += int64(n)
		}

		if err == io.EOF {
			break
		}
	}

	log.Printf("File %s received from %s (%d bytes)", filepath.Base(filePath), username, bytesReceived)

	// Send acknowledgment with newline
	if _, err := conn.Write([]byte("Done\n")); err != nil {
		return fmt.Errorf("error sending acknowledgment: %v", err)
	}

	return nil
}

func handleFileDownload(conn net.Conn, username string) error {
	var fileNameLen int32
	if err := binary.Read(conn, binary.LittleEndian, &fileNameLen); err != nil {
		return fmt.Errorf("error reading filename length: %v", err)
	}

	fileNameBytes := make([]byte, fileNameLen)
	if _, err := io.ReadFull(conn, fileNameBytes); err != nil {
		return fmt.Errorf("error reading filename: %v", err)
	}
	fileName := string(fileNameBytes)

	filePath := filepath.Join("uploads", username, fileName)
	file, err := os.Open(filePath)
	if err != nil {
		// File doesn't exist or other error
		// Send error status (0) followed by error message
		if err := binary.Write(conn, binary.LittleEndian, int64(0)); err != nil {
			return fmt.Errorf("error sending error status: %v", err)
		}
		errMsg := fmt.Sprintf("File %s does not exist", fileName)
		errMsgLen := int32(len(errMsg))
		if err := binary.Write(conn, binary.LittleEndian, errMsgLen); err != nil {
			return fmt.Errorf("error sending error message length: %v", err)
		}
		if _, err := conn.Write([]byte(errMsg)); err != nil {
			return fmt.Errorf("error sending error message: %v", err)
		}
		fmt.Printf("Client requested non-existent file: %s\n", fileName)
		return nil
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("error getting file info: %v", err)
	}

	// Send file size (positive number indicates success)
	fileSize := fileInfo.Size()
	if err := binary.Write(conn, binary.LittleEndian, fileSize); err != nil {
		return fmt.Errorf("error sending file size: %v", err)
	}

	buf := make([]byte, 1024)
	bytesSent := int64(0)
	for {
		n, err := file.Read(buf)
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("error reading file: %v", err)
		}

		if _, err := conn.Write(buf[:n]); err != nil {
			return fmt.Errorf("error sending file content: %v", err)
		}
		bytesSent += int64(n)
	}

	fmt.Printf("File '%s' successfully downloaded by user '%s' (%d bytes)\n", fileName, username, bytesSent)
	return nil
}

func handleListFiles(conn net.Conn, clientDir string) error {
	files, err := os.ReadDir(clientDir)
	if err != nil {
		log.Printf("Error reading directory: %v", err)
		return err
	}

	// Send file count
	fileCount := int32(len(files))
	if err := binary.Write(conn, binary.LittleEndian, fileCount); err != nil {
		return fmt.Errorf("error sending file count: %v", err)
	}

	// Send file information
	for _, file := range files {
		info, err := file.Info()
		if err != nil {
			continue
		}

		// Send filename length
		fileNameLen := int32(len(file.Name()))
		if err := binary.Write(conn, binary.LittleEndian, fileNameLen); err != nil {
			return fmt.Errorf("error sending filename length: %v", err)
		}

		// Send filename
		if _, err := conn.Write([]byte(file.Name())); err != nil {
			return fmt.Errorf("error sending filename: %v", err)
		}

		// Send file size
		if err := binary.Write(conn, binary.LittleEndian, info.Size()); err != nil {
			return fmt.Errorf("error sending file size: %v", err)
		}

		// Send modification time
		modTime := info.ModTime().Unix()
		if err := binary.Write(conn, binary.LittleEndian, modTime); err != nil {
			return fmt.Errorf("error sending modification time: %v", err)
		}
	}

	if _, err := conn.Write([]byte{0xFF}); err != nil {
		return fmt.Errorf("error sending completion acknowledgment: %v", err)
	}

	return nil
}

func handleShutdown(signalChannel chan os.Signal, wg *sync.WaitGroup) {
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
}
