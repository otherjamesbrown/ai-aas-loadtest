package simulator

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"time"

	"github.com/otherjamesbrown/ai-aas-loadtest/internal/config"
	"github.com/otherjamesbrown/ai-aas-loadtest/internal/questions"
	"go.uber.org/zap"
)

// UserSimulator simulates a single user making API requests
type UserSimulator struct {
	userID       string
	userName     string
	orgID        string
	apiKey       string
	apiRouterURL string
	httpClient   *http.Client
	rng          *rand.Rand
	logger       *zap.Logger
	metrics      MetricsCollector
}

// MetricsCollector defines the interface for collecting metrics during simulation
type MetricsCollector interface {
	// RecordRequest records a completed request with its metrics
	RecordRequest(userID, orgID, model string, latency time.Duration, tokens int, success bool, errorType string)

	// RecordLLMMetrics records LLM-specific performance metrics
	RecordLLMMetrics(userID, model string, ttft time.Duration, tps float64)
}

// NewUserSimulator creates a new user simulator
func NewUserSimulator(
	user *config.BootstrappedUser,
	apiKey string,
	apiRouterURL string,
	seed int64,
	timeout time.Duration,
	logger *zap.Logger,
	metrics MetricsCollector,
) *UserSimulator {
	return &UserSimulator{
		userID:       user.ID,
		userName:     user.Name,
		orgID:        user.OrgID,
		apiKey:       apiKey,
		apiRouterURL: apiRouterURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		rng:     rand.New(rand.NewSource(seed)),
		logger:  logger,
		metrics: metrics,
	}
}

// Run executes the user simulation based on the provided configuration
func (s *UserSimulator) Run(ctx context.Context, cfg *config.LoadTestConfig, rngConfig config.RandomNumberGenerator) error {
	s.logger.Info("Starting user simulation",
		zap.String("user_id", s.userID),
		zap.String("user_name", s.userName),
		zap.String("org_id", s.orgID),
	)

	// Determine number of questions for this session
	numQuestions := cfg.Spec.UserBehavior.QuestionsPerSession.GetValue(rngConfig)

	// Generate question strategy (pick from test types if configured)
	questionStrategy := s.selectQuestionStrategy(cfg)

	// Create question generator with user-specific seed
	qGen := questions.NewGenerator(s.rng.Int63())

	// Track conversation history for multi-turn conversations
	var conversationHistory []map[string]string

	// Execute questions
	for i := 0; i < numQuestions; i++ {
		select {
		case <-ctx.Done():
			s.logger.Info("User simulation canceled", zap.String("user_id", s.userID))
			return ctx.Err()
		default:
		}

		// Generate question
		question := qGen.GetRandomQuestion(questionStrategy)

		// Decide if this should be a multi-turn conversation
		isMultiTurn := s.shouldStartMultiTurn(cfg, i, numQuestions)

		if isMultiTurn {
			conversationHistory = append(conversationHistory, map[string]string{
				"role":    "user",
				"content": question,
			})
		} else {
			// Single-turn: reset conversation
			conversationHistory = []map[string]string{
				{"role": "user", "content": question},
			}
		}

		// Send request
		model := s.selectModel(cfg, question)
		streaming := s.shouldUseStreaming(cfg)

		response, err := s.sendChatCompletionRequest(ctx, model, conversationHistory, streaming)
		if err != nil {
			s.logger.Error("Request failed",
				zap.String("user_id", s.userID),
				zap.Error(err),
			)
			s.metrics.RecordRequest(s.userID, s.orgID, model, 0, 0, false, "request_error")

			// Handle error based on config
			if s.shouldStopOnError(cfg, err) {
				return err
			}
			continue
		}

		// Add assistant response to conversation history if multi-turn
		if isMultiTurn && response.AssistantMessage != "" {
			conversationHistory = append(conversationHistory, map[string]string{
				"role":    "assistant",
				"content": response.AssistantMessage,
			})
		}

		// Record metrics
		s.metrics.RecordRequest(s.userID, s.orgID, model, response.Latency, response.TotalTokens, true, "")

		if response.TTFT > 0 && response.TPS > 0 {
			s.metrics.RecordLLMMetrics(s.userID, model, response.TTFT, response.TPS)
		}

		// Think time before next question (except for last question)
		if i < numQuestions-1 {
			thinkTime := cfg.Spec.UserBehavior.ThinkTimeSeconds.GetThinkTime(rngConfig)
			s.logger.Debug("Think time",
				zap.String("user_id", s.userID),
				zap.Duration("duration", thinkTime),
			)

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(thinkTime):
			}
		}
	}

	s.logger.Info("User simulation complete",
		zap.String("user_id", s.userID),
		zap.Int("questions_completed", numQuestions),
	)

	return nil
}

