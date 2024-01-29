package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type FileHash struct {
	FileName string `json:"fileName"`
	Hash     string `json:"hash"`
}

var fileHashes []FileHash
var fileHashesMutex sync.Mutex
var fileStoreMutex sync.Mutex

const uploadDirectory = "./uploads"
const filestoreMetadata = "./metadata" // Optimization with hashing technique.
const jsonFilePath = "./metadata/fileHashes.json"

func init() {
	// Create the upload directory if it doesn't exist
	err := os.MkdirAll(uploadDirectory, 0755)
	if err != nil {
		fmt.Println("Error creating upload directory:", err)
		return
	}

	// Create metadata directory if it doesn't exist
	err = os.MkdirAll(filestoreMetadata, 0755)
	if err != nil {
		fmt.Println("Error creating metadata directory:", err)
		return
	}

	// Create json to store hash or Load eixtsing filehash
	loadFileHashes()
}

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
	http.HandleFunc("/update", updateHandler)

	// 5. Count words in files
	http.HandleFunc("/wordCount", wordCountHandler)

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

	// Write the metadata (filename and hash) to the file
	metadatainfo := FileHash{FileName: handler.Filename, Hash: hash}
	err = appendDataToFile(metadatainfo, jsonFilePath)
	if err != nil {
		http.Error(w, "Unable to write metadata to file", http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, "File '%s' uploaded successfully.", handler.Filename)
}

func updateHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Traversing through update function.")
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	file, handler, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Error retrieving file from form", http.StatusBadRequest)
		return
	}
	defer file.Close()

	filePath := fmt.Sprintf("%s/%s", uploadDirectory, handler.Filename)
	fmt.Println(filePath)
	dst, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
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

	//remove existing hash entry for the updated file.
	loadFileHashes()
	for i, fh := range fileHashes {
		if fh.FileName == handler.Filename {
			fmt.Println(fh.FileName)
			fileHashes = append(fileHashes[:i], fileHashes[i+1:]...)
			err := writeJSONToFile(fileHashes, jsonFilePath)
			if err != nil {
				http.Error(w, "Error removing hash.", http.StatusInternalServerError)
				return
			}
		}
	}

	// Write the metadata (filename and hash) to the file
	metadatainfo := FileHash{FileName: handler.Filename, Hash: hash}
	err = appendDataToFile(metadatainfo, jsonFilePath)
	if err != nil {
		http.Error(w, "Unable to write metadata to file", http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, "File '%s' updated successfully.", handler.Filename)
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
	loadFileHashes()
	//remove hash entry for the deleted file.
	for i, fh := range fileHashes {
		if fh.FileName == fileName {
			fileHashes = append(fileHashes[:i], fileHashes[i+1:]...)
			err := writeJSONToFile(fileHashes, jsonFilePath)
			if err != nil {
				http.Error(w, "Error removing hash.", http.StatusInternalServerError)
				return
			}
		}
	}

	// Respond with a success message
	fmt.Fprintf(w, "File '%s' deleted successfully.", fileName)
}

func writeJSONToFile(data interface{}, filePath string) error {
	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	err = encoder.Encode(data)
	if err != nil {
		return err
	}

	fmt.Println("Data written to", filePath)
	return nil
}

func appendDataToFile(newData FileHash, filePath string) error {
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
	return existingData, nil
}

func loadFileHashes() {
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

	// Find the matching file for the given hash
	matchingFileName, err := findMatchingFile(hashToCompare)
	if err != nil {
		http.Error(w, "Error finding matching file", http.StatusInternalServerError)
		return
	}

	// Return the matching file name as JSON response
	response := map[string]string{"matchingFileName": matchingFileName}
	fmt.Printf("File with matching content identified: %s\n", matchingFileName)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func findMatchingFile(hashToCompare string) (string, error) {
	loadFileHashes()
	fileHashesMutex.Lock()
	defer fileHashesMutex.Unlock()
	// Look for a file with a matching hash in the JSON file
	matchedfilename := ""
	for _, fh := range fileHashes {
		fmt.Println(fh.Hash)
		fmt.Println(hashToCompare)
		if fh.Hash == hashToCompare {
			matchedfilename = fh.FileName
			return matchedfilename, nil
		}
	}
	return "unmatched", nil
}

func copyFileHandler(w http.ResponseWriter, r *http.Request) {
	// Parse request parameters
	r.ParseForm()
	srcFileName := r.Form.Get("src")
	destFileName := r.Form.Get("dest")
	hashstring := r.Form.Get("hashstring")

	// Call the copyFile function
	err := copyFile(srcFileName, destFileName, hashstring)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error copying file: %v", err), http.StatusInternalServerError)
		return
	}
	// Respond with success message
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("File saved by duplciation at server successfully"))
}

func copyFile(srcFile string, destFile string, hashstring string) error {
	srcPath := uploadDirectory + "/" + string(srcFile)
	destPath := uploadDirectory + "/" + string(destFile)

	// Open the source file for reading
	src_file, err := os.Open(srcPath)
	if err != nil {
		fmt.Println("Error opening JSON file:", err)
		return nil
	}
	defer src_file.Close()

	// Create or open the destination file for writing
	dest_file, err := os.OpenFile(destPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer dest_file.Close()

	// Write the metadata (filename and hash) to the file
	metadatainfo := FileHash{FileName: destFile, Hash: hashstring}
	err = appendDataToFile(metadatainfo, jsonFilePath)
	if err != nil {
		return err
	}

	// Copy the content from the source to the destination
	_, copyerr := io.Copy(dest_file, src_file)
	if copyerr != nil {
		return err
	}

	fmt.Printf("File copied from %s to %s\n", srcPath, destPath)
	return nil
}

func wordCountHandler(w http.ResponseWriter, r *http.Request) {
	totalWords, err := countWordsInFiles()
	if err != nil {
		http.Error(w, "Error counting words", http.StatusInternalServerError)
		return
	}
	response := map[string]int{"totalWords": totalWords}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func countWords(text string) int {
	scanner := bufio.NewScanner(strings.NewReader(text))
	scanner.Split(bufio.ScanWords)

	var count int
	for scanner.Scan() {
		count++
	}

	return count
}

func countWordsInFiles() (int, error) {
	fileStoreMutex.Lock()
	defer fileStoreMutex.Unlock()

	var totalWords int

	err := filepath.Walk(uploadDirectory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			fileContent, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			words := countWords(string(fileContent))
			totalWords += words
		}

		return nil
	})

	return totalWords, err
}
