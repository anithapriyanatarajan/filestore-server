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

	// Handle file downloads
	http.HandleFunc("/download/", downloadHandler)

	// Handle file updates
	http.HandleFunc("/update/", updateHandler)

	// Handle file deletes
	http.HandleFunc("/delete/", deleteHandler)

	// Handle file listing
	http.HandleFunc("/list", listHandler)

	// Start the server
	port := 8080
	fmt.Printf("Server is running on http://localhost:%d\n", port)
	err = http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
	if err != nil {
		fmt.Println("Error starting server:", err)
	}
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse the form data, limit the upload size to 10 MB
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		http.Error(w, "Unable to parse form", http.StatusBadRequest)
		return
	}

	// Get the file from the form data
	file, handler, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Error retrieving file from form", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Create the file on the server
	filePath := filepath.Join(uploadDirectory, handler.Filename)
	dst, err := os.Create(filePath)
	if err != nil {
		http.Error(w, "Error creating file on server", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	// Copy the uploaded file to the server file
	_, err = io.Copy(dst, file)
	if err != nil {
		http.Error(w, "Error copying file to server", http.StatusInternalServerError)
		return
	}

	// Respond with a success message
	fmt.Fprintf(w, "File '%s' uploaded successfully.", handler.Filename)
}

func downloadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract the filename from the URL
	fileName := r.URL.Path[len("/download/"):]

	// Open the file for reading
	filePath := filepath.Join(uploadDirectory, fileName)
	file, err := os.Open(filePath)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}
	defer file.Close()

	// Set the Content-Disposition header to prompt the user to download the file
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", fileName))

	// Copy the file to the response writer
	_, err = io.Copy(w, file)
	if err != nil {
		http.Error(w, "Error serving file", http.StatusInternalServerError)
		return
	}
}

func updateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract the filename from the URL
	fileName := r.URL.Path[len("/update/"):]

	// Open the existing file for writing
	filePath := filepath.Join(uploadDirectory, fileName)
	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
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

	return files, nil
}
