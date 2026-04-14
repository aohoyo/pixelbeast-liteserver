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
	// Handles: func Name, type Name, const Name, var Name
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
	
	// Process each file
	for _, file := range files {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			continue
		}
		
		f, err := os.Open(file)
		if err != nil {
			fmt.Printf("Error opening %s: %v\n", file, err)
			continue
		}
		
		var exportedSymbols []string
		var missingSymbols []string
		var hasDoc bool
		scanner := bufio.NewScanner(f)
		
		for scanner.Scan() {
			line := scanner.Text()
			
			// Check for exported declaration
			if match := exportedPattern.FindStringSubmatch(line); match != nil {
				symbolType := match[1]
				symbolName := match[2]
				
				// Skip test files
				if strings.Contains(file, "_test.go") {
					continue
				}
				
				// For types, check if it's a struct or interface definition
				if symbolType == "type" && strings.Contains(line, "{") {
					// This is a struct/interface definition - skip these as they're usually documented with the type
					continue
				}
				
				// Special handling for certain cases
				if symbolType == "func" && strings.Contains(line, "(") && !strings.Contains(line, "{") {
					// Function declaration with signature but no body - skip
					continue
				}
				
				exportedSymbols = append(exportedSymbols, fmt.Sprintf("%s %s", symbolType, symbolName))
				totalExported++
				
				// Check if there's a godoc comment immediately before
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
		
		f.Close()
	}
	
	// Print detailed results
	fmt.Println("=== Godoc Comment Analysis Results ===")
	fmt.Printf("Total exported symbols: %d\n", totalExported)
	fmt.Printf("Missing godoc comments: %d (%.1f%%)\n", missingDoc, float64(missingDoc)/float64(totalExported)*100)
	
	fmt.Println("\n=== Symbols Missing Godoc Comments by File ===")
	
	// Group by package for better organization
	packageGroups := make(map[string]map[string][]string)
	for file, symbols := range missingByFile {
		// Extract package name from file path
		pkg := ""
		if strings.Contains(file, "/") {
			parts := strings.Split(file, "/")
			pkg = parts[len(parts)-2] // Second to last is package name
		} else {
			pkg = "unknown"
		}
		
		if _, exists := packageGroups[pkg]; !exists {
			packageGroups[pkg] = make(map[string][]string)
		}
		packageGroups[pkg][file] = symbols
	}
	
	// Print organized results
	for pkg, files := range packageGroups {
		fmt.Printf("\n📁 Package: %s\n", pkg)
		fmt.Println(strings.Repeat("-", 50))
		
		for file, symbols := range files {
			fmt.Printf("  %s:\n", file)
			for _, sym := range symbols {
				fmt.Printf("    - %s\n", sym)
			}
			fmt.Println()
		}
	}
	
	// Summary by package
	fmt.Println("\n=== Summary by Package ===")
	for pkg, files := range packageGroups {
		totalInPkg := 0
		missingInPkg := 0
		
		for _, symbols := range files {
			totalInPkg += len(symbols)
			missingInPkg += len(symbols)
		}
		
		fmt.Printf("%s: %d symbols missing godoc comments\n", pkg, missingInPkg)
	}
	
	if missingDoc == 0 {
		fmt.Println("\n✅ All exported symbols have godoc comments!")
	} else {
		fmt.Println("\n❌ Found exported symbols without godoc comments")
	}
}
