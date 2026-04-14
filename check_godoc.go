package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

func main() {
	totalExported := 0
	missingDoc := 0
	missingByFile := make(map[string][]string)
	
	// Pattern to match exported declarations
	exportedPattern := regexp.MustCompile(`^(func|type|const|var)\s+([A-Z][a-zA-Z0-9_]*)`)
	
	// Pattern to match godoc comment (with optional spaces before)
	docPattern := regexp.MustCompile(`^\s*// `)
	
	files := []string{
		"src/backup/manager.go",
		"src/config/config.go",
		"src/crypto/crypto.go",
		"src/file/compress.go",
		"src/file/operations.go",
		"src/ftp/server.go",
		"src/logger/logger.go",
		"src/monitor/mem_release_darwin.go",
		"src/monitor/mem_release_linux.go",
		"src/monitor/mem_release_windows.go",
		"src/monitor/mem_windows.go",
		"src/monitor/system.go",
		"src/panel/api_backup.go",
		"src/panel/api_config.go",
		"src/panel/api_file.go",
		"src/panel/api_ftp.go",
		"src/panel/api_log.go",
		"src/panel/api_service.go",
		"src/panel/api_site.go",
		"src/panel/api_ssl.go",
		"src/panel/api_system.go",
		"src/panel/handler.go",
		"src/panel/middleware.go",
		"src/panel/priv_other.go",
		"src/panel/priv_windows.go",
		"src/panel/response.go",
		"src/panel/router.go",
		"src/panel/static.go",
		"src/panel/uptime_other.go",
		"src/panel/uptime_windows.go",
		"src/site/http_server.go",
		"src/site/proxy.go",
		"src/site/server.go",
		"src/site/vhost.go",
		"src/ssl/lego.go",
		"src/ssl/manager.go",
		"src/ssl/provider.go",
	}
	
	for _, file := range files {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			continue
		}
		
		f, err := os.Open(file)
		if err != nil {
			fmt.Printf("Error opening %s: %v\n", file, err)
			continue
		}
		defer f.Close()
		
		var exportedSymbols []string
		var missingSymbols []string
		var hasDoc bool
		scanner := bufio.NewScanner(f)
		var lineNum int
		
		for scanner.Scan() {
			lineNum++
			line := scanner.Text()
			
			// Check for exported declaration
			if match := exportedPattern.FindStringSubmatch(line); match != nil {
				symbolType := match[1]
				symbolName := match[2]
				
				// Skip test files
				if strings.Contains(file, "_test.go") {
					continue
				}
				
				// For types, check if it's a struct or interface
				if symbolType == "type" && strings.Contains(line, "{") {
					// This is a struct/interface definition
					continue
				}
				
				exportedSymbols = append(exportedSymbols, fmt.Sprintf("%s %s", symbolType, symbolName))
				totalExported++
				
				// Check if previous line is a godoc comment
				if !hasDoc {
					missingSymbols = append(missingSymbols, fmt.Sprintf("%s %s", symbolType, symbolName))
					missingDoc++
				}
				hasDoc = false
			} else if docPattern.MatchString(line) {
				// Found a doc comment
				hasDoc = true
			} else if strings.TrimSpace(line) != "" && !strings.HasPrefix(line, "//") {
				// Non-empty, non-comment line resets the doc flag
				hasDoc = false
			}
		}
		
		if len(missingSymbols) > 0 {
			missingByFile[file] = missingSymbols
		}
	}
	
	// Print summary
	fmt.Println("=== Godoc Comment Analysis ===")
	fmt.Printf("Total exported symbols: %d\n", totalExported)
	fmt.Printf("Missing godoc comments: %d (%.1f%%)\n", missingDoc, float64(missingDoc)/float64(totalExported)*100)
	fmt.Println("\n=== Missing godoc comments by file ===")
	
	for file, symbols := range missingByFile {
		fmt.Printf("\n%s:\n", file)
		for _, sym := range symbols {
			fmt.Printf("  - %s\n", sym)
		}
	}
}
