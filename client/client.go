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
	bufferSize    = 32 * 1024 // fix later
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

	go func() {
		<-sigChannel
		log.Println("\nCtrl+C detected, shutting down the client...")
		conn.Close()
		os.Exit(0)
	}()

	if !authenticate(conn) {
		return
	}

	for {
		fmt.Print("\nEnter file path to upload (or 'exit' to quit): ")
		reader := bufio.NewReader(os.Stdin)
		filePath, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("Error reading input: %v\n", err)
			continue
		}

		filePath = strings.TrimSpace(filePath)

		if filePath == "exit" {
			fmt.Println("Exiting...")
			return
		}

		if err := sendFile(conn, filePath); err != nil {
			if strings.Contains(err.Error(), "connection reset by peer") ||
				strings.Contains(err.Error(), "broken pipe") {
				fmt.Println("Connection to server lost")
				return
			}
			fmt.Printf("Failed to send file: %v\n", err)
			continue
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

	if strings.Contains(serverResponse, "Authentication successful") {
		fmt.Println("Authentication successful")
		return true
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

	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("error getting file info: %v", err)
	}

	if fileInfo.IsDir() {
		return fmt.Errorf("cannot send directories")
	}
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

	// Send file size
	if err := binary.Write(conn, binary.LittleEndian, fileInfo.Size()); err != nil {
		return fmt.Errorf("error sending file size: %v", err)
	}

	// Send file content
	buf := make([]byte, bufferSize)
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

		// Show progress
		progress := float64(bytesSent) / float64(fileInfo.Size()) * 100
		fmt.Printf("\rProgress: %.1f%%", progress)
	}
	fmt.Println()

	// Wait for server ack
	response, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		return fmt.Errorf("error reading server response: %v", err)
	}

	if strings.TrimSpace(response) != "Done" {
		return fmt.Errorf("unexpected server response: %s", response)
	}

	fmt.Printf("Successfully sent %s (%d bytes)\n", fileName, bytesSent)
	return nil
}
