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
	"syscall"
	"time"

	"golang.org/x/term"
)

const (
	serverAddress = "localhost:9090"
	bufferSize    = 1024
	idleTimeout   = 5 * time.Minute
)

func main() {
	sigChannel := make(chan os.Signal, 1)
	signal.Notify(sigChannel, os.Interrupt, syscall.SIGTERM)

	conn, err := net.Dial("tcp", serverAddress)
	if err != nil {
		log.Fatalf("Failed to connect to server: %v", err)
	}
	defer conn.Close()

	done := make(chan struct{})

	go func() {
		<-sigChannel
		log.Println("\nCtrl+C detected, shutting down the client...")
		close(done)
		conn.Close()
	}()

	// Authentication process
	if !authenticate(conn) {
		return
	}

	// File upload loop
	for {
		select {
		case <-done:
			return
		default:
			// Prompt for file path or exit command
			fmt.Print("Enter file path to upload (or 'exit' to quit): ")
			reader := bufio.NewReader(os.Stdin)
			filePath, _ := reader.ReadString('\n')
			filePath = strings.TrimSpace(filePath)

			if filePath == "exit" {
				// The exit message is commented as per the original code's requirement
				return
			}

			// Check if the file exists and is not a directory
			fileInfo, err := os.Stat(filePath)
			if os.IsNotExist(err) || (err == nil && fileInfo.IsDir()) {
				fmt.Println("File does not exist at the specified path. Please try again.")
				continue
			}

			// Send the file to the server
			if err := sendFile(conn, filePath); err != nil {
				log.Printf("Failed to send file: %v", err)
			}

			// Reset the connection's idle timeout
			conn.SetDeadline(time.Now().Add(idleTimeout))
		}
	}
}

// Authentication method to login the user
func authenticate(conn net.Conn) bool {
	// Prompt the user for username and password
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter username: ")
	username, _ := reader.ReadString('\n')
	username = strings.TrimSpace(username)

	fmt.Print("Enter password: ")
	bytePassword, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		fmt.Println("\nError reading password:", err)
		return false
	}
	password := strings.TrimSpace(string(bytePassword))
	fmt.Println()

	// Send credentials to the server
	credentials := fmt.Sprintf("%s:%s", username, password)
	if _, err := conn.Write([]byte(credentials + "\n")); err != nil {
		fmt.Println("Error sending credentials:", err)
		return false
	}

	// Read the server response for authentication
	serverResponse, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		fmt.Println("Error reading server response:", err)
		return false
	}

	// Check if authentication was successful
	if serverResponse != "Authentication successful. You are now connected.\n" {
		fmt.Printf("Authentication failed. Server response: %s\n", serverResponse)
		return false
	}

	fmt.Println("Authentication successful.")
	return true
}

// Send the file to the server
func sendFile(conn net.Conn, filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("error opening file: %v", err)
	}
	defer file.Close()

	// Get the base name of the file
	fileName := filepath.Base(filePath)
	fileNameBytes := []byte(fileName)
	fileNameLen := int32(len(fileNameBytes))

	// Send file name length
	if err := binary.Write(conn, binary.LittleEndian, fileNameLen); err != nil {
		return fmt.Errorf("error sending filename length: %v", err)
	}

	// Send the file name
	if _, err := conn.Write(fileNameBytes); err != nil {
		return fmt.Errorf("error sending filename: %v", err)
	}

	// Send file content in chunks
	buf := make([]byte, bufferSize)
	for {
		n, err := file.Read(buf)
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("error reading file: %v", err)
		}
		if _, err := conn.Write(buf[:n]); err != nil {
			return fmt.Errorf("error sending file content: %v", err)
		}
	}

	// Await server confirmation
	confirmation := make([]byte, 1024)
	conn.SetReadDeadline(time.Now().Add(10 * time.Second)) // Avoid hanging indefinitely
	if _, err = conn.Read(confirmation); err == nil {
		serverResponse := string(confirmation)
		if strings.Contains(serverResponse, "successfully") {
			// Commented out per the original request
		} else {
			// Commented out per the original request
		}
	} else {
		// If there is a problem receiving confirmation, no action is taken,
		// but the error is silently ignored to prevent the client from crashing.
	}

	// Reset read deadline after confirmation
	conn.SetReadDeadline(time.Time{})

	return nil
}
