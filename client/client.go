// package main

// import (
// 	"bufio"
// 	"encoding/binary"
// 	"fmt"
// 	"io"
// 	"log"
// 	"net"
// 	"os"
// 	"os/signal"
// 	"strings"
// 	"syscall"
// 	"time"

// 	"golang.org/x/term"
// )

// const (
// 	authServerAddress = "localhost:9090"
// 	fileServerAddress = "localhost:17000"
// 	bufferSize        = 4096
// )

// func main() {
// 	// Handle OS signals for graceful termination
// 	sigChannel := make(chan os.Signal, 1)
// 	signal.Notify(sigChannel, os.Interrupt, syscall.SIGTERM)

// 	// Connect to authentication server
// 	authConn, err := net.Dial("tcp", authServerAddress)
// 	if err != nil {
// 		log.Fatalf("Failed to connect to auth server: %v", err)
// 	}
// 	defer authConn.Close()

// 	// Create channels for cleanup
// 	done := make(chan struct{})
// 	authDone := make(chan bool)

// 	// Handle Ctrl+C termination
// 	go func() {
// 		<-sigChannel
// 		log.Println("\nCtrl+C detected, closing connections...")
// 		close(done)
// 		authConn.Close()
// 	}()

// 	// Authentication process
// 	go func() {
// 		defer close(authDone)

// 		// Request username and password from the user
// 		reader := bufio.NewReader(os.Stdin)
// 		fmt.Print("Enter username: ")
// 		username, _ := reader.ReadString('\n')
// 		username = strings.TrimSpace(username)

// 		fmt.Print("Enter password: ")
// 		bytePassword, err := term.ReadPassword(int(syscall.Stdin))
// 		if err != nil {
// 			fmt.Println("\nError reading password:", err)
// 			authDone <- false
// 			return
// 		}
// 		password := strings.TrimSpace(string(bytePassword))
// 		fmt.Println()

// 		// Send credentials
// 		credentials := fmt.Sprintf("%s:%s", username, password)
// 		_, err = authConn.Write([]byte(credentials + "\n"))
// 		if err != nil {
// 			fmt.Println("Error sending credentials:", err)
// 			authDone <- false
// 			return
// 		}

// 		// Read response from server
// 		serverResponse, err := bufio.NewReader(authConn).ReadString('\n')
// 		if err != nil || serverResponse != "Authentication successful. You are now connected.\n" {
// 			fmt.Printf("Authentication failed. Server response: %s\n", serverResponse)
// 			authDone <- false
// 			return
// 		}

// 		fmt.Println("Authentication successful.")
// 		authDone <- true
// 	}()

// 	// Wait for authentication or interruption
// 	select {
// 	case <-done:
// 		return
// 	case success := <-authDone:
// 		if !success {
// 			return
// 		}
// 	}

// 	// Delay before file transfer
// 	time.Sleep(2 * time.Second)

// 	// Begin file transfer
// 	if len(os.Args) < 2 {
// 		fmt.Println("Please provide the filename for transfer.")
// 		return
// 	}
// 	fileName := os.Args[1]

// 	if _, err := os.Stat(fileName); os.IsNotExist(err) {
// 		fmt.Println("Error: File doesn't exist.")
// 		return
// 	}

// 	// Connect to file transfer server
// 	fileConn, err := net.Dial("tcp", fileServerAddress)
// 	if err != nil {
// 		fmt.Println("Error connecting to file server:", err)
// 		return
// 	}
// 	defer fileConn.Close()

// 	// Send file information to server
// 	fileNameBytes := []byte(fileName)
// 	fileNameLen := int32(len(fileNameBytes))
// 	if err = binary.Write(fileConn, binary.LittleEndian, fileNameLen); err != nil {
// 		fmt.Println("Error sending filename length:", err)
// 		return
// 	}

// 	_, err = fileConn.Write(fileNameBytes)
// 	if err != nil {
// 		fmt.Println("Error sending filename:", err)
// 		return
// 	}

// 	file, err := os.Open(fileName)
// 	if err != nil {
// 		fmt.Println("Error opening file:", err)
// 		return
// 	}
// 	defer file.Close()

