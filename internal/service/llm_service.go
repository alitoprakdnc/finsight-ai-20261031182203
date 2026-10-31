package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"finSight/internal/config"
)

// LLMProvider defines which backend to use.
type LLMProvider string

const (
	ProviderOllama LLMProvider = "ollama"
	ProviderOpenAI LLMProvider = "openai"
)

// LLMResult represents the structured data extracted from OCR text.
type LLMResult struct {
	Vendor    string      `json:"vendor"`
	LineItems []LineItem  `json:"line_items"`
	Taxes     []TaxDetail `json:"taxes"`
}

// LineItem represents a single line item on an invoice/receipt.
type LineItem struct {
	Description string  `json:"description"`
	Amount      float64 `json:"amount"`
	Category    string  `json:"category,omitempty"`
}

// TaxDetail represents a tax entry.
type TaxDetail struct {
	Type   string  `json:"type"`
	Amount float64 `json:"amount"`
}

// LLMService encapsulates calls to either Ollama or OpenAI.
type LLMService struct {
	cfg        *config.Config
	httpClient *http.Client
	logger     zerolog.Logger
}

// NewLLMService creates a new LLMService instance.
func NewLLMService(cfg *config.Config) *LLMService {
	// A shared HTTP client with sensible defaults.
	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			// Keep‑alive and connection pooling are useful for repeated calls.
			MaxIdleConns:        10,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	return &LLMService{
		cfg:        cfg,
		httpClient: client,
		logger:     log.With().Str("service", "llm").Logger(),
	}
}

// ParseOCR sends the raw OCR text to the configured LLM and returns structured data.
func (s *LLMService) ParseOCR(ctx context.Context, ocrText string) (*LLMResult, error) {
	if ocrText == "" {
		return nil, errors.New("ocr text cannot be empty")
	}

	switch LLMProvider(s.cfg.LLMProvider) {
	case ProviderOllama:
		return s.callOllama(ctx, ocrText)
	case ProviderOpenAI:
		return s.callOpenAI(ctx, ocrText)
	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s", s.cfg.LLMProvider)
	}
}

// ---------------------------------------------------------------------------
// Ollama implementation
// ---------------------------------------------------------------------------

type ollamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type ollamaResponse struct {
	Response string `json:"response"` // The generated text (should be JSON)
}

// callOllama builds the request, sends it to the Ollama endpoint and parses the JSON response.
func (s *LLMService) callOllama(ctx context.Context, ocrText string) (*LLMResult, error) {
	endpoint := s.cfg.OllamaEndpoint
	if endpoint == "" {
		return nil, errors.New("ollama endpoint not configured")
	}
	model := s.cfg.OllamaModel
	if model == "" {
		model = "llama3.1:8b" // sensible default
	}

	reqBody := ollamaRequest{
		Model:  model,
		Prompt: buildPrompt(ocrText),
		Stream: false,
	}
	payload, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal ollama request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/api/generate", endpoint), bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("failed to create ollama request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	s.logger.Debug().
		Str("provider", "ollama").
		Str("url", req.URL.String()).
		Msg("sending request to Ollama")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ollama request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama returned status %d: %s", resp.StatusCode, string(b))
	}

	var ollResp ollamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollResp); err != nil {
		return nil, fmt.Errorf("failed to decode ollama response: %w", err)
	}

	return parseLLMJSON(ollResp.Response)
}

// ---------------------------------------------------------------------------
// OpenAI implementation
// ---------------------------------------------------------------------------

type openAIMessage struct {
	Role    string `json:"role"`    // "system" | "user"
	Content string `json:"content"` // message text
}

type openAIRequest struct {
	Model    string          `json:"model"`
	Messages []openAIMessage `json:"messages"`
	Temperature float32     `json:"temperature,omitempty"`
	TopP       float32     `json:"top_p,omitempty"`
}

type openAIChoice struct {
	Message openAIMessage `json:"message"`
	Index   int           `json:"index"`
	FinishReason string   `json:"finish_reason,omitempty"`
}

type openAIResponse struct {
	ID      string        `json:"id"`
	Object  string        `json:"object"`
	Created int64         `json:"created"`
	Choices []openAIChoice `json:"choices"`
	Usage   struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens       int `json:"total_tokens"`
	} `json:"usage"`
}

