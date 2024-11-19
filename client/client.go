package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"golang.org/x/term"
)

var (
	serverAddress string
	bufferSize    = 1024
)

func init() {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter server address (e.g., IP:8080):")
	address, _ := reader.ReadString('\n')
	serverAddress = strings.TrimSpace(address)
}

// FileOperation represents different file operations
type FileOperation struct {
	conn net.Conn
}

func main() {
	conn, err := net.Dial("tcp", serverAddress)
	if err != nil {
		log.Fatalf("Failed to connect to server: %v", err)
	}
	defer conn.Close()

	if !authenticate(conn) {
		return
	}

	fileOp := FileOperation{conn: conn}

	for {
		fmt.Println("\nFile Transfer Menu:")
		fmt.Println("1. Upload File")
		fmt.Println("2. Download File")
		fmt.Println("3. View File")
		fmt.Println("4. Delete File")
		fmt.Println("5. List Files")
		fmt.Println("6. Exit")
		fmt.Print("\nEnter your choice: ")

		reader := bufio.NewReader(os.Stdin)
		choice, _ := reader.ReadString('\n')
		choice = strings.TrimSpace(choice)

		switch choice {
		case "1":
			fmt.Print("Enter file path to upload: ")
			filePath, _ := reader.ReadString('\n')
			filePath = strings.TrimSpace(filePath)
			if err := fileOp.uploadFile(filePath); err != nil {
				fmt.Printf("Upload failed: %v\n", err)
			}
		case "2":
			fmt.Print("Enter file name to download: ")
			fileName, _ := reader.ReadString('\n')
			fileName = strings.TrimSpace(fileName)
			if err := fileOp.downloadFile(fileName); err != nil {
				fmt.Printf("Download failed: %v\n", err)
			}
		case "3":
			fmt.Print("Enter file name to view: ")
			fileName, _ := reader.ReadString('\n')
			fileName = strings.TrimSpace(fileName)
			fileOp.viewFile(fileName)
		case "4":
			fmt.Print("Enter file name to delete: ")
			fileName, _ := reader.ReadString('\n')
			fileName = strings.TrimSpace(fileName)
			fileOp.deleteFile(fileName)
		case "5":
			fileOp.listFiles()
		case "6":
			fmt.Println("Exiting...")
			return
		default:
			fmt.Println("Invalid choice. Please try again.")
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

	response := make([]byte, 1024)
	n, err := conn.Read(response)
	if err != nil {
		fmt.Println("Error reading server response:", err)
		return false

	}

	serverResponse := string(response[:n])
	if strings.Contains(serverResponse, "Authentication failed") {
		fmt.Println(strings.TrimSpace(serverResponse))
		return false
	}

	if strings.Contains(serverResponse, "Authentication successful") {
		fmt.Println("Authentication successful")
		return true
	}

	fmt.Println("Unexpected server response")
	return false
}

func (f *FileOperation) uploadFile(filePath string) error {
	// Set deadline for entire operation
	f.conn.SetDeadline(time.Now().Add(5 * time.Minute))
	defer f.conn.SetDeadline(time.Time{})

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("error opening file: %v", err)
	}
	defer file.Close()

	// Send operation type with explicit write
	if _, err := f.conn.Write([]byte{1}); err != nil {
		return fmt.Errorf("error sending operation type: %v", err)
	}

	// Small delay to ensure operation type is received
	time.Sleep(100 * time.Millisecond)

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
	if err := binary.Write(f.conn, binary.LittleEndian, fileNameLen); err != nil {
		return fmt.Errorf("error sending filename length: %v", err)
	}

	// Send file name
	if _, err := f.conn.Write(fileNameBytes); err != nil {
		return fmt.Errorf("error sending filename: %v", err)
	}

	// Send file size
	if err := binary.Write(f.conn, binary.LittleEndian, fileInfo.Size()); err != nil {
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

		if _, err := f.conn.Write(buf[:n]); err != nil {
			return fmt.Errorf("error sending file content: %v", err)
		}
		bytesSent += int64(n)

		// Show progress
		progress := float64(bytesSent) / float64(fileInfo.Size()) * 100
		fmt.Printf("\rProgress: %.1f%%", progress)
	}
	fmt.Println()

	// After sending all file content, flush the connection
	if flusher, ok := f.conn.(interface{ Flush() error }); ok {
		if err := flusher.Flush(); err != nil {
			return fmt.Errorf("error flushing connection: %v", err)
		}
	}

	// After upload, ensure connection is clear
	response := make([]byte, 5) // "Done\n" is 5 bytes
	if _, err := io.ReadFull(f.conn, response); err != nil {
		return fmt.Errorf("error reading server response: %v", err)
	}

	if string(response) != "Done\n" {
		return fmt.Errorf("unexpected server response: %s", response)
	}

	fmt.Printf("Successfully sent %s (%d bytes)\n", fileName, bytesSent)
	return nil
}

func (f *FileOperation) downloadFile(fileName string) error {
	f.conn.SetDeadline(time.Now().Add(5 * time.Minute))
	defer f.conn.SetDeadline(time.Time{})

	if _, err := f.conn.Write([]byte{2}); err != nil {
		return fmt.Errorf("error sending operation type: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	fileNameBytes := []byte(fileName)
	fileNameLen := int32(len(fileNameBytes))

	if err := binary.Write(f.conn, binary.LittleEndian, fileNameLen); err != nil {
		return fmt.Errorf("error sending filename length: %v", err)
	}

	if _, err := f.conn.Write(fileNameBytes); err != nil {
		return fmt.Errorf("error sending filename: %v", err)
	}

	var fileSize int64
	if err := binary.Read(f.conn, binary.LittleEndian, &fileSize); err != nil {
		return fmt.Errorf("error reading file size: %v", err)
	}

	// Check if server reported an error (fileSize == 0)
	if fileSize == 0 {
		var errMsgLen int32
		if err := binary.Read(f.conn, binary.LittleEndian, &errMsgLen); err != nil {
			return fmt.Errorf("error reading error message length: %v", err)
		}

		errMsgBytes := make([]byte, errMsgLen)
		if _, err := io.ReadFull(f.conn, errMsgBytes); err != nil {
			return fmt.Errorf("error reading error message: %v", err)
		}

		return fmt.Errorf("server error: %s", string(errMsgBytes))
	}

	downloadPath := filepath.Join("Downloads", fileName)
	if err := os.MkdirAll("Downloads", os.ModePerm); err != nil {
		return fmt.Errorf("error creating Downloads directory: %v", err)
	}
	file, err := os.Create(downloadPath)
	if err != nil {
		return fmt.Errorf("error creating file: %v", err)
	}
	defer file.Close()

	buf := make([]byte, bufferSize)
	bytesReceived := int64(0)
	for bytesReceived < fileSize {
		n, err := f.conn.Read(buf)
		if err != nil && err != io.EOF {
			return fmt.Errorf("error reading file content: %v", err)
		}

		if n > 0 {
			if _, err := file.Write(buf[:n]); err != nil {
				return fmt.Errorf("error writing to file: %v", err)
			}
			bytesReceived += int64(n)
		}

		if err == io.EOF {
			break
		}
	}

	fmt.Printf("Successfully received %s (%d bytes)\n", fileName, bytesReceived)
	return nil
}

func (f *FileOperation) viewFile(fileName string) {
	//Create temp directory
	tempDir, err := os.MkdirTemp("", "file-view-*")
	if err != nil {
		fmt.Printf("Error creating temporary directory: %v\n", err)
		return
	}
	defer os.RemoveAll(tempDir) //Clear temp directory after viewing is done

	//Deadline for operation
	f.conn.SetDeadline(time.Now().Add(30 * time.Second))
	defer f.conn.SetDeadline(time.Time{})

	if _, err := f.conn.Write([]byte{3}); err != nil {
		fmt.Printf("Error sending operation type: %v\n", err)
		return
	}

	fileNameLen := int32(len(fileName))
	if err := binary.Write(f.conn, binary.LittleEndian, fileNameLen); err != nil {
		fmt.Printf("Error sending filename length: %v\n", err)
		return
	}

	if _, err := f.conn.Write([]byte(fileName)); err != nil {
		fmt.Printf("Error sending filename: %v\n", err)
		return
	}

	status := make([]byte, 1)
	if _, err := f.conn.Read(status); err != nil {
		fmt.Printf("Error reading status: %v\n", err)
		return
	}

	if status[0] == 0 {
		fmt.Println("File not found or error occurred")
		return
	}

	var fileSize int64
	if err := binary.Read(f.conn, binary.LittleEndian, &fileSize); err != nil {
		fmt.Printf("Error reading file size: %v\n", err)
		return
	}

	// Create temporary file
	tempFile := filepath.Join(tempDir, fileName)
	file, err := os.Create(tempFile)
	if err != nil {
		fmt.Printf("Error creating temporary file: %v\n", err)
		return
	}
	defer file.Close()

	// Read and save file content
	buf := make([]byte, 1024)
	fmt.Println("\nFile content:")
	fmt.Println(strings.Repeat("-", 80))

	n, err := f.conn.Read(buf)
	if err != nil && err != io.EOF {
		fmt.Printf("\nError receiving file content: %v\n", err)
		return
	}

	fmt.Print(string(buf[:n])) //Writing to console
	//Writing to temp directory
	if _, err := file.Write(buf[:n]); err != nil {
		fmt.Printf("\nError writing to temporary file: %v\n", err)
		return
	}

	fmt.Println("\n" + strings.Repeat("-", 80))
	fmt.Printf("\nReceived 1024 bytes\n")
}

func (f *FileOperation) deleteFile(fileName string) {
	// Set deadline for operation
	f.conn.SetDeadline(time.Now().Add(30 * time.Second))
	defer f.conn.SetDeadline(time.Time{})

	// Send operation type (4 for delete)
	if _, err := f.conn.Write([]byte{4}); err != nil {
		fmt.Printf("Error sending operation type: %v\n", err)
		return
	}

	// Send filename length and filename
	fileNameBytes := []byte(fileName)
	fileNameLen := int32(len(fileNameBytes))

	if err := binary.Write(f.conn, binary.LittleEndian, fileNameLen); err != nil {
		fmt.Printf("Error sending filename length: %v\n", err)
		return
	}

	if _, err := f.conn.Write(fileNameBytes); err != nil {
		fmt.Printf("Error sending filename: %v\n", err)
		return
	}

	// Read response from server
	status := make([]byte, 1)
	if _, err := io.ReadFull(f.conn, status); err != nil {
		fmt.Printf("Error reading status: %v\n", err)
		return
	}

	if status[0] == 1 {
		fmt.Printf("File '%s' deleted successfully.\n", fileName)
	} else {
		fmt.Printf("Failed to delete file '%s'. File may not exist or an error occurred.\n", fileName)
	}
}

func (f *FileOperation) listFiles() {
	// Set a deadline for the operation
	f.conn.SetDeadline(time.Now().Add(30 * time.Second))
	defer f.conn.SetDeadline(time.Time{})

	// Send list files operation type (5)
	// if _, err := f.conn.Write([]byte{5}); err != nil {
	// 	fmt.Printf("Error sending operation type: %v\n", err)
	// 	return
	// }
	// Send list files operation type (5)
	if err := binary.Write(f.conn, binary.LittleEndian, byte(5)); err != nil {
		fmt.Printf("Error sending operation type: %v\n", err)
		return
	}

	// Read the number of files
	var fileCount int32
	if err := binary.Read(f.conn, binary.LittleEndian, &fileCount); err != nil {
		fmt.Printf("Error reading file count: %v\n", err)
		return
	}

	if fileCount == 0 {
		fmt.Println("No files found in your directory.")
		return
	}

	fmt.Println("\nYour files:")
	fmt.Println(strings.Repeat("-", 76))
	fmt.Printf("%-40s %-15s %-20s\n", "Filename", "Size", "Modified")
	fmt.Println(strings.Repeat("-", 76))

	for i := int32(0); i < fileCount; i++ {
		// Read filename length
		var fileNameLen int32
		if err := binary.Read(f.conn, binary.LittleEndian, &fileNameLen); err != nil {
			fmt.Printf("Error reading filename length: %v\n", err)
			return
		}

		// Read filename
		fileNameBytes := make([]byte, fileNameLen)
		if _, err := io.ReadFull(f.conn, fileNameBytes); err != nil {
			fmt.Printf("Error reading filename: %v\n", err)
			return
		}
		fileName := string(fileNameBytes)

		// Read file size
		var fileSize int64
		if err := binary.Read(f.conn, binary.LittleEndian, &fileSize); err != nil {
			fmt.Printf("Error reading file size: %v\n", err)
			return
		}

		// Read modification time
		var modTime int64
		if err := binary.Read(f.conn, binary.LittleEndian, &modTime); err != nil {
			fmt.Printf("Error reading modification time: %v\n", err)
			return
		}

		// Format the size
		var sizeStr string
		if fileSize < 1024 {
			sizeStr = fmt.Sprintf("%d B", fileSize)
		} else if fileSize < 1024*1024 {
			sizeStr = fmt.Sprintf("%.1f KB", float64(fileSize)/1024)
		} else {
			sizeStr = fmt.Sprintf("%.1f MB", float64(fileSize)/(1024*1024))
		}

		// Format the time
		timeStr := time.Unix(modTime, 0).Format("2006-01-02 15:04:05")

		fmt.Printf("%-40s %-15s %-20s\n", fileName, sizeStr, timeStr)
	}

	ack := make([]byte, 1)
	if _, err := f.conn.Read(ack); err != nil {
		fmt.Printf("Error reading completion acknowledgment: %v\n", err)
		return
	}
	if ack[0] != 0xFF {
		fmt.Println("Invalid completion acknowledgment")
		return
	}

	// Clear any remaining data in connection
	f.conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	discardBuf := make([]byte, 1024)
	for {
		_, err := f.conn.Read(discardBuf)
		if err != nil {
			break
		}
	}
	f.conn.SetReadDeadline(time.Time{})
}