// 	// Transfer file content
// 	buf := make([]byte, bufferSize)
// 	for {
// 		n, err := file.Read(buf)
// 		if err != nil {
// 			if err == io.EOF {
// 				fmt.Println("File sent successfully:", fileName)
// 				break
// 			} else {
// 				fmt.Println("Error reading file:", err)
// 				return
// 			}
// 		}
// 		if _, writeErr := fileConn.Write(buf[:n]); writeErr != nil {
// 			fmt.Println("Error sending file content:", writeErr)
// 			return
// 		}
// 	}

// 	// Receive server confirmation
// 	confirmation := make([]byte, 4)
// 	_, err = fileConn.Read(confirmation)
// 	if err == nil && string(confirmation) == "Done" {
// 		fmt.Println("Server confirmed successful file transfer.")
// 	} else {
// 		fmt.Println("Error receiving server confirmation:", err)
// 	}
// }

// package main

// import (
// 	"bufio"
// 	"encoding/binary"
// 	"fmt"
// 	"io"
// 	"log"
// 	"net"
// 	"os"
// 	"os/signal"
// 	"strings"
// 	"syscall"
// 	"time"

// 	"golang.org/x/term"
// )

// const (
// 	authServerAddress = "localhost:9090"
// 	fileServerAddress = "localhost:17000"
// 	bufferSize        = 4096
// )

// var testingMode = true // Set to false when ready to connect to the real authentication server

// // func main() {
// // 	// Handle OS signals for graceful termination
// // 	sigChannel := make(chan os.Signal, 1)
// // 	signal.Notify(sigChannel, os.Interrupt, syscall.SIGTERM)

// // 	// Create channels for cleanup
// // 	done := make(chan struct{})
// // 	authDone := make(chan bool)

// // 	// Handle Ctrl+C termination
// // 	go func() {
// // 		<-sigChannel
// // 		log.Println("\nCtrl+C detected, closing connections...")
// // 		close(done)
// // 	}()

// // 	// Authentication process
// // 	go func() {
// // 		defer close(authDone)

// // 		if testingMode {
// // 			fmt.Println("Testing mode enabled: Skipping authentication server connection.")
// // 			fmt.Println("Authentication successful.")
// // 			authDone <- true
// // 			return
// // 		}

// // 		// Connect to authentication server
// // 		authConn, err := net.Dial("tcp", authServerAddress)
// // 		if err != nil {
// // 			log.Fatalf("Failed to connect to auth server: %v", err)
// // 		}
// // 		defer authConn.Close()

// // 		// Request username and password from the user
// // 		reader := bufio.NewReader(os.Stdin)
// // 		fmt.Print("Enter username: ")
// // 		username, _ := reader.ReadString('\n')
// // 		username = strings.TrimSpace(username)

// // 		fmt.Print("Enter password: ")
// // 		bytePassword, err := term.ReadPassword(int(syscall.Stdin))
// // 		if err != nil {
// // 			fmt.Println("\nError reading password:", err)
// // 			authDone <- false
// // 			return
// // 		}
// // 		password := strings.TrimSpace(string(bytePassword))
// // 		fmt.Println()

// // 		// Send credentials
// // 		credentials := fmt.Sprintf("%s:%s", username, password)
// // 		_, err = authConn.Write([]byte(credentials + "\n"))
// // 		if err != nil {
// // 			fmt.Println("Error sending credentials:", err)
// // 			authDone <- false
// // 			return
// // 		}

// // 		// Read response from server
// // 		serverResponse, err := bufio.NewReader(authConn).ReadString('\n')
// // 		if err != nil || serverResponse != "Authentication successful. You are now connected.\n" {
// // 			fmt.Printf("Authentication failed. Server response: %s\n", serverResponse)
// // 			authDone <- false
// // 			return
// // 		}

// // 		fmt.Println("Authentication successful.")
// // 		authDone <- true
// // 	}()

// // 	// Wait for authentication or interruption
// // 	select {
// // 	case <-done:
// // 		return
// // 	case success := <-authDone:
// // 		if !success {
// // 			return
// // 		}
// // 	}

// // 	// Delay before file transfer
// // 	time.Sleep(2 * time.Second)

// // 	// Begin file transfer
// // 	if len(os.Args) < 2 {
// // 		fmt.Println("Please provide the filename for transfer.")
// // 		return
// // 	}
// // 	fileName := os.Args[1]

// // 	if _, err := os.Stat(fileName); os.IsNotExist(err) {
// // 		fmt.Println("Error: File doesn't exist.")
// // 		return
// // 	}

