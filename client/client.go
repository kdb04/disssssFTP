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

	if !authenticate(conn) {
		return
	}

	for {
		select {
		case <-done:
			return
		default:
			fmt.Print("Enter file path to upload (or 'exit' to quit): ")
			reader := bufio.NewReader(os.Stdin)
			filePath, _ := reader.ReadString('\n')
			filePath = strings.TrimSpace(filePath)

			if filePath == "exit" {
				fmt.Println("Exiting...")
				return
			}

			fileInfo, err := os.Stat(filePath)
			if os.IsNotExist(err) || (err == nil && fileInfo.IsDir()) {
				fmt.Println("File does not exist at the specified path. Please try again.")
				continue
			}

			if err := sendFile(conn, filePath); err != nil {
				log.Printf("Failed to send file: %v", err)
			} else {
				fmt.Printf("File %s sent successfully.\n", filePath)
			}

			conn.SetDeadline(time.Now().Add(idleTimeout))
		}
	}
}

func authenticate(conn net.Conn) bool {
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

	credentials := fmt.Sprintf("%s:%s", username, password)
	if _, err := conn.Write([]byte(credentials + "\n")); err != nil {
		fmt.Println("Error sending credentials:", err)
		return false
	}

	serverResponse, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		fmt.Println("Error reading server response:", err)
		return false
	}

	if serverResponse != "Authentication successful. You are now connected.\n" {
		fmt.Printf("Authentication failed. Server response: %s\n", serverResponse)
		return false
	}

	fmt.Println("Authentication successful.")
	return true
}

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

	// Send file name
	if _, err := conn.Write(fileNameBytes); err != nil {
		return fmt.Errorf("error sending filename: %v", err)
	}

	// Send file content in chunks
	buf := make([]byte, bufferSize)
	for {
		n, err := file.Read(buf)
		if err != nil {
			if err == io.EOF {
				fmt.Printf("File %s upload complete.\n", filePath)
				break
			}
			return fmt.Errorf("error reading file: %v", err)
		}
		if _, err := conn.Write(buf[:n]); err != nil {
			return fmt.Errorf("error sending file content: %v", err)
		}
	}

	confirmation := make([]byte, 4)
	if _, err = conn.Read(confirmation); err == nil && string(confirmation) == "Done" {
		fmt.Println("Server confirmed successful file transfer.")
	} else {
		return fmt.Errorf("error receiving server confirmation: %v", err)
	}
	return nil
}
