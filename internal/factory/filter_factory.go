package factory

import (
	"fmt"

	"github.com/mikey/llm-spam-filter/internal/adapters/filter"
	"github.com/mikey/llm-spam-filter/internal/config"
	"github.com/mikey/llm-spam-filter/internal/core"
	"github.com/mikey/llm-spam-filter/internal/ports"
	"go.uber.org/zap"
)

// FilterFactory creates email filters based on configuration
type FilterFactory struct {
	cfg         *config.Config
	logger      *zap.Logger
	spamService *core.SpamFilterService
}

// NewFilterFactory creates a new filter factory
func NewFilterFactory(cfg *config.Config, logger *zap.Logger, spamService *core.SpamFilterService) *FilterFactory {
	return &FilterFactory{
		cfg:         cfg,
		logger:      logger,
		spamService: spamService,
	}
}

// CreateEmailFilter creates an email filter based on the configuration
func (f *FilterFactory) CreateEmailFilter() (ports.EmailFilter, error) {
	filterType := f.cfg.GetString("server.filter_type")
	
	switch filterType {
	case "postfix":
		return filter.NewPostfixFilter(
			f.spamService,
			f.logger,
			f.cfg.GetString("server.listen_address"),
			f.cfg.GetBool("server.block_spam"),
			f.cfg.GetString("server.headers.spam"),
			f.cfg.GetString("server.headers.score"),
			f.cfg.GetString("server.headers.reason"),
			f.cfg.GetString("server.postfix.address"),
			f.cfg.GetInt("server.postfix.port"),
			f.cfg.GetBool("server.postfix.enabled"),
			f.cfg.GetString("server.subject_prefix"),
			f.cfg.GetBool("server.modify_subject"),
		), nil
	case "cli":
		return filter.NewCliFilter(
			f.spamService,
			f.logger,
			f.cfg.GetBool("cli.verbose"),
		)
	default:
		return nil, fmt.Errorf("unsupported filter type: %s", filterType)
	}
}