// // 	// Connect to file transfer server
// // 	fileConn, err := net.Dial("tcp", fileServerAddress)
// // 	if err != nil {
// // 		fmt.Println("Error connecting to file server:", err)
// // 		return
// // 	}
// // 	defer fileConn.Close()

// // 	// Send file information to server
// // 	fileNameBytes := []byte(fileName)
// // 	fileNameLen := int32(len(fileNameBytes))
// // 	if err = binary.Write(fileConn, binary.LittleEndian, fileNameLen); err != nil {
// // 		fmt.Println("Error sending filename length:", err)
// // 		return
// // 	}

// // 	_, err = fileConn.Write(fileNameBytes)
// // 	if err != nil {
// // 		fmt.Println("Error sending filename:", err)
// // 		return
// // 	}

// // 	file, err := os.Open(fileName)
// // 	if err != nil {
// // 		fmt.Println("Error opening file:", err)
// // 		return
// // 	}
// // 	defer file.Close()

// // 	// Transfer file content
// // 	buf := make([]byte, bufferSize)
// // 	for {
// // 		n, err := file.Read(buf)
// // 		if err != nil {
// // 			if err == io.EOF {
// // 				fmt.Println("File sent successfully:", fileName)
// // 				break
// // 			} else {
// // 				fmt.Println("Error reading file:", err)
// // 				return
// // 			}
// // 		}
// // 		if _, writeErr := fileConn.Write(buf[:n]); writeErr != nil {
// // 			fmt.Println("Error sending file content:", writeErr)
// // 			return
// // 		}
// // 	}

// // 	// Receive server confirmation
// // 	confirmation := make([]byte, 4)
// // 	_, err = fileConn.Read(confirmation)
// // 	if err == nil && string(confirmation) == "Done" {
// // 		fmt.Println("Server confirmed successful file transfer.")
// // 	} else {
// // 		fmt.Println("Error receiving server confirmation:", err)
// // 	}
// // }

// func main() {
// 	// Handle OS signals for graceful termination
// 	sigChannel := make(chan os.Signal, 1)
// 	signal.Notify(sigChannel, os.Interrupt, syscall.SIGTERM)

// 	// Connect to authentication server
// 	authConn, err := net.Dial("tcp", authServerAddress)
// 	if err != nil {
// 		log.Fatalf("Failed to connect to auth server: %v", err)
// 	}
// 	defer authConn.Close()

// 	// Create channels for cleanup
// 	done := make(chan struct{})
// 	authDone := make(chan bool)

// 	// Handle Ctrl+C termination
// 	go func() {
// 		<-sigChannel
// 		log.Println("\nCtrl+C detected, closing connections...")
// 		close(done)  // Close the 'done' channel to notify other goroutines
// 		authConn.Close()
// 	}()

// 	// Authentication process
// 	go func() {
// 		defer close(authDone)

// 		// Request username and password from the user
// 		reader := bufio.NewReader(os.Stdin)
// 		fmt.Print("Enter username: ")
// 		username, _ := reader.ReadString('\n')
// 		username = strings.TrimSpace(username)

// 		fmt.Print("Enter password: ")
// 		bytePassword, err := term.ReadPassword(int(syscall.Stdin))
// 		if err != nil {
// 			fmt.Println("\nError reading password:", err)
// 			authDone <- false
// 			return
// 		}
// 		password := strings.TrimSpace(string(bytePassword))
// 		fmt.Println()

// 		// Send credentials
// 		credentials := fmt.Sprintf("%s:%s", username, password)
// 		_, err = authConn.Write([]byte(credentials + "\n"))
// 		if err != nil {
// 			fmt.Println("Error sending credentials:", err)
// 			authDone <- false
// 			return
// 		}

// 		// Read response from server
// 		serverResponse, err := bufio.NewReader(authConn).ReadString('\n')
// 		if err != nil || serverResponse != "Authentication successful. You are now connected.\n" {
// 			fmt.Printf("Authentication failed. Server response: %s\n", serverResponse)
// 			authDone <- false
// 			return
// 		}

// 		fmt.Println("Authentication successful.")
// 		authDone <- true
// 	}()

// 	// Wait for authentication or interruption
// 	select {
// 	case <-done:  // Handle Ctrl+C
// 		return
// 	case success := <-authDone:
// 		if !success {
// 			return
// 		}
// 	}

// 	// Delay before file transfer
// 	time.Sleep(2 * time.Second)

