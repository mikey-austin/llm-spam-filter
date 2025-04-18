package filter

import (
	"bytes"
	"encoding/base64"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"strings"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/htmlindex"
	"golang.org/x/text/encoding/ianaindex"
)

// extractTextFromMessage extracts the text content from an email message
// For multipart messages, it tries to find text/plain parts
func extractTextFromMessage(msg *mail.Message) (string, error) {
	contentType := msg.Header.Get("Content-Type")
	
	// If it's not a multipart message, decode and return the body
	if !strings.Contains(strings.ToLower(contentType), "multipart/") {
		bodyBytes, err := io.ReadAll(msg.Body)
		if err != nil {
			return "", err
		}
		
		// Check for Content-Transfer-Encoding and decode if necessary
		encoding := msg.Header.Get("Content-Transfer-Encoding")
		decodedBytes, err := decodeContent(bodyBytes, encoding)
		if err != nil {
			// If decoding fails, use the original content
			return string(bodyBytes), nil
		}
		return string(decodedBytes), nil
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
		// Not a multipart message, decode and return the body
		bodyBytes, err := io.ReadAll(msg.Body)
		if err != nil {
			return "", err
		}
		
		// Check for Content-Transfer-Encoding and decode if necessary
		encoding := msg.Header.Get("Content-Transfer-Encoding")
		decodedBytes, err := decodeContent(bodyBytes, encoding)
		if err != nil {
			// If decoding fails, use the original content
			return string(bodyBytes), nil
		}
		return string(decodedBytes), nil
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
			
			// Check for Content-Transfer-Encoding and decode if necessary
			encoding := part.Header.Get("Content-Transfer-Encoding")
			decodedBytes, err := decodeContent(partBytes, encoding)
			if err != nil {
				// If decoding fails, use the original content
				textContent.Write(partBytes)
			} else {
				textContent.Write(decodedBytes)
			}
			textContent.WriteString("\n")
		} else if strings.Contains(strings.ToLower(partContentType), "multipart/") {
			// For nested multipart messages, we'll extract text recursively
			nestedContentType := part.Header.Get("Content-Type")
			nestedMediaType, nestedParams, err := mime.ParseMediaType(nestedContentType)
			if err != nil || !strings.HasPrefix(nestedMediaType, "multipart/") {
				continue
			}
			
			// We don't actually need to use the nested boundary directly
			// since we're creating a new mail.Message for the nested part
			_, ok := nestedParams["boundary"]
			if !ok {
				continue
			}
			
			// Read the entire part into a buffer
			partBytes, err := io.ReadAll(part)
			if err != nil {
				continue
			}
			
			// Create a new mail.Message for the nested part
			nestedMsg := &mail.Message{
				Header: mail.Header{
					"Content-Type": []string{nestedContentType},
				},
				Body: bytes.NewReader(partBytes),
			}
			
			// Extract text from the nested multipart message
			nestedText, err := extractTextFromMessage(nestedMsg)
			if err == nil && nestedText != "" {
				textContent.WriteString(nestedText)
				textContent.WriteString("\n")
			}
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

// decodeContent decodes content based on the Content-Transfer-Encoding
func decodeContent(content []byte, encoding string) ([]byte, error) {
	switch strings.ToLower(encoding) {
	case "base64":
		// Decode base64 content
		decoded := make([]byte, base64.StdEncoding.DecodedLen(len(content)))
		n, err := base64.StdEncoding.Decode(decoded, content)
		if err != nil {
			return nil, err
		}
		return decoded[:n], nil
		
	case "quoted-printable":
		// Decode quoted-printable content
		reader := quotedprintable.NewReader(bytes.NewReader(content))
		return io.ReadAll(reader)
		
	default:
		// For other encodings or no encoding, return the content as is
		return content, nil
	}
}

// decodeEncodedHeader decodes MIME encoded-word syntax in headers
// as per RFC 2047, e.g. "=?UTF-8?B?U3ViamVjdA==?="
func decodeEncodedHeader(header string) (string, error) {
	// Use the standard library's WordDecoder to decode the header
	dec := new(mime.WordDecoder)
	
	// Set a custom charset decoder function to handle non-UTF-8 charsets
	dec.CharsetReader = func(charset string, input io.Reader) (io.Reader, error) {
		enc, err := getEncoding(charset)
		if err != nil {
			// If we can't find the encoding, just return the input
			return input, nil
		}
		return enc.NewDecoder().Reader(input), nil
	}
	
	// Decode the header
	decoded, err := dec.DecodeHeader(header)
	if err != nil {
		// If decoding fails, return the original header
		return header, err
	}
	
	return decoded, nil
}

// getEncoding returns the encoding for a given charset
func getEncoding(charset string) (encoding.Encoding, error) {
	// Try IANA index first
	if enc, err := ianaindex.IANA.Encoding(charset); err == nil && enc != nil {
		return enc, nil
	}
	
	// Try HTML index
	if enc, err := htmlindex.Get(charset); err == nil {
		return enc, nil
	}
	
	// Try some common charsets directly
	switch strings.ToLower(charset) {
	case "windows-1252", "cp1252":
		return charmap.Windows1252, nil
	case "iso-8859-1", "latin1":
		return charmap.ISO8859_1, nil
	case "iso-8859-15", "latin9":
		return charmap.ISO8859_15, nil
	}
	
	// Default to Windows-1252 as a fallback
	return charmap.Windows1252, nil
}
