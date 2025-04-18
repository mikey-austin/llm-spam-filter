package filter

import (
	"bytes"
	"io"
	"mime"
	"mime/multipart"
	"net/mail"
	"strings"
)

// extractTextFromMessage extracts the text content from an email message
// For multipart messages, it tries to find text/plain parts
func extractTextFromMessage(msg *mail.Message) (string, error) {
	contentType := msg.Header.Get("Content-Type")
	
	// If it's not a multipart message, just return the body
	if !strings.Contains(strings.ToLower(contentType), "multipart/") {
		bodyBytes, err := io.ReadAll(msg.Body)
		if err != nil {
			return "", err
		}
		return string(bodyBytes), nil
	}
	
	// Parse the Content-Type header to get the boundary
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		// If we can't parse the Content-Type, just return the body
		bodyBytes, err := io.ReadAll(msg.Body)
		if err != nil {
			return "", err
		}
		return string(bodyBytes), nil
	}
	
	if !strings.HasPrefix(mediaType, "multipart/") {
		// Not a multipart message, return the body
		bodyBytes, err := io.ReadAll(msg.Body)
		if err != nil {
			return "", err
		}
		return string(bodyBytes), nil
	}
	
	// Get the boundary
	boundary, ok := params["boundary"]
	if !ok {
		// No boundary found, return the body as is
		bodyBytes, err := io.ReadAll(msg.Body)
		if err != nil {
			return "", err
		}
		return string(bodyBytes), nil
	}
	
	// Create a multipart reader
	mr := multipart.NewReader(msg.Body, boundary)
	
	// Buffer to store text parts
	var textContent bytes.Buffer
	
	// Read each part
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			// If we encounter an error reading parts, just return what we have so far
			if textContent.Len() > 0 {
				return textContent.String(), nil
			}
			// If we haven't found any text content yet, try to read the original body
			bodyBytes, err := io.ReadAll(msg.Body)
			if err != nil {
				return "", err
			}
			return string(bodyBytes), nil
		}
		
		// Get the Content-Type of this part
		partContentType := part.Header.Get("Content-Type")
		
		// If it's a text part, add it to our text content
		if strings.Contains(strings.ToLower(partContentType), "text/plain") {
			partBytes, err := io.ReadAll(part)
			if err != nil {
				continue // Skip this part if we can't read it
			}
			textContent.Write(partBytes)
			textContent.WriteString("\n")
		} else if strings.Contains(strings.ToLower(partContentType), "multipart/") {
			// For nested multipart messages, we'll just skip them for simplicity
			// In a production system, you might want to recursively process them
			continue
		}
		// Skip other parts (attachments, etc.)
	}
	
	// If we found text content, return it
	if textContent.Len() > 0 {
		return textContent.String(), nil
	}
	
	// If we didn't find any text content, return a placeholder
	return "[No text content found in multipart message]", nil
}
