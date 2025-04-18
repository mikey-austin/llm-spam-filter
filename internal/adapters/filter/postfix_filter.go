package filter

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/mail"
	"os"
	"strings"
	"time"

	"github.com/emersion/go-smtp"
	"github.com/mikey/llm-spam-filter/internal/core"
	"go.uber.org/zap"
)

// PostfixFilter implements a Postfix content filter
type PostfixFilter struct {
	service        *core.SpamFilterService
	logger         *zap.Logger
	listenAddr     string
	server         *smtp.Server
	blockSpam      bool
	spamHeader     string
	scoreHeader    string
	reasonHeader   string
	postfixAddr    string
	postfixPort    int
	postfixEnabled bool
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
	postfixAddr string,
	postfixPort int,
	postfixEnabled bool,
) *PostfixFilter {
	return &PostfixFilter{
		service:        service,
		logger:         logger,
		listenAddr:     listenAddr,
		blockSpam:      blockSpam,
		spamHeader:     spamHeader,
		scoreHeader:    scoreHeader,
		reasonHeader:   reasonHeader,
		postfixAddr:    postfixAddr,
		postfixPort:    postfixPort,
		postfixEnabled: postfixEnabled,
	}
}

// Start starts the Postfix filter service
func (f *PostfixFilter) Start() error {
	// Create a new SMTP server
	f.server = smtp.NewServer(&smtpBackend{filter: f})
	
	// Configure the server
	f.server.Addr = f.listenAddr
	f.server.Domain = "localhost"
	f.server.ReadTimeout = 30 * time.Second
	f.server.WriteTimeout = 30 * time.Second
	f.server.MaxMessageBytes = 30 * 1024 * 1024 // 30MB
	f.server.MaxRecipients = 50
	f.server.AllowInsecureAuth = true
	
	f.logger.Info("Postfix filter starting", zap.String("address", f.listenAddr))
	
	// Start the server in a goroutine
	go func() {
		if err := f.server.ListenAndServe(); err != nil {
			if err != smtp.ErrServerClosed {
				f.logger.Error("SMTP server error", zap.Error(err))
			}
		}
	}()
	
	return nil
}

// Stop stops the Postfix filter service
func (f *PostfixFilter) Stop() error {
	if f.server != nil {
		return f.server.Close()
	}
	return nil
}

// ProcessEmail processes an email and returns the filtering result
// This is mainly used for testing or direct API calls
func (f *PostfixFilter) ProcessEmail(ctx context.Context, email *core.Email) (*core.SpamAnalysisResult, error) {
	return f.service.AnalyzeEmail(ctx, email)
}

// sendToPostfix sends the processed email back to Postfix on the configured port using go-smtp
func (f *PostfixFilter) sendToPostfix(sender string, recipients []string, emailData []byte) error {
	// Connect to Postfix using go-smtp
	postfixAddr := fmt.Sprintf("%s:%d", f.postfixAddr, f.postfixPort)
	
	// Get hostname for EHLO
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "localhost"
	}
	
	// Connect to the server with a timeout
	conn, err := net.DialTimeout("tcp", postfixAddr, 10*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect to Postfix: %w", err)
	}
	
	// Set a deadline for the connection
	if err := conn.SetDeadline(time.Now().Add(30 * time.Second)); err != nil {
		conn.Close()
		return fmt.Errorf("failed to set connection deadline: %w", err)
	}
	
	// Create a client
	c := smtp.NewClient(conn)
	defer c.Close()
	
	// Send EHLO
	if err := c.Hello(hostname); err != nil {
		return fmt.Errorf("EHLO failed: %w", err)
	}
	
	// Set the sender
	if err := c.Mail(sender, nil); err != nil {
		return fmt.Errorf("MAIL FROM failed: %w", err)
	}
	
	// Set the recipients
	recipientOK := false
	for _, recipient := range recipients {
		if err := c.Rcpt(recipient, nil); err != nil {
			f.logger.Warn("RCPT TO failed for recipient", 
				zap.String("recipient", recipient),
				zap.Error(err))
			// Continue with other recipients even if one fails
		} else {
			recipientOK = true
		}
	}
	
	if !recipientOK {
		return fmt.Errorf("all recipients were rejected")
	}
	
	// Send the email data
	wc, err := c.Data()
	if err != nil {
		return fmt.Errorf("DATA command failed: %w", err)
	}
	
	_, err = wc.Write(emailData)
	if err != nil {
		wc.Close()
		return fmt.Errorf("failed to send email data: %w", err)
	}
	
	if err := wc.Close(); err != nil {
		return fmt.Errorf("failed to close data writer: %w", err)
	}
	
	// Quit the connection
	if err := c.Quit(); err != nil {
		f.logger.Warn("QUIT command failed", zap.Error(err))
		// Not returning an error here as the email has already been sent
	}
	
	return nil
}