// ChatCompletionResponse represents the response from a chat completion request
type ChatCompletionResponse struct {
	AssistantMessage string
	Latency          time.Duration
	TTFT             time.Duration // Time to first token
	TPS              float64       // Tokens per second
	TotalTokens      int
	PromptTokens     int
	CompletionTokens int
}

// sendChatCompletionRequest sends a chat completion request to the API
func (s *UserSimulator) sendChatCompletionRequest(
	ctx context.Context,
	model string,
	messages []map[string]string,
	streaming bool,
) (*ChatCompletionResponse, error) {
	url := s.apiRouterURL + "/v1/chat/completions"

	payload := map[string]interface{}{
		"model":    model,
		"messages": messages,
		"stream":   streaming,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.apiKey)

	startTime := time.Now()
	var firstTokenTime time.Time

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	latency := time.Since(startTime)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	if streaming {
		// Handle streaming response
		return s.handleStreamingResponse(resp.Body, latency, startTime)
	}

	// Handle non-streaming response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var apiResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	assistantMessage := ""
	if len(apiResp.Choices) > 0 {
		assistantMessage = apiResp.Choices[0].Message.Content
		// Approximate TTFT for non-streaming (assume immediate first token)
		firstTokenTime = startTime.Add(latency / 2)
	}

	// Calculate TPS
	tps := 0.0
	if apiResp.Usage.CompletionTokens > 0 && latency > 0 {
		tps = float64(apiResp.Usage.CompletionTokens) / latency.Seconds()
	}

	return &ChatCompletionResponse{
		AssistantMessage: assistantMessage,
		Latency:          latency,
		TTFT:             firstTokenTime.Sub(startTime),
		TPS:              tps,
		TotalTokens:      apiResp.Usage.TotalTokens,
		PromptTokens:     apiResp.Usage.PromptTokens,
		CompletionTokens: apiResp.Usage.CompletionTokens,
	}, nil
}

// handleStreamingResponse processes a streaming API response
func (s *UserSimulator) handleStreamingResponse(body io.Reader, totalLatency time.Duration, startTime time.Time) (*ChatCompletionResponse, error) {
	var firstTokenTime time.Time
	var content string
	tokenCount := 0

	decoder := json.NewDecoder(body)

	for {
		var chunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
		}

		if err := decoder.Decode(&chunk); err == io.EOF {
			break
		} else if err != nil {
			// Some streaming responses use SSE format, not JSON lines
			// For simplicity, we'll just count this as one token received
			if firstTokenTime.IsZero() {
				firstTokenTime = time.Now()
			}
			tokenCount++
			break
		}

		if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
			if firstTokenTime.IsZero() {
				firstTokenTime = time.Now()
			}
			content += chunk.Choices[0].Delta.Content
			tokenCount++
		}
	}

	ttft := time.Duration(0)
	if !firstTokenTime.IsZero() {
		ttft = firstTokenTime.Sub(startTime)
	}

	// Calculate TPS
	tps := 0.0
	if tokenCount > 0 && totalLatency > 0 {
		tps = float64(tokenCount) / totalLatency.Seconds()
	}

	return &ChatCompletionResponse{
		AssistantMessage: content,
		Latency:          totalLatency,
		TTFT:             ttft,
		TPS:              tps,
		CompletionTokens: tokenCount,
		TotalTokens:      tokenCount, // Approximate
	}, nil
}

// selectQuestionStrategy selects a question strategy based on test configuration
func (s *UserSimulator) selectQuestionStrategy(cfg *config.LoadTestConfig) questions.Strategy {
	if len(cfg.Spec.TestTypes) == 0 {
		return questions.StrategyMixed
	}

	// Weighted random selection
	totalWeight := 0
	for _, tt := range cfg.Spec.TestTypes {
		totalWeight += tt.Weight
	}

	r := s.rng.Intn(totalWeight)
	cumulative := 0

	for _, tt := range cfg.Spec.TestTypes {
		cumulative += tt.Weight
		if r < cumulative {
			// Map test type strategy to question strategy
			return mapTestTypeToQuestionStrategy(tt.QuestionStrategy)
		}
	}

	return questions.StrategyMixed
}

// mapTestTypeToQuestionStrategy maps test type strategy string to question.Strategy
func mapTestTypeToQuestionStrategy(strategy string) questions.Strategy {
	switch strategy {
	case "historical":
		return questions.StrategyHistorical
	case "mathematical":
		return questions.StrategyMathematical
	case "geographical":
		return questions.StrategyGeographical
	case "hypothetical":
		return questions.StrategyHypothetical
	case "technical":
		return questions.StrategyTechnical
	case "mixed":
		return questions.StrategyMixed
	default:
		return questions.StrategyMixed
	}
}

// selectModel selects which model to use for a request
func (s *UserSimulator) selectModel(cfg *config.LoadTestConfig, question string) string {
	// Default model
	defaultModel := "gpt-4o"

	// If no test types configured, use default
	if len(cfg.Spec.TestTypes) == 0 {
		return defaultModel
	}

	// Simple heuristic: short questions (<100 chars) go to SLM, longer to medium
	questionLen := len(question)

	for _, tt := range cfg.Spec.TestTypes {
		if tt.ModelTargeting.SLMModels != nil && questionLen < 100 {
			if len(tt.ModelTargeting.SLMModels) > 0 {
				return tt.ModelTargeting.SLMModels[0]
			}
		}
		if tt.ModelTargeting.MediumModels != nil && questionLen >= 100 {
			if len(tt.ModelTargeting.MediumModels) > 0 {
				return tt.ModelTargeting.MediumModels[0]
			}
		}
	}

	return defaultModel
}

// shouldUseStreaming determines if streaming should be used
func (s *UserSimulator) shouldUseStreaming(cfg *config.LoadTestConfig) bool {
	if len(cfg.Spec.TestTypes) == 0 {
		return false
	}

	// Check if any test type has streaming enabled
	for _, tt := range cfg.Spec.TestTypes {
		if tt.StreamingResponse {
			return true
		}
	}

	return false
}

// shouldStartMultiTurn determines if this question should be part of a multi-turn conversation
func (s *UserSimulator) shouldStartMultiTurn(cfg *config.LoadTestConfig, currentQuestion, totalQuestions int) bool {
	// Don't start multi-turn on last question
	if currentQuestion >= totalQuestions-1 {
		return false
	}

	// Check probability
	prob := cfg.Spec.UserBehavior.ConversationStyle.MultiTurnProbability
	return s.rng.Float64() < prob
}

// shouldStopOnError determines if simulation should stop on error
func (s *UserSimulator) shouldStopOnError(cfg *config.LoadTestConfig, err error) bool {
	// For now, continue on errors unless it's a context cancellation
	return err == context.Canceled || err == context.DeadlineExceeded
}