// callOpenAI builds a chat completion request, sends it to the OpenAI API and parses the JSON result.
func (s *LLMService) callOpenAI(ctx context.Context, ocrText string) (*LLMResult, error) {
	apiKey := s.cfg.OpenAIKey
	if apiKey == "" {
		return nil, errors.New("openai API key not configured")
	}
	model := s.cfg.OpenAIModel
	if model == "" {
		model = "gpt-4o-mini"
	}

	systemPrompt := `You are an AI assistant that extracts structured financial data from raw OCR text of invoices, receipts or contracts.
Return ONLY a JSON object with the following shape:
{
  "vendor": "Vendor Name",
  "line_items": [
    {"description": "Item description", "amount": 12.34, "category": "optional category"}
  ],
  "taxes": [
    {"type": "VAT", "amount": 2.34}
  ]
}
All monetary values must be numbers (no currency symbols). Do not include any additional text, explanations or markdown.`
	userPrompt := fmt.Sprintf("OCR Text:\n%s", ocrText)

	reqBody := openAIRequest{
		Model: model,
		Messages: []openAIMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature: 0.0,
		TopP:        1.0,
	}
	payload, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal openai request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("failed to create openai request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	s.logger.Debug().
		Str("provider", "openai").
		Str("model", model).
		Msg("sending request to OpenAI")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("openai request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("openai returned status %d: %s", resp.StatusCode, string(b))
	}

	var oaResp openAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&oaResp); err != nil {
		return nil, fmt.Errorf("failed to decode openai response: %w", err)
	}
	if len(oaResp.Choices) == 0 {
		return nil, errors.New("openai response contains no choices")
	}

	content := oaResp.Choices[0].Message.Content
	return parseLLMJSON(content)
}

// ---------------------------------------------------------------------------
// Shared helpers
// ---------------------------------------------------------------------------

// buildPrompt creates a concise prompt for Ollama that mirrors the OpenAI system prompt.
func buildPrompt(ocrText string) string {
	return fmt.Sprintf(`Extract structured financial data from the following OCR text and output ONLY a JSON object with the fields:
vendor (string), line_items (array of {description:string, amount:number, category?:string}), taxes (array of {type:string, amount:number}).

OCR Text:
%s

JSON:`, ocrText)
}

// parseLLMJSON attempts to unmarshal a JSON string into LLMResult.
// It trims any surrounding whitespace and returns a detailed error if unmarshalling fails.
func parseLLMJSON(raw string) (*LLMResult, error) {
	trimmed := bytes.TrimSpace([]byte(raw))
	if len(trimmed) == 0 {
		return nil, errors.New("LLM returned empty response")
	}

	var result LLMResult
	if err := json.Unmarshal(trimmed, &result); err != nil {
		// Include a snippet of the raw response for easier debugging.
		snippet := string(trimmed)
		if len(snippet) > 200 {
			snippet = snippet[:200] + "..."
		}
		return nil, fmt.Errorf("failed to unmarshal LLM JSON: %w; response snippet: %s", err, snippet)
	}

	// Basic validation – vendor must be present, at least one line item.
	if result.Vendor == "" {
		return nil, errors.New("parsed LLM result missing vendor")
	}
	if len(result.LineItems) == 0 {
		return nil, errors.New("parsed LLM result contains no line items")
	}
	return &result, nil
}

// ---------------------------------------------------------------------------
// Public helper for tracing – generates a correlation ID for each request.
// ---------------------------------------------------------------------------

// WithCorrelationID returns a new context containing a request‑scoped correlation ID.
// If the parent context already has one, it is reused.
func WithCorrelationID(ctx context.Context) context.Context {
	if cid := ctx.Value(CorrelationIDKey{}); cid != nil {
		return ctx
	}
	newID := uuid.NewString()
	return context.WithValue(ctx, CorrelationIDKey{}, newID)
}

// CorrelationIDKey is the type used for storing the correlation ID in context.
type CorrelationIDKey struct{}

// GetCorrelationID extracts the correlation ID from the context, if present.
func GetCorrelationID(ctx context.Context) (string, bool) {
	if v := ctx.Value(CorrelationIDKey{}); v != nil {
		if id, ok := v.(string); ok {
			return id, true
		}
	}
	return "", false
}