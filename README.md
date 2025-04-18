# LLM Spam Filter

A Postfix content filter application that uses AI models to evaluate whether received emails are spam or not.

## Features

- AI-powered spam detection with multiple provider options:
  - Amazon Bedrock
  - Google Gemini
  - OpenAI
- Implements ports and adapters pattern for flexibility
- Can run as a standalone Postfix content filter
- Caching system to reduce costs by remembering trusted senders
- Multiple cache backends (Memory, SQLite, MySQL)
- Domain whitelist to bypass spam checking for trusted domains
- Configurable body size limit to control LLM costs
- Docker support for easy deployment
- Non-blocking by default (adds headers rather than rejecting messages)
- Shared configuration system between filter and CLI tool

## Architecture

The application follows the hexagonal (ports and adapters) architecture:

- **Core Domain**: Contains the business logic for spam detection
- **Ports**: Define interfaces for the application to interact with external systems
- **Adapters**: Implement the interfaces defined by ports
  - Input adapters: Postfix content filter
  - Output adapters: Amazon Bedrock client, Google Gemini client, OpenAI client, Memory cache, SQLite cache, MySQL cache

## Setup

### Prerequisites

- Go 1.21+
- Docker and Docker Compose
- AWS credentials with Bedrock access (if using Amazon Bedrock)
- Google API key with Gemini access (if using Google Gemini)
- OpenAI API key (if using OpenAI)
- Postfix mail server

### Configuration

See `configs/config.yaml` for configuration options. You can override settings using environment variables:

```bash
# For Amazon Bedrock
export SPAM_FILTER_LLM_PROVIDER=bedrock
export SPAM_FILTER_BEDROCK_MODEL_ID=anthropic.claude-v2
export SPAM_FILTER_BEDROCK_MAX_BODY_SIZE=8192

# For Google Gemini
export SPAM_FILTER_LLM_PROVIDER=gemini
export SPAM_FILTER_GEMINI_API_KEY=your_api_key
export SPAM_FILTER_GEMINI_MODEL_NAME=gemini-pro
export SPAM_FILTER_GEMINI_MAX_BODY_SIZE=8192

# For OpenAI
export SPAM_FILTER_LLM_PROVIDER=openai
export SPAM_FILTER_OPENAI_API_KEY=your_api_key
export SPAM_FILTER_OPENAI_MODEL_NAME=gpt-4
export SPAM_FILTER_OPENAI_MAX_BODY_SIZE=8192

# General settings
export SPAM_FILTER_SPAM_THRESHOLD=0.7
export SPAM_FILTER_SERVER_BLOCK_SPAM=false
```

### Running with Docker

1. Clone the repository:
```bash
git clone https://github.com/mikey/llm-spam-filter.git
cd llm-spam-filter
```

2. Configure credentials based on your chosen LLM provider:

For Amazon Bedrock:
```bash
export AWS_ACCESS_KEY_ID=your_access_key
export AWS_SECRET_ACCESS_KEY=your_secret_key
export AWS_REGION=us-east-1
```

For Google Gemini:
```bash
export SPAM_FILTER_GEMINI_API_KEY=your_gemini_api_key
```

For OpenAI:
```bash
export SPAM_FILTER_OPENAI_API_KEY=your_openai_api_key
```

3. Start the service:
```bash
docker-compose up -d
```

### Postfix Integration

1. Run the setup script:
```bash
sudo ./scripts/setup-postfix.sh
```

2. Restart Postfix:
```bash
sudo systemctl restart postfix
```

## How It Works

1. Postfix receives an email and passes it to the filter
2. The filter extracts the email content and metadata
3. The filter checks if the sender's domain is in the whitelist
   - If whitelisted, the email is marked as non-spam and returned immediately
4. If not whitelisted, the filter checks if the sender is in the cache
5. If not cached, it truncates the email body if it exceeds the configured size limit
6. It sends the email to the configured LLM provider (Amazon Bedrock or Google Gemini) for analysis
7. Based on the analysis result, it adds headers to the email
8. The email is returned to Postfix for delivery
9. The sender is cached to save costs on future emails

## Cache Configuration

You can choose between three cache backends:

- **Memory**: Fast but not persistent across restarts
- **SQLite**: Persistent storage, suitable for small to medium deployments
- **MySQL**: Scalable persistent storage, suitable for larger deployments

Configure the cache in `configs/config.yaml`:

```yaml
cache:
  type: "sqlite"  # or "memory" or "mysql"
  enabled: true
  ttl: "24h"
  sqlite_path: "/data/spam_cache.db"
  mysql_dsn: "user:password@tcp(localhost:3306)/spam_filter"
```

## Whitelist Configuration

You can configure domains to bypass spam checking:

```yaml
spam:
  threshold: 0.7
  whitelisted_domains:
    - "example.com"
    - "trusted-company.org"
    - "internal-domain.net"
```

## Body Size Limit

To control costs and improve performance, you can limit the size of email bodies sent to the LLM:

```yaml
# For Amazon Bedrock
bedrock:
  max_body_size: 4096  # Maximum body size in bytes (0 for no limit)

# For Google Gemini
gemini:
  max_body_size: 4096  # Maximum body size in bytes (0 for no limit)

# For OpenAI
openai:
  max_body_size: 4096  # Maximum body size in bytes (0 for no limit)
```
## LLM Provider Configuration

You can choose between different LLM providers for spam detection:

### Amazon Bedrock

```yaml
llm:
  provider: "bedrock"

bedrock:
  region: "us-east-1"
  model_id: "anthropic.claude-v2"
  max_tokens: 1000
  temperature: 0.1
  top_p: 0.9
  max_body_size: 4096
```

### Google Gemini

```yaml
llm:
  provider: "gemini"

gemini:
  api_key: "your-gemini-api-key"
  model_name: "gemini-pro"
  max_tokens: 1000
  temperature: 0.1
  top_p: 0.9
  max_body_size: 4096
```

### OpenAI

```yaml
llm:
  provider: "openai"

openai:
  api_key: "your-openai-api-key"
  model_name: "gpt-4"
  max_tokens: 1000
  temperature: 0.1
  top_p: 0.9
  max_body_size: 4096
```