// 	// Begin file transfer
// 	if len(os.Args) < 2 {
// 		fmt.Println("Please provide the filename for transfer.")
// 		return
// 	}
// 	fileName := os.Args[1]

// 	if _, err := os.Stat(fileName); os.IsNotExist(err) {
// 		fmt.Println("Error: File doesn't exist.")
// 		return
// 	}

// 	// Connect to file transfer server
// 	fileConn, err := net.Dial("tcp", fileServerAddress)
// 	if err != nil {
// 		fmt.Println("Error connecting to file server:", err)
// 		return
// 	}
// 	defer fileConn.Close()

// 	// Send file information to server
// 	fileNameBytes := []byte(fileName)
// 	fileNameLen := int32(len(fileNameBytes))
// 	if err = binary.Write(fileConn, binary.LittleEndian, fileNameLen); err != nil {
// 		fmt.Println("Error sending filename length:", err)
// 		return
// 	}

// 	_, err = fileConn.Write(fileNameBytes)
// 	if err != nil {
// 		fmt.Println("Error sending filename:", err)
// 		return
// 	}

// 	file, err := os.Open(fileName)
// 	if err != nil {
// 		fmt.Println("Error opening file:", err)
// 		return
// 	}
// 	defer file.Close()

// 	// Transfer file content
// 	buf := make([]byte, bufferSize)
// 	for {
// 		n, err := file.Read(buf)
// 		if err != nil {
// 			if err == io.EOF {
// 				fmt.Println("File sent successfully:", fileName)
// 				break
// 			} else {
// 				fmt.Println("Error reading file:", err)
// 				return
// 			}
// 		}
// 		if _, writeErr := fileConn.Write(buf[:n]); writeErr != nil {
// 			fmt.Println("Error sending file content:", writeErr)
// 			return
// 		}
// 	}

// 	// Receive server confirmation
// 	confirmation := make([]byte, 4)
// 	_, err = fileConn.Read(confirmation)
// 	if err == nil && string(confirmation) == "Done" {
// 		fmt.Println("Server confirmed successful file transfer.")
// 	} else {
// 		fmt.Println("Error receiving server confirmation:", err)
// 	}
// }

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
	"strings"
	"syscall"
	"time"

	"golang.org/x/term"
)

const (
	serverAddress = "localhost:9090"
	bufferSize    = 4096
	idleTimeout   = 5 * time.Minute
)

func main() {
	// Set up signal handling for graceful shutdown
	sigChannel := make(chan os.Signal, 1)
	signal.Notify(sigChannel, os.Interrupt, syscall.SIGTERM)

	// Connect to the server
	conn, err := net.Dial("tcp", serverAddress)
	if err != nil {
		log.Fatalf("Failed to connect to server: %v", err)
	}
	defer conn.Close()

	// Create a done channel for cleanup coordination
	done := make(chan struct{})

	// Handle signals in a separate goroutine
	go func() {
		<-sigChannel
		log.Println("\nCtrl+C detected, shutting down the client...")
		close(done)
		conn.Close()
	}()

	// Perform authentication
	if !authenticate(conn) {
		return
	}

	// Allow repeated file uploads in a loop until Ctrl+C is detected
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

			// Check if file exists
			if _, err := os.Stat(filePath); os.IsNotExist(err) {
				fmt.Println("File does not exist. Please try again.")
				continue
			}

			// Send the file to the server
			if err := sendFile(conn, filePath); err != nil {
				log.Printf("Failed to send file: %v", err)
			} else {
				fmt.Printf("File %s sent successfully.\n", filePath)
			}

			// Set an idle timeout to disconnect if idle
			conn.SetDeadline(time.Now().Add(idleTimeout))
		}
	}
}

// authenticate sends the username and password to the server and waits for authentication response.
func authenticate(conn net.Conn) bool {
	// Ask for username and password
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

	// Read server response
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

// sendFile sends the file name and content to the server
func sendFile(conn net.Conn, filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("error opening file: %v", err)
	}
	defer file.Close()

	// Send the file name length and name
	fileName := file.Name()
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

	// Send file content
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

	// Await server confirmation
	confirmation := make([]byte, 4)
	if _, err = conn.Read(confirmation); err == nil && string(confirmation) == "Done" {
		fmt.Println("Server confirmed successful file transfer.")
	} else {
		return fmt.Errorf("error receiving server confirmation: %v", err)
	}
	return nil
}