// smtpBackend implements the go-smtp Backend interface
type smtpBackend struct {
	filter *PostfixFilter
}

// NewSession creates a new SMTP session
func (b *smtpBackend) NewSession(c *smtp.Conn) (smtp.Session, error) {
	return &smtpSession{
		filter:     b.filter,
		recipients: make([]string, 0),
	}, nil
}

// smtpSession implements the go-smtp Session interface
type smtpSession struct {
	filter     *PostfixFilter
	sender     string
	recipients []string
	data       []byte
}

// Reset resets the session state
func (s *smtpSession) Reset() {
	s.sender = ""
	s.recipients = make([]string, 0)
	s.data = nil
}

// AuthPlain handles PLAIN authentication (not needed for our filter)
func (s *smtpSession) AuthPlain(_ []byte) error {
	return smtp.ErrAuthUnsupported
}

// Mail sets the sender address
func (s *smtpSession) Mail(from string, _ *smtp.MailOptions) error {
	s.sender = from
	return nil
}

// Rcpt adds a recipient
func (s *smtpSession) Rcpt(to string, _ *smtp.RcptOptions) error {
	s.recipients = append(s.recipients, to)
	return nil
}

// Data handles the email data
func (s *smtpSession) Data(r io.Reader) error {
	// Read the message data
	data, err := io.ReadAll(r)
	if err != nil {
		s.filter.logger.Error("Failed to read message data", zap.Error(err))
		return err
	}
	
	// Parse the email message
	msg, err := mail.ReadMessage(bytes.NewReader(data))
	if err != nil {
		s.filter.logger.Error("Failed to parse email message", zap.Error(err))
		return err
	}
	
	// Read the message body
	bodyBytes, err := io.ReadAll(msg.Body)
	if err != nil {
		s.filter.logger.Error("Failed to read message body", zap.Error(err))
		return err
	}
	
	// Create email object
	email := &core.Email{
		Headers: make(map[string][]string),
		Body:    string(bodyBytes),
		From:    s.sender,
		To:      s.recipients,
	}
	
	// Convert headers
	for key, values := range msg.Header {
		email.Headers[key] = values
		
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
	
	result, err := s.filter.service.AnalyzeEmail(ctx, email)
	if err != nil {
		s.filter.logger.Error("Failed to analyze email",
			zap.Error(err),
			zap.String("sender", email.From),
			zap.String("sender_domain", senderDomain))
		return err
	}
	
	// Add headers to the email
	isSpam := result.IsSpam
	
	// Determine action based on spam status
	if isSpam && s.filter.blockSpam {
		// Reject the email
		s.filter.logger.Info("Rejecting spam email",
			zap.String("from", email.From),
			zap.String("sender_domain", senderDomain),
			zap.Float64("score", result.Score),
			zap.String("reason", result.Explanation),
			zap.String("model", result.ModelUsed))
		return fmt.Errorf("550 Rejected as spam (score: %.2f)", result.Score)
	}
	
	// Prepare the modified email with spam headers
	var modifiedEmail bytes.Buffer
	
	// Write all original headers
	for key, values := range msg.Header {
		for _, value := range values {
			fmt.Fprintf(&modifiedEmail, "%s: %s\r\n", key, value)
		}
	}
	
	// Add our spam detection headers
	fmt.Fprintf(&modifiedEmail, "%s: %t\r\n", s.filter.spamHeader, isSpam)
	fmt.Fprintf(&modifiedEmail, "%s: %.4f\r\n", s.filter.scoreHeader, result.Score)
	fmt.Fprintf(&modifiedEmail, "%s: %s\r\n", s.filter.reasonHeader, result.Explanation)
	
	// End of headers
	fmt.Fprintf(&modifiedEmail, "\r\n")
	
	// Write the original email body
	fmt.Fprintf(&modifiedEmail, "%s", email.Body)
	
	if s.filter.postfixEnabled {
		// Send the email back to Postfix on the configured port
		if err := s.filter.sendToPostfix(s.sender, s.recipients, modifiedEmail.Bytes()); err != nil {
			s.filter.logger.Error("Failed to send email back to Postfix",
				zap.Error(err),
				zap.String("sender", email.From))
			return err
		}
	} else {
		// This should never happen in practice as we always want to send back to Postfix
		// But we keep it for completeness
		s.filter.logger.Warn("Postfix forwarding disabled, this is likely a misconfiguration")
	}
	
	s.filter.logger.Info("Processed email",
		zap.String("from", email.From),
		zap.String("sender_domain", senderDomain),
		zap.Bool("is_spam", isSpam),
		zap.Float64("score", result.Score),
		zap.String("model", result.ModelUsed))
	
	return nil
}

// Logout handles SMTP logout (not needed for our filter)
func (s *smtpSession) Logout() error {
	return nil
}
