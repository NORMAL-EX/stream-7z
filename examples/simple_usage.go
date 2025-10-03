package main

import (
	"fmt"
	"io"
	"log"
	"os"

	"github.com/NORMAL-EX/stream-7z/lib"
	"github.com/NORMAL-EX/stream-7z/lib/formats"
)

func main() {
	// Example 1: Get archive information
	fmt.Println("Example 1: Get Archive Information")
	fmt.Println("===================================")
	
	archiveURL := "https://example.com/archive.zip"
	password := "" // Leave empty if no password

	// Quick way - creates and closes archive automatically
	info, err := lib.QuickInfo(archiveURL, password, nil)
	if err != nil {
		log.Fatalf("Failed to get info: %v", err)
	}

	fmt.Printf("Archive Information:\n")
	fmt.Printf("  Total Files: %d\n", info.TotalFiles)
	fmt.Printf("  Total Size: %d bytes\n", info.TotalSize)
	fmt.Printf("  Is Encrypted: %v\n", info.IsEncrypted)
	fmt.Printf("  Requires Password: %v\n", info.RequiresPassword)
	
	fmt.Println()

	// Example 2: List files in archive
	fmt.Println("Example 2: List Files")
	fmt.Println("=====================")

	files, err := lib.QuickList(archiveURL, "", password, nil)
	if err != nil {
		log.Fatalf("Failed to list files: %v", err)
	}

	fmt.Printf("Files in archive (%d):\n", len(files))
	for _, file := range files {
		if file.IsDir {
			fmt.Printf("  [DIR]  %s\n", file.Path)
		} else {
			fmt.Printf("  [FILE] %s (%d bytes)\n", file.Path, file.Size)
		}
	}

	fmt.Println()

	// Example 3: Extract a specific file
	fmt.Println("Example 3: Extract File")
	fmt.Println("=======================")

	fileToExtract := "readme.txt"
	reader, size, err := lib.QuickExtract(archiveURL, fileToExtract, password, nil)
	if err != nil {
		log.Fatalf("Failed to extract file: %v", err)
	}
	defer reader.Close()

	// Save to local file
	outFile, err := os.Create("extracted_" + fileToExtract)
	if err != nil {
		log.Fatalf("Failed to create output file: %v", err)
	}
	defer outFile.Close()

	written, err := io.Copy(outFile, reader)
	if err != nil {
		log.Fatalf("Failed to write file: %v", err)
	}

	fmt.Printf("Successfully extracted %s (%d bytes)\n", fileToExtract, written)

	fmt.Println()

	// Example 4: Using Archive instance for multiple operations
	fmt.Println("Example 4: Reusable Archive Instance")
	fmt.Println("=====================================")

	// Create custom config
	config := lib.DefaultConfig()
	config.WithTimeout(60 * 1000000000) // 60 seconds
	config.WithHeader("User-Agent", "MyApp/1.0")

	// Create archive instance
	archive, err := lib.NewArchive(archiveURL, config)
	if err != nil {
		log.Fatalf("Failed to create archive: %v", err)
	}
	defer archive.Close()

	fmt.Printf("Archive Format: %s\n", archive.Format())
	fmt.Printf("Archive Size: %d bytes\n", archive.Size())

	// Get info
	info, err = archive.GetInfo(password)
	if err != nil {
		log.Fatalf("Failed to get info: %v", err)
	}
	fmt.Printf("Total Files: %d\n", info.TotalFiles)

	// List files
	files, err = archive.ListFiles("", password)
	if err != nil {
		log.Fatalf("Failed to list files: %v", err)
	}
	fmt.Printf("File Count: %d\n", len(files))

	// Extract file
	if len(files) > 0 {
		firstFile := files[0]
		if !firstFile.IsDir {
			reader, size, err := archive.ExtractFile(firstFile.Path, password)
			if err != nil {
				log.Printf("Failed to extract: %v", err)
			} else {
				defer reader.Close()
				fmt.Printf("Extracted %s (%d bytes)\n", firstFile.Path, size)
			}
		}
	}

	fmt.Println()

	// Example 5: Handle encrypted archives
	fmt.Println("Example 5: Handle Encrypted Archive")
	fmt.Println("====================================")

	encryptedURL := "https://example.com/encrypted.zip"
	
	// First, check if password is required
	info, err = lib.QuickInfo(encryptedURL, "", nil)
	if err != nil {
		// Check if error is due to password requirement
		if info != nil && info.RequiresPassword {
			fmt.Println("Archive is encrypted, password required")
			
			// Try with password
			correctPassword := "mypassword"
			info, err = lib.QuickInfo(encryptedURL, correctPassword, nil)
			if err != nil {
				fmt.Printf("Password verification failed: %v\n", err)
			} else {
				fmt.Println("Password verified successfully")
			}
		} else {
			fmt.Printf("Error: %v\n", err)
		}
	} else {
		fmt.Println("Archive is not encrypted")
	}

	fmt.Println()

	// Example 6: Custom error handling
	fmt.Println("Example 6: Custom Error Handling")
	fmt.Println("=================================")

	demoErrorHandling()
}

func demoErrorHandling() {
	invalidURL := "https://example.com/nonexistent.zip"
	
	_, err := lib.QuickInfo(invalidURL, "", nil)
	if err != nil {
		fmt.Printf("Error occurred: %v\n", err)
		
		// Handle specific errors
		switch {
		case err.Error() == "invalid or unsupported URL":
			fmt.Println("  → URL is invalid")
		case err.Error() == "unsupported archive format":
			fmt.Println("  → Archive format not supported")
		case err.Error() == "incorrect password for encrypted archive":
			fmt.Println("  → Wrong password")
		case err.Error() == "password required for encrypted archive":
			fmt.Println("  → Password is required")
		default:
			fmt.Println("  → Other error occurred")
		}
	}
}

// Example 7: Iterate through all files
func exampleIterateAllFiles() {
	archiveURL := "https://example.com/archive.zip"
	
	info, err := lib.QuickInfo(archiveURL, "", nil)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Iterating through %d files:\n", len(info.Files))
	
	for _, file := range info.Files {
		if !file.IsDir {
			fmt.Printf("File: %s\n", file.Path)
			fmt.Printf("  Size: %d bytes\n", file.Size)
			fmt.Printf("  Compressed: %d bytes\n", file.CompressedSize)
			fmt.Printf("  Modified: %v\n", file.ModTime)
			fmt.Println()
		}
	}
}

// Example 8: List files in a specific directory
func exampleListSubdirectory() {
	archiveURL := "https://example.com/archive.zip"
	subdirectory := "docs"
	
	files, err := lib.QuickList(archiveURL, subdirectory, "", nil)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Files in '%s' directory:\n", subdirectory)
	for _, file := range files {
		fmt.Printf("  %s\n", file.Path)
	}
}
