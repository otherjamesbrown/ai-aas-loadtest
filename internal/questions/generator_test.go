package questions

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGenerator(t *testing.T) {
	gen := NewGenerator(42)
	require.NotNil(t, gen)
	require.NotNil(t, gen.rng)
}

func TestHistoricalQuestions(t *testing.T) {
	gen := NewGenerator(42)
	questions := gen.historicalQuestions()

	assert.Len(t, questions, 8)

	// Check that questions contain year references
	hasYear := false
	for _, q := range questions {
		if strings.Contains(q, "year") || strings.Contains(q, "in ") {
			hasYear = true
			break
		}
	}
	assert.True(t, hasYear, "Historical questions should reference years")
}

func TestMathematicalQuestions(t *testing.T) {
	gen := NewGenerator(42)
	questions := gen.mathematicalQuestions()

	assert.Len(t, questions, 8)

	// Check that questions contain numbers
	hasNumbers := false
	for _, q := range questions {
		// Look for digit patterns
		for _, char := range q {
			if char >= '0' && char <= '9' {
				hasNumbers = true
				break
			}
		}
		if hasNumbers {
			break
		}
	}
	assert.True(t, hasNumbers, "Mathematical questions should contain numbers")
}

func TestGeographicalQuestions(t *testing.T) {
	gen := NewGenerator(42)
	questions := gen.geographicalQuestions()

	assert.Len(t, questions, 8)

	// Should have diverse question types
	assert.Contains(t, strings.Join(questions, " "), "population")
}

func TestHypotheticalQuestions(t *testing.T) {
	gen := NewGenerator(42)
	questions := gen.hypotheticalQuestions()

	assert.Len(t, questions, 8)

	// Hypothetical questions should be creative
	hasWhatIf := false
	for _, q := range questions {
		if strings.Contains(q, "What if") || strings.Contains(q, "would") || strings.Contains(q, "How") {
			hasWhatIf = true
			break
		}
	}
	assert.True(t, hasWhatIf, "Hypothetical questions should contain 'what if' style queries")
}

func TestTechnicalQuestions(t *testing.T) {
	gen := NewGenerator(42)
	questions := gen.technicalQuestions()

	assert.Len(t, questions, 8)

	// Should contain technical terms
	allQuestions := strings.Join(questions, " ")
	hasTechnical := strings.Contains(allQuestions, "algorithm") ||
		strings.Contains(allQuestions, "cache") ||
		strings.Contains(allQuestions, "complexity") ||
		strings.Contains(allQuestions, "array")
	assert.True(t, hasTechnical, "Technical questions should contain technical terms")
}

func TestMixedQuestions(t *testing.T) {
	gen := NewGenerator(42)
	questions := gen.mixedQuestions()

	// Mixed should return 8 questions (2 from each of 5 categories, truncated to 8)
	assert.LessOrEqual(t, len(questions), 10)
	assert.GreaterOrEqual(t, len(questions), 8)

	// Questions should be diverse
	assert.NotEmpty(t, questions)
}

func TestGenerate_AllStrategies(t *testing.T) {
	strategies := []Strategy{
		StrategyHistorical,
		StrategyMathematical,
		StrategyGeographical,
		StrategyHypothetical,
		StrategyTechnical,
		StrategyMixed,
	}

	for _, strategy := range strategies {
		t.Run(string(strategy), func(t *testing.T) {
			gen := NewGenerator(42)
			questions := gen.Generate(strategy)

			assert.NotEmpty(t, questions, "Strategy %s should generate questions", strategy)
			assert.GreaterOrEqual(t, len(questions), 8, "Should generate at least 8 questions")

			// All questions should be non-empty
			for i, q := range questions {
				assert.NotEmpty(t, q, "Question %d should not be empty", i)
			}
		})
	}
}

func TestGenerate_InvalidStrategy(t *testing.T) {
	gen := NewGenerator(42)
	questions := gen.Generate("invalid")

	// Should default to mixed strategy
	assert.NotEmpty(t, questions)
	assert.GreaterOrEqual(t, len(questions), 8)
}

