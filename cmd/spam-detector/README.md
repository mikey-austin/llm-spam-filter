# Spam Detector CLI Tool

A command-line tool for testing the LLM spam filter with different configurations.

## Overview

This tool allows you to quickly test email spam detection using different LLM providers without setting up the full Postfix/Milter integration. It's designed for development, testing, and debugging purposes.

## Usage

```bash
# Basic usage with default settings (Bedrock provider)
cat email.eml | ./spam-detector

# Using a specific provider (OpenAI)
./spam-detector --provider=openai --openai-api-key=your_api_key --file=email.eml

# Using Gemini with custom settings
./spam-detector --provider=gemini --gemini-api-key=your_api_key --gemini-model=gemini-pro --threshold=0.8 --file=spam_example.eml
```

## Command Line Options

### General Options

- `--provider`: LLM provider to use (`bedrock`, `gemini`, or `openai`). Default: `bedrock`
- `--max-tokens`: Maximum tokens for LLM response. Default: `1000`
- `--temperature`: Temperature for LLM generation. Default: `0.1`
- `--top-p`: Top-p for LLM generation. Default: `0.9`
- `--max-body-size`: Maximum email body size to send to LLM. Default: `4096`
- `--threshold`: Threshold for spam detection. Default: `0.7`
- `--whitelist`: Comma-separated list of whitelisted domains. Default: empty
- `--file`: Input email file (use stdin if not specified)
- `--verbose`: Enable verbose logging
- `--json-log`: Output logs in JSON format

### Provider-Specific Options

#### Amazon Bedrock

- `--bedrock-region`: AWS region for Bedrock. Default: `us-east-1`
- `--bedrock-model`: Bedrock model ID. Default: `anthropic.claude-v2`

#### Google Gemini

- `--gemini-api-key`: API key for Google Gemini
- `--gemini-model`: Gemini model name. Default: `gemini-pro`

#### OpenAI

- `--openai-api-key`: API key for OpenAI
- `--openai-model`: OpenAI model name. Default: `gpt-4`

## Examples

### Using with a file

```bash
./spam-detector --file=path/to/email.eml --provider=bedrock
```

### Using with stdin

```bash
cat path/to/email.eml | ./spam-detector --provider=openai --openai-api-key=your_key
```

### Using with whitelisted domains

```bash
./spam-detector --file=email.eml --whitelist="trusted.com,example.org" --verbose
```

### Using with custom thresholds

```bash
./spam-detector --file=email.eml --threshold=0.85 --max-body-size=8192
```

## Output Format

The tool provides a human-readable output with:

1. Email summary (From, To, Subject, Body length)
2. Analysis configuration (Provider, Threshold)
3. Results (Is spam, Score, Confidence, Explanation, Model used, Processing time)

Example output:

```
=== Email Summary ===
From: sender@example.com
To: recipient@example.com
Subject: Special offer just for you!
Body length: 1245 bytes

=== Analysis ===
Provider: openai
Spam threshold: 0.70

=== Results ===
Is spam: true
Spam score: 0.8750
Confidence: 0.9200
Explanation: The email contains multiple spam indicators including excessive use of capital letters, urgency language, and suspicious links.
Model used: gpt-4
Processing time: 1.245s
```
