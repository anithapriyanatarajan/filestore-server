package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
)

type FileHash struct {
	FileName string `json:"fileName"`
	Hash     string `json:"hash"`
}

var fileHashes []FileHash
var fileHashesMutex sync.Mutex

func init() {
	// Create the upload directory if it doesn't exist
	err := os.MkdirAll(uploadDirectory, 0755)
	if err != nil {
		fmt.Println("Error creating upload directory:", err)
		return
	}

	err = os.MkdirAll(filestoreMetadata, 0755)
	if err != nil {
		fmt.Println("Error creating metadata directory:", err)
		return
	}

	loadFileHashes()
}

const uploadDirectory = "./uploads"
const filestoreMetadata = "./metadata" // Optimization with hashing technique.
const jsonFilePath = "./metadata/fileHashes.json"

func main() {

	// 1. Handle file uploads
	http.HandleFunc("/upload", uploadHandler)

	// 1a. FileHashMatching
	http.HandleFunc("/findMatchingFile", findMatchingFileHandler)

	// 1a. CopyFile
	http.HandleFunc("/copyFile", copyFileHandler)

	// 2. Handle file listing
	http.HandleFunc("/list", listHandler)

	// 3. Handle file deletes
	http.HandleFunc("/delete/", deleteHandler)

	// 4. Handle file updates
	http.HandleFunc("/update/", updateHandler)

	// Start the server
	port := 8080
	fmt.Printf("Server is running on http://127.0.0.1:%d\n", port)
	err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
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

	// Store the hash value in JSON file
	hash := r.FormValue("hash")
	fmt.Printf("ireached %s\n", hash)
	// Write the metadata (filename and hash) to the file
	metadatainfo := FileHash{FileName: handler.Filename, Hash: hash}
	err = appendDataToFile(metadatainfo, jsonFilePath)
	if err != nil {
		http.Error(w, "Unable to write metadata to file", http.StatusInternalServerError)
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

// =====================================
func writeJSONToFile(data interface{}, filePath string) error {
	fmt.Println("WriteData")
	//fileHashesMutex.Lock()
	//defer fileHashesMutex.Unlock()
	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}
	defer file.Close()

	fmt.Println(data)
	encoder := json.NewEncoder(file)
	err = encoder.Encode(data)
	if err != nil {
		return err
	}

	fmt.Println("Data written to", filePath)
	return nil
}

func appendDataToFile(newData FileHash, filePath string) error {
	fmt.Println("InsideappenData")
	// Read existing data from the JSON file
	existingData, err := readExistingData(filePath)
	if err != nil {
		return err
	}

	// Append the new data
	existingData = append(existingData, newData)

	// Write the updated data back to the JSON file
	err = writeJSONToFile(existingData, filePath)
	if err != nil {
		return err
	}

	return nil
}

func readExistingData(filePath string) ([]FileHash, error) {
	fmt.Println("ReadExistingData")
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var existingData []FileHash
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&existingData)
	if err != nil && err != io.EOF {
		return nil, err
	}
	fmt.Println(existingData)
	return existingData, nil
}

//=====================================

func loadFileHashes() {
	fmt.Println("accessingloadfileHashes")
	fileHashesMutex.Lock()
	defer fileHashesMutex.Unlock()

	// Append the metadata to a file
	file, err := os.OpenFile(jsonFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Error opening JSON file:", err)
		return
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		fmt.Println("Error getting file info:", err)
		return
	}
	fmt.Println(fileInfo.Size())
	if !(fileInfo.Size() == 0) {
		data, err := os.ReadFile(jsonFilePath)
		if err != nil {
			fmt.Println("Error reading JSON file:", err)
			return
		}
		err = json.Unmarshal(data, &fileHashes)
		if err != nil {
			fmt.Println("Error unmarshalling file hashes:", err)
			return
		}
	}
}

func findMatchingFileHandler(w http.ResponseWriter, r *http.Request) {
	// Parse request parameters
	r.ParseForm()
	hashToCompare := r.Form.Get("hash")
	fmt.Println(hashToCompare)
	// Find the matching file for the given hash
	matchingFileName, err := findMatchingFile(hashToCompare)
	if err != nil {
		http.Error(w, "Error finding matching file", http.StatusInternalServerError)
		return
	}

	// Return the matching file name as JSON response
	response := map[string]string{"matchingFileName": matchingFileName}
	fmt.Println(response)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func findMatchingFile(hashToCompare string) (string, error) {
	fileHashesMutex.Lock()
	defer fileHashesMutex.Unlock()
	fmt.Printf("Inside FUNCTION: %v", hashToCompare)
	// Look for a file with a matching hash in the JSON file
	fmt.Printf("%T\n", fileHashes)
	fmt.Printf("%v\n", fileHashes)
	fmt.Println(len(fileHashes))
	for _, fh := range fileHashes {
		fmt.Println(fh.Hash)
		if fh.Hash == hashToCompare {
			return fh.FileName, nil
		}
	}
	return "unmatched", nil
}

func copyFileHandler(w http.ResponseWriter, r *http.Request) {
	// Parse request parameters
	r.ParseForm()
	srcFilePath := r.Form.Get("src")
	destFilePath := r.Form.Get("dest")

	// Call the copyFile function
	err := copyFile(srcFilePath, destFilePath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error copying file: %v", err), http.StatusInternalServerError)
		return
	}

	// Respond with success message
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("File copied successfully"))
}

func copyFile(srcPath, destPath string) error {
	// Open the source file for reading
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	// Create or open the destination file for writing
	destFile, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer destFile.Close()

	// Copy the content from the source to the destination
	_, err = io.Copy(destFile, srcFile)
	if err != nil {
		return err
	}

	fmt.Printf("File copied from %s to %s\n", srcPath, destPath)
	return nil
}
