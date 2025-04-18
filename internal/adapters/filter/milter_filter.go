package filter

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/emersion/go-milter"
	"github.com/mikey/llm-spam-filter/internal/core"
	"go.uber.org/zap"
)

// MilterFilter implements a Milter filter for spam detection
type MilterFilter struct {
	service      *core.SpamFilterService
	logger       *zap.Logger
	listenAddr   string
	server       *milter.Server
	blockSpam    bool
	spamHeader   string
	scoreHeader  string
	reasonHeader string
}

// NewMilterFilter creates a new Milter filter
func NewMilterFilter(
	service *core.SpamFilterService,
	logger *zap.Logger,
	listenAddr string,
	blockSpam bool,
	spamHeader string,
	scoreHeader string,
	reasonHeader string,
) *MilterFilter {
	return &MilterFilter{
		service:      service,
		logger:       logger,
		listenAddr:   listenAddr,
		blockSpam:    blockSpam,
		spamHeader:   spamHeader,
		scoreHeader:  scoreHeader,
		reasonHeader: reasonHeader,
	}
}

// Start starts the Milter filter service
func (f *MilterFilter) Start() error {
	// Create a new Milter server
	f.server = milter.NewServer(f)
	
	// Start the server
	ln, err := net.Listen("tcp", f.listenAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", f.listenAddr, err)
	}
	
	f.logger.Info("Milter filter started", zap.String("address", f.listenAddr))
	
	go func() {
		if err := f.server.Serve(ln); err != nil {
			f.logger.Error("Milter server error", zap.Error(err))
		}
	}()
	
	return nil
}

// Stop stops the Milter filter service
func (f *MilterFilter) Stop() error {
	if f.server != nil {
		return f.server.Close()
	}
	return nil
}

// ProcessEmail processes an email and returns the filtering result
// This is mainly used for testing or direct API calls
func (f *MilterFilter) ProcessEmail(ctx context.Context, email *core.Email) (*core.SpamAnalysisResult, error) {
	return f.service.AnalyzeEmail(ctx, email)
}

// Connect implements the milter.Handler interface
func (f *MilterFilter) Connect(ctx context.Context, hostname string, addr net.Addr) (milter.Response, error) {
	return milter.Continue, nil
}

// Helo implements the milter.Handler interface
func (f *MilterFilter) Helo(ctx context.Context, hostname string) (milter.Response, error) {
	return milter.Continue, nil
}

// MailFrom implements the milter.Handler interface
func (f *MilterFilter) MailFrom(ctx context.Context, from string) (milter.Response, error) {
	return milter.Continue, nil
}

// RcptTo implements the milter.Handler interface
func (f *MilterFilter) RcptTo(ctx context.Context, to string) (milter.Response, error) {
	return milter.Continue, nil
}

// Header implements the milter.Handler interface
func (f *MilterFilter) Header(ctx context.Context, name string, value string) (milter.Response, error) {
	return milter.Continue, nil
}

// Headers implements the milter.Handler interface
func (f *MilterFilter) Headers(ctx context.Context, headers map[string][]string) (milter.Response, error) {
	return milter.Continue, nil
}

// BodyChunk implements the milter.Handler interface
func (f *MilterFilter) BodyChunk(ctx context.Context, chunk []byte) (milter.Response, error) {
	return milter.Continue, nil
}

// Body implements the milter.Handler interface
func (f *MilterFilter) Body(ctx context.Context, headers map[string][]string, body []byte) (milter.Response, error) {
	// Extract email information
	email := &core.Email{
		Headers: headers,
		Body:    string(body),
	}
	
	// Extract From
	if fromHeaders, ok := headers["From"]; ok && len(fromHeaders) > 0 {
		email.From = extractEmailAddress(fromHeaders[0])
	}
	
	// Extract To
	if toHeaders, ok := headers["To"]; ok {
		for _, to := range toHeaders {
			email.To = append(email.To, extractEmailAddress(to))
		}
	}
	
	// Extract Subject
	if subjectHeaders, ok := headers["Subject"]; ok && len(subjectHeaders) > 0 {
		email.Subject = subjectHeaders[0]
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
		return milter.Accept, nil
	}
	
	// Check if the email is spam
	isSpam := f.service.IsSpam(result)
	
	// Add headers
	modifications := milter.ModificationHeader{
		Add: map[string][]string{
			f.spamHeader:   {fmt.Sprintf("%t", isSpam)},
			f.scoreHeader:  {fmt.Sprintf("%.4f", result.Score)},
			f.reasonHeader: {result.Explanation},
		},
	}
	
	f.logger.Info("Processed email", 
		zap.String("from", email.From),
		zap.String("sender_domain", senderDomain),
		zap.Bool("is_spam", isSpam),
		zap.Float64("score", result.Score),
		zap.String("model", result.ModelUsed))
	
	// Reject if it's spam and blocking is enabled
	if isSpam && f.blockSpam {
		f.logger.Info("Rejecting spam email", 
			zap.String("from", email.From),
			zap.String("sender_domain", senderDomain),
			zap.Float64("score", result.Score),
			zap.String("reason", result.Explanation),
			zap.String("model", result.ModelUsed))
		return milter.Reject, nil
	}
	
	// Otherwise, add headers and accept
	return milter.NewResponseModification(&modifications), nil
}

// extractEmailAddress extracts the email address from a string
func extractEmailAddress(s string) string {
	// Simple extraction for addresses like "Name <email@example.com>"
	start := strings.LastIndex(s, "<")
	end := strings.LastIndex(s, ">")
	
	if start >= 0 && end > start {
		return s[start+1 : end]
	}
	
	// If no angle brackets, return as is
	return s
}
