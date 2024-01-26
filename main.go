package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

const uploadDirectory = "./uploads"

func main() {
	// Create the upload directory if it doesn't exist
	err := os.MkdirAll(uploadDirectory, 0755)
	if err != nil {
		fmt.Println("Error creating upload directory:", err)
		return
	}

	// Handle file uploads
	http.HandleFunc("/upload", uploadHandler)

	// Handle file updates
	http.HandleFunc("/update/", updateHandler)

	// Handle file deletes
	http.HandleFunc("/delete/", deleteHandler)

	// Handle file listing
	http.HandleFunc("/list", listHandler)

	// Start the server
	port := 8080
	fmt.Printf("Server is running on http://127.0.0.1:%d\n", port)
	err = http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
	if err != nil {
		fmt.Println("Error starting server:", err)
	}
}

func fileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	return !os.IsNotExist(err)
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseMultipartForm(10 << 20) // 10 MB limit
	if err != nil {
		http.Error(w, "Unable to parse form", http.StatusBadRequest)
		return
	}

	file, handler, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Error retrieving file from form", http.StatusBadRequest)
		return
	}
	defer file.Close()

	filePath := fmt.Sprintf("%s/%s", uploadDirectory, handler.Filename)

	if fileExists(filePath) {
		http.Error(w, "Error Adding file to server", http.StatusConflict)
		return
	}
	dst, err := os.Create(filePath)
	if err != nil {
		http.Error(w, "Error creating file on server", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	_, err = io.Copy(dst, file)
	if err != nil {
		http.Error(w, "Error copying file to server", http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "File '%s' uploaded successfully.", handler.Filename)
}

func listHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get the list of files in the upload directory
	files, err := listFiles(uploadDirectory)
	if err != nil {
		http.Error(w, "Error listing files", http.StatusInternalServerError)
		return
	}

	// Respond with the list of files
	w.Header().Set("Content-Type", "text/plain")
	for _, file := range files {
		fmt.Fprintln(w, file)
	}
}

func updateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	// Extract the filename from the URL
	fileName := r.URL.Path[len("/update/"):]
	filePath := filepath.Join(uploadDirectory, fileName)

	// If the file does not exist, create it
	_, err := os.Stat(filePath)
	if err != nil {
		file, err := os.Create(filePath)
		if err != nil {
			http.Error(w, "Error creating file on server", http.StatusInternalServerError)
			return
		}
		defer file.Close()
	}

	// Open the file for editing
	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		http.Error(w, "Cannot Open File", http.StatusNotFound)
		return
	}
	defer file.Close()

	// Copy the new content from the request body to the file
	_, err = io.Copy(file, r.Body)
	if err != nil {
		http.Error(w, "Error updating file", http.StatusInternalServerError)
		return
	}

	// Respond with a success message
	fmt.Fprintf(w, "File '%s' updated successfully.", fileName)
}

func deleteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract the filename from the URL
	fileName := r.URL.Path[len("/delete/"):]

	// Remove the file from the server
	filePath := filepath.Join(uploadDirectory, fileName)
	err := os.Remove(filePath)
	if err != nil {
		http.Error(w, "Error deleting file", http.StatusInternalServerError)
		return
	}

	// Respond with a success message
	fmt.Fprintf(w, "File '%s' deleted successfully.", fileName)
}

func listFiles(directory string) ([]string, error) {
	var files []string

	err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			// Exclude directories from the list
			relativePath, err := filepath.Rel(uploadDirectory, path)
			if err != nil {
				return err
			}
			files = append(files, relativePath)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}
	fmt.Printf("%T\n", files)
	return files, nil
}