func TestGetQuestion(t *testing.T) {
	gen := NewGenerator(42)

	// Test valid index
	question := gen.GetQuestion(StrategyHistorical, 0)
	assert.NotEmpty(t, question)

	// Test another valid index
	question2 := gen.GetQuestion(StrategyHistorical, 1)
	assert.NotEmpty(t, question2)
	assert.NotEqual(t, question, question2, "Different indices should return different questions")

	// Test out of bounds index (should return random question)
	question3 := gen.GetQuestion(StrategyHistorical, 999)
	assert.NotEmpty(t, question3)
}

func TestGetRandomQuestion(t *testing.T) {
	gen := NewGenerator(42)
	question := gen.GetRandomQuestion(StrategyMathematical)

	assert.NotEmpty(t, question)
}

func TestUniqueness(t *testing.T) {
	// Test that different seeds produce different questions
	gen1 := NewGenerator(42)
	gen2 := NewGenerator(43)

	questions1 := gen1.Generate(StrategyHistorical)
	questions2 := gen2.Generate(StrategyHistorical)

	assert.Len(t, questions1, 8)
	assert.Len(t, questions2, 8)

	// At least some questions should be different
	differentCount := 0
	for i := 0; i < len(questions1) && i < len(questions2); i++ {
		if questions1[i] != questions2[i] {
			differentCount++
		}
	}

	assert.Greater(t, differentCount, 0, "Different seeds should produce different questions")
}

func TestDeterminism(t *testing.T) {
	// Test that same seed produces same questions
	gen1 := NewGenerator(42)
	gen2 := NewGenerator(42)

	questions1 := gen1.Generate(StrategyMathematical)
	questions2 := gen2.Generate(StrategyMathematical)

	assert.Equal(t, questions1, questions2, "Same seed should produce identical questions")
}

func TestQuestionFormat(t *testing.T) {
	gen := NewGenerator(42)

	strategies := []Strategy{
		StrategyHistorical,
		StrategyMathematical,
		StrategyGeographical,
		StrategyHypothetical,
		StrategyTechnical,
	}

	for _, strategy := range strategies {
		questions := gen.Generate(strategy)
		for _, q := range questions {
			// Each question should end with ? or be a statement
			assert.NotEmpty(t, q, "Question should not be empty")
			assert.Greater(t, len(q), 10, "Question should be substantial")

			// Should start with capital letter or digit
			firstChar := rune(q[0])
			assert.True(t,
				(firstChar >= 'A' && firstChar <= 'Z') ||
					(firstChar >= '0' && firstChar <= '9') ||
					firstChar == 'W' || firstChar == 'H' || firstChar == 'I' || firstChar == 'D' || firstChar == 'C',
				"Question should start with capital letter: %s", q)
		}
	}
}

func TestLargeScaleUniqueness(t *testing.T) {
	// Test uniqueness for mathematical and historical strategies which have highest variability
	questionSet := make(map[string]bool)

	// Mathematical questions have very high uniqueness due to random numbers
	for i := 0; i < 500; i++ {
		gen := NewGenerator(int64(i * 1337))
		questions := gen.Generate(StrategyMathematical)
		for _, q := range questions {
			questionSet[q] = true
		}
	}

	mathQuestions := len(questionSet)
	questionSet = make(map[string]bool) // Reset

	// Historical questions also have high uniqueness (1000+ years to choose from)
	for i := 0; i < 500; i++ {
		gen := NewGenerator(int64(i * 1337))
		questions := gen.Generate(StrategyHistorical)
		for _, q := range questions {
			questionSet[q] = true
		}
	}

	historicalQuestions := len(questionSet)

	// With 500 iterations * 8 questions each = 4000 questions per strategy
	// Each seed generates 8 questions sharing same random parameters
	// So we expect roughly 500 unique parameter sets * some variation = ~70-80% uniqueness
	mathRatio := float64(mathQuestions) / 4000.0
	assert.Greater(t, mathRatio, 0.70, "Mathematical questions should have >70%% uniqueness, got %.2f%%", mathRatio*100)

	// Historical should have similar uniqueness
	histRatio := float64(historicalQuestions) / 4000.0
	assert.Greater(t, histRatio, 0.70, "Historical questions should have >70%% uniqueness, got %.2f%%", histRatio*100)
}
