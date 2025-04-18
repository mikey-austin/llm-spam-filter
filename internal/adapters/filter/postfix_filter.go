package filter

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/mail"
	"strings"
	"time"

	"github.com/mikey/llm-spam-filter/internal/core"
	"go.uber.org/zap"
)

// PostfixFilter implements a Postfix content filter
type PostfixFilter struct {
	service      *core.SpamFilterService
	logger       *zap.Logger
	listenAddr   string
	listener     net.Listener
	blockSpam    bool
	spamHeader   string
	scoreHeader  string
	reasonHeader string
}

// NewPostfixFilter creates a new Postfix content filter
func NewPostfixFilter(
	service *core.SpamFilterService,
	logger *zap.Logger,
	listenAddr string,
	blockSpam bool,
	spamHeader string,
	scoreHeader string,
	reasonHeader string,
) *PostfixFilter {
	return &PostfixFilter{
		service:      service,
		logger:       logger,
		listenAddr:   listenAddr,
		blockSpam:    blockSpam,
		spamHeader:   spamHeader,
		scoreHeader:  scoreHeader,
		reasonHeader: reasonHeader,
	}
}

// Start starts the Postfix filter service
func (f *PostfixFilter) Start() error {
	var err error
	f.listener, err = net.Listen("tcp", f.listenAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", f.listenAddr, err)
	}
	
	f.logger.Info("Postfix filter started", zap.String("address", f.listenAddr))
	
	go f.acceptConnections()
	
	return nil
}

// Stop stops the Postfix filter service
func (f *PostfixFilter) Stop() error {
	if f.listener != nil {
		return f.listener.Close()
	}
	return nil
}

// acceptConnections accepts incoming connections from Postfix
func (f *PostfixFilter) acceptConnections() {
	for {
		conn, err := f.listener.Accept()
		if err != nil {
			// Check if the listener was closed
			if strings.Contains(err.Error(), "use of closed network connection") {
				return
			}
			f.logger.Error("Failed to accept connection", zap.Error(err))
			continue
		}
		
		go f.handleConnection(conn)
	}
}

// handleConnection processes a single connection from Postfix
func (f *PostfixFilter) handleConnection(conn net.Conn) {
	defer conn.Close()
	
	// Set a timeout for the connection
	if err := conn.SetDeadline(time.Now().Add(30 * time.Second)); err != nil {
		f.logger.Error("Failed to set connection deadline", zap.Error(err))
		return
	}
	
	// Read the entire message into a buffer
	reader := bufio.NewReader(conn)
	var messageBuffer bytes.Buffer
	
	// Read until we find the end of message marker "."
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			f.logger.Error("Failed to read message", zap.Error(err))
			fmt.Fprintf(conn, "500 Failed to read message\n")
			return
		}
		
		// Check for end of message
		if line == ".\r\n" || line == ".\n" {
			break
		}
		
		// Handle dot-stuffing (RFC 5321)
		if strings.HasPrefix(line, "..") {
			line = line[1:]
		}
		
		messageBuffer.WriteString(line)
	}
	
	// Parse the email message
	msg, err := mail.ReadMessage(&messageBuffer)
	if err != nil {
		f.logger.Error("Failed to parse email message", zap.Error(err))
		fmt.Fprintf(conn, "500 Failed to parse email message\n")
		return
	}
	
	// Read the message body
	bodyBytes, err := io.ReadAll(msg.Body)
	if err != nil {
		f.logger.Error("Failed to read message body", zap.Error(err))
		fmt.Fprintf(conn, "500 Failed to read message body\n")
		return
	}
	
	// Create email object
	email := &core.Email{
		Headers: make(map[string][]string),
		Body:    string(bodyBytes),
	}
	
	// Convert headers
	for key, values := range msg.Header {
		email.Headers[key] = values
		
		// Extract From
		if strings.EqualFold(key, "From") && len(values) > 0 {
			addr, err := mail.ParseAddress(values[0])
			if err == nil && addr != nil {
				email.From = addr.Address
			} else {
				email.From = values[0]
			}
		}
		
		// Extract To
		if strings.EqualFold(key, "To") && len(values) > 0 {
			for _, value := range values {
				addrs, err := mail.ParseAddressList(value)
				if err == nil {
					for _, addr := range addrs {
						email.To = append(email.To, addr.Address)
					}
				} else {
					email.To = append(email.To, value)
				}
			}
		}
		
		// Extract Subject
		if strings.EqualFold(key, "Subject") && len(values) > 0 {
			email.Subject = values[0]
		}
	}
	
	// Extract sender domain for logging
	senderDomain := "unknown"
	if parts := strings.Split(email.From, "@"); len(parts) == 2 {
		senderDomain = parts[1]
	}
	
	// Process the email
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	result, err := f.service.AnalyzeEmail(ctx, email)
	if err != nil {
		f.logger.Error("Failed to analyze email", 
			zap.Error(err),
			zap.String("sender", email.From),
			zap.String("sender_domain", senderDomain))
		fmt.Fprintf(conn, "500 Failed to analyze email\n")
		return
	}
	
	// Add headers to the email
	isSpam := f.service.IsSpam(result)
	
	// Determine action based on spam status
	if isSpam && f.blockSpam {
		// Reject the email
		f.logger.Info("Rejecting spam email", 
			zap.String("from", email.From), 
			zap.String("sender_domain", senderDomain),
			zap.Float64("score", result.Score),
			zap.String("reason", result.Explanation),
			zap.String("model", result.ModelUsed))
		fmt.Fprintf(conn, "550 Rejected as spam (score: %.2f)\n", result.Score)
		return
	}
	
	// Add headers and accept the email
	fmt.Fprintf(conn, "220 OK\n")
	
	// Add spam headers
	fmt.Fprintf(conn, "%s: %t\n", f.spamHeader, isSpam)
	fmt.Fprintf(conn, "%s: %.4f\n", f.scoreHeader, result.Score)
	fmt.Fprintf(conn, "%s: %s\n", f.reasonHeader, result.Explanation)
	
	// End of headers
	fmt.Fprintf(conn, "\n")
	
	// Write the original email body
	fmt.Fprintf(conn, "%s\n", email.Body)
	
	// End of message
	fmt.Fprintf(conn, ".\n")
	
	f.logger.Info("Processed email", 
		zap.String("from", email.From),
		zap.String("sender_domain", senderDomain),
		zap.Bool("is_spam", isSpam),
		zap.Float64("score", result.Score),
		zap.String("model", result.ModelUsed))
}

// ProcessEmail processes an email and returns the filtering result
// This is mainly used for testing or direct API calls
func (f *PostfixFilter) ProcessEmail(ctx context.Context, email *core.Email) (*core.SpamAnalysisResult, error) {
	return f.service.AnalyzeEmail(ctx, email)
}
