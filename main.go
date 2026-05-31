// LAN File Server - Simple local network file sharing
// Built with Go standard library (no external dependencies)

package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
)

const (
	uploadDir   = "./uploads"
	serverPort  = ":8080"
	maxUploadSize = 100 << 20 // 100 MB limit
)

func main() {
	// Create uploads directory if it doesn't exist
	if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
		fmt.Printf("Failed to create upload directory: %v\n", err)
		return
	}

	// Setup HTTP routes
	http.HandleFunc("/", indexHandler)           // File listing and upload form
	http.HandleFunc("/upload", uploadHandler)    // File upload endpoint
	http.HandleFunc("/download/", downloadHandler) // File download endpoint

	// Print server info
	fmt.Println("╔══════════════════════════════════════╗")
	fmt.Println("║      LAN File Server Started         ║")
	fmt.Println("╚══════════════════════════════════════╝")
	fmt.Printf("Local:   http://localhost%s\n", serverPort)
	
	// Show network IP for LAN access
	if ip := getLocalIP(); ip != "" {
		fmt.Printf("Network: http://%s%s\n", ip, serverPort)
	}
	fmt.Println("\nPress Ctrl+C to stop")

	// Start server
	if err := http.ListenAndServe(serverPort, nil); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}

// getLocalIP returns the machine's local IP address for LAN sharing
func getLocalIP() string {
	// Simple cross-platform IP detection
	// Returns empty string if unable to determine
	switch runtime.GOOS {
	case "windows":
		return getWindowsIP()
	default:
		return getUnixIP()
	}
}

func getWindowsIP() string {
	// Windows-specific IP detection placeholder
	// In production, use net.InterfaceAddrs()
	return ""
}

func getUnixIP() string {
	// Unix-specific IP detection placeholder
	return ""
}

// indexHandler serves the main page with file list and upload form
func indexHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>LAN File Server</title>
    <style>
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: #f5f5f5;
            color: #333;
            max-width: 800px;
            margin: 0 auto;
            padding: 40px 20px;
        }
        h1 {
            color: #2c3e50;
            margin-bottom: 10px;
            font-size: 2em;
        }
        .subtitle {
            color: #7f8c8d;
            margin-bottom: 30px;
        }
        .upload-box {
            background: white;
            border: 2px dashed #3498db;
            border-radius: 10px;
            padding: 40px;
            text-align: center;
            margin-bottom: 30px;
            transition: border-color 0.3s;
        }
        .upload-box:hover {
            border-color: #2980b9;
        }
        .upload-box input[type="file"] {
            margin: 15px 0;
        }
        button {
            background: #3498db;
            color: white;
            border: none;
            padding: 12px 30px;
            border-radius: 5px;
            cursor: pointer;
            font-size: 16px;
            transition: background 0.3s;
        }
        button:hover {
            background: #2980b9;
        }
        .file-list {
            background: white;
            border-radius: 10px;
            padding: 25px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
        }
        .file-list h2 {
            margin-bottom: 20px;
            color: #2c3e50;
        }
        .file-item {
            display: flex;
            justify-content: space-between;
            align-items: center;
            padding: 15px;
            border-bottom: 1px solid #ecf0f1;
            transition: background 0.2s;
        }
        .file-item:hover {
            background: #f8f9fa;
        }
        .file-item:last-child {
            border-bottom: none;
        }
        .file-name {
            font-weight: 500;
            color: #2c3e50;
        }
        .file-size {
            color: #7f8c8d;
            font-size: 0.9em;
            margin-left: 10px;
        }
        .download-btn {
            background: #27ae60;
            padding: 8px 20px;
            font-size: 14px;
            text-decoration: none;
            display: inline-block;
        }
        .download-btn:hover {
            background: #229954;
        }
        .empty-state {
            text-align: center;
            color: #95a5a6;
            padding: 40px;
        }
        .server-info {
            background: #ecf0f1;
            padding: 15px;
            border-radius: 5px;
            margin-bottom: 20px;
            font-family: monospace;
            font-size: 0.9em;
        }
    </style>
</head>
<body>
    <h1>📁 LAN File Server</h1>
    <p class="subtitle">Share files across your local network</p>
    
    <div class="upload-box">
        <h3>Upload File</h3>
        <form method="POST" action="/upload" enctype="multipart/form-data">
            <input type="file" name="file" required />
            <br><br>
            <button type="submit">Upload</button>
        </form>
    </div>
    
    <div class="file-list">
        <h2>📂 Shared Files</h2>`

	// List uploaded files
	files, err := os.ReadDir(uploadDir)
	if err != nil || len(files) == 0 {
		html += `<div class="empty-state">No files uploaded yet</div>`
	} else {
		for _, file := range files {
			info, _ := file.Info()
			size := formatSize(info.Size())
			name := file.Name()
			
			html += fmt.Sprintf(`
		<div class="file-item">
			<div>
				<span class="file-name">%s</span>
				<span class="file-size">(%s)</span>
			</div>
			<a href="/download/%s" class="download-btn">Download</a>
		</div>`, name, size, name)
		}
	}

	html += `
    </div>
</body>
</html>`

	fmt.Fprint(w, html)
}

// uploadHandler processes file uploads
func uploadHandler(w http.ResponseWriter, r *http.Request) {
	// Limit request size
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		http.Error(w, "File too large (max 100MB)", http.StatusBadRequest)
		return
	}

	// Retrieve file from form
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Failed to retrieve file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Security: prevent directory traversal
	filename := filepath.Base(header.Filename)
	if filename == "." || filename == "/" {
		http.Error(w, "Invalid filename", http.StatusBadRequest)
		return
	}

	// Create destination file
	dstPath := filepath.Join(uploadDir, filename)
	dst, err := os.Create(dstPath)
	if err != nil {
		http.Error(w, "Failed to save file", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	// Copy file content
	written, err := io.Copy(dst, file)
	if err != nil {
		os.Remove(dstPath) // Clean up on failure
		http.Error(w, "Failed to write file", http.StatusInternalServerError)
		return
	}

	// Redirect back to home with success
	fmt.Printf("Uploaded: %s (%s)\n", filename, formatSize(written))
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// downloadHandler serves files for download
func downloadHandler(w http.ResponseWriter, r *http.Request) {
	// Extract filename from URL
	filename := filepath.Base(r.URL.Path[len("/download/"):])
	if filename == "." || filename == "/" {
		http.Error(w, "Invalid filename", http.StatusBadRequest)
		return
	}

	// Build secure file path
	filePath := filepath.Join(uploadDir, filename)

	// Security check: ensure file is within upload directory
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	absUploadDir, _ := filepath.Abs(uploadDir)
	if !filepath.HasPrefix(absPath, absUploadDir) {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	// Check if file exists
	info, err := os.Stat(absPath)
	if os.IsNotExist(err) || info.IsDir() {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	// Serve file with proper headers
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	http.ServeFile(w, r, absPath)
}

// formatSize converts bytes to human-readable format
func formatSize(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}