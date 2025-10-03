package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/NORMAL-EX/stream-7z/lib"
	"github.com/NORMAL-EX/stream-7z/lib/formats"
	"github.com/schollz/progressbar/v3"
)

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorPurple = "\033[35m"
	colorCyan   = "\033[36m"
)

// treeNode represents a node in the file tree structure
type treeNode struct {
	name     string
	isDir    bool
	size     int64
	children map[string]*treeNode
}

func main() {
	// Setup signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Printf("\n%s[!] Received interrupt signal, exiting...%s\n", colorYellow, colorReset)
		cancel()
		os.Exit(0)
	}()

	printBanner()

	reader := bufio.NewReader(os.Stdin)

	// Get archive URL
	fmt.Printf("%sEnter archive URL: %s", colorCyan, colorReset)
	archiveURL, err := reader.ReadString('\n')
	if err != nil {
		printError("Failed to read input: " + err.Error())
		return
	}
	archiveURL = strings.TrimSpace(archiveURL)

	if archiveURL == "" {
		printError("URL cannot be empty")
		return
	}

	printInfo("Analyzing archive...")

	// Create archive instance with NO timeout limit for large file downloads
	config := lib.DefaultConfig()
	config.WithDebug(true)
	config.WithTimeout(-1 * time.Second) // 负数表示无超时限制
	
	// Also set HTTP client to no timeout
	config.HTTPClient.Timeout = 0 // 0 表示无超时

	archive, err := lib.NewArchive(archiveURL, config)
	if err != nil {
		printError("Failed to open archive: " + err.Error())
		return
	}
	defer archive.Close()

	printSuccess(fmt.Sprintf("Detected format: %s", archive.Format()))
	printInfo(fmt.Sprintf("Archive size: %s", formatBytes(archive.Size())))

	// Get archive info (may require password)
	var password string
	var info *formats.ArchiveInfo

	for {
		info, err = archive.GetInfo(password)
		if err != nil {
			if strings.Contains(err.Error(), "password") {
				if password != "" {
					printError("Incorrect password")
				} else {
					printWarning("Archive is encrypted")
				}

				fmt.Printf("%sEnter password: %s", colorCyan, colorReset)
				password, err = reader.ReadString('\n')
				if err != nil {
					printError("Failed to read password: " + err.Error())
					return
				}
				password = strings.TrimSpace(password)
				continue
			}
			printError("Failed to get archive info: " + err.Error())
			return
		}
		break
	}

	if password != "" {
		printSuccess("Password verified")
	}

	// Display archive information
	fmt.Println()
	printInfo("Archive Contents:")
	fmt.Printf("  Total files: %d\n", info.TotalFiles)
	fmt.Printf("  Total size: %s\n", formatBytes(info.TotalSize))
	if info.IsEncrypted {
		fmt.Printf("  %sEncrypted: Yes%s\n", colorYellow, colorReset)
	}

	// Display file tree
	fmt.Println()
	printInfo("File Tree:")
	displayFileTree(info.Files)

	// Extract file loop
	fmt.Println()
	for {
		fmt.Printf("%sEnter file path to extract (or 'quit' to exit): %s", colorCyan, colorReset)
		filePath, err := reader.ReadString('\n')
		if err != nil {
			printError("Failed to read input: " + err.Error())
			return
		}
		filePath = strings.TrimSpace(filePath)

		if filePath == "" {
			continue
		}

		if filePath == "quit" || filePath == "exit" || filePath == "q" {
			break
		}

		// Extract file
		if err := extractFile(ctx, archive, filePath, password); err != nil {
			printError("Extraction failed: " + err.Error())
			continue
		}
	}

	printSuccess("Done!")
}

func extractFile(ctx context.Context, archive *lib.Archive, filePath string, password string) error {
	printInfo(fmt.Sprintf("Extracting: %s", filePath))

	reader, size, err := archive.ExtractFile(filePath, password)
	if err != nil {
		return err
	}
	defer reader.Close()

	// Create output file
	outputPath := filepath.Base(filePath)
	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	// Create progress bar
	bar := progressbar.DefaultBytes(
		size,
		fmt.Sprintf("Downloading %s", outputPath),
	)

	// Copy with progress
	written, err := io.Copy(io.MultiWriter(outFile, bar), reader)
	if err != nil {
		os.Remove(outputPath)
		return fmt.Errorf("failed to write file: %w", err)
	}

	if written != size {
		os.Remove(outputPath)
		return fmt.Errorf("incomplete download: got %d bytes, expected %d", written, size)
	}

	fmt.Println()
	printSuccess(fmt.Sprintf("Extracted to: %s (%s)", outputPath, formatBytes(written)))

	return nil
}

func displayFileTree(files []formats.FileEntry) {
	// Build tree structure
	root := &treeNode{
		name:     "",
		isDir:    true,
		children: make(map[string]*treeNode),
	}

	// Build tree
	for _, file := range files {
		parts := strings.Split(strings.Trim(file.Path, "/"), "/")
		current := root

		for i, part := range parts {
			if part == "" {
				continue
			}

			if _, exists := current.children[part]; !exists {
				current.children[part] = &treeNode{
					name:     part,
					isDir:    i < len(parts)-1 || file.IsDir,
					size:     file.Size,
					children: make(map[string]*treeNode),
				}
			}
			current = current.children[part]
		}
	}

	// Print tree
	printTree(root, "", true, 0)
}

func printTree(node *treeNode, prefix string, isLast bool, depth int) {
	if depth > 10 {
		return // Prevent too deep trees
	}

	// Skip root
	if node.name != "" {
		connector := "├── "
		if isLast {
			connector = "└── "
		}

		icon := "📄"
		suffix := ""
		if node.isDir {
			icon = "📁"
		} else {
			suffix = fmt.Sprintf(" (%s)", formatBytes(node.size))
		}

		fmt.Printf("%s%s%s %s%s%s\n", prefix, connector, icon, node.name, suffix, colorReset)
	}

	// Print children
	childPrefix := prefix
	if node.name != "" {
		if isLast {
			childPrefix += "    "
		} else {
			childPrefix += "│   "
		}
	}

	i := 0
	total := len(node.children)
	for _, child := range node.children {
		i++
		printTree(child, childPrefix, i == total, depth+1)
	}
}

func printBanner() {
	banner := `
╔═══════════════════════════════════════╗
║       Stream-7z Demo Program         ║
║  HTTP Range-based Archive Preview    ║
╚═══════════════════════════════════════╝
`
	fmt.Printf("%s%s%s\n", colorPurple, banner, colorReset)
}

func printSuccess(msg string) {
	fmt.Printf("%s[✓]%s %s\n", colorGreen, colorReset, msg)
}

func printError(msg string) {
	fmt.Printf("%s[✗]%s %s\n", colorRed, colorReset, msg)
}

func printWarning(msg string) {
	fmt.Printf("%s[!]%s %s\n", colorYellow, colorReset, msg)
}

func printInfo(msg string) {
	fmt.Printf("%s[i]%s %s\n", colorBlue, colorReset, msg)
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
