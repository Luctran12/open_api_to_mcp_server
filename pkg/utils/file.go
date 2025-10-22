package utils

import (
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"
)

var allowedFileExtensions = []string{".json", ".yaml", ".yml"}
var allowedFileSize = 1 << 20 // 1MB
var allowedMimeTypes = []string{"application/json"} // YAML is often detected as text/plain
var allowedFileSizeError = errors.New("file size must be less than 1MB")

func IsAllowedFileExtension(header *multipart.FileHeader) (bool, error) {
	// Enforce size limit first
	if header.Size > int64(allowedFileSize) {
		return false, allowedFileSizeError
	}

	// Check extension (case-insensitive)
	ext := strings.ToLower(filepath.Ext(header.Filename))
	for _, allowedExt := range allowedFileExtensions {
		if ext == allowedExt {
			// For YAML, extension check is sufficient (MIME often detected as text/plain)
			if ext == ".yaml" || ext == ".yml" {
				return true, nil
			}
			// For JSON, also try to validate MIME if possible
			break
		}
	}

	// Best-effort MIME validation using file header bytes
	file, err := header.Open()
	if err == nil {
		defer file.Close()
		head := make([]byte, 512)
		n, _ := io.ReadFull(file, head)
		if n > 0 && n <= len(head) {
			detected := http.DetectContentType(head[:n])
			for _, allowed := range allowedMimeTypes {
				if detected == allowed {
					return true, nil
				}
			}
		}
	}

	// Allow by extension if it's one of the allowed ones
	for _, allowedExt := range allowedFileExtensions {
		if ext == allowedExt {
			return true, nil
		}
	}

	return false, errors.New("file type not allowed; expected .json, .yaml, or .yml")
}
