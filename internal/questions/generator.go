package questions

import (
	"crypto/md5"
	"fmt"
	"math/rand"
)

// Strategy defines the type of questions to generate
type Strategy string

const (
	StrategyHistorical   Strategy = "historical"
	StrategyMathematical Strategy = "mathematical"
	StrategyGeographical Strategy = "geographical"
	StrategyHypothetical Strategy = "hypothetical"
	StrategyTechnical    Strategy = "technical"
	StrategyMixed        Strategy = "mixed"
)

// Generator generates unique questions using various strategies to avoid cache hits
type Generator struct {
	rng *rand.Rand
}

// NewGenerator creates a new question generator with the given seed
func NewGenerator(seed int64) *Generator {
	return &Generator{
		rng: rand.New(rand.NewSource(seed)),
	}
}

// Generate generates questions based on the selected strategy
func (g *Generator) Generate(strategy Strategy) []string {
	switch strategy {
	case StrategyHistorical:
		return g.historicalQuestions()
	case StrategyMathematical:
		return g.mathematicalQuestions()
	case StrategyGeographical:
		return g.geographicalQuestions()
	case StrategyHypothetical:
		return g.hypotheticalQuestions()
	case StrategyTechnical:
		return g.technicalQuestions()
	case StrategyMixed:
		return g.mixedQuestions()
	default:
		return g.mixedQuestions()
	}
}

// historicalQuestions generates questions about random historical years
func (g *Generator) historicalQuestions() []string {
	year := g.rng.Intn(1024) + 1000 // 1000-2023

	return []string{
		fmt.Sprintf("What major events occurred in the year %d?", year),
		fmt.Sprintf("Who were the influential leaders during %d?", year),
		fmt.Sprintf("What was the state of technology in %d?", year),
		fmt.Sprintf("Describe the political climate of %d.", year),
		fmt.Sprintf("What were the major conflicts or peace treaties around %d?", year),
		fmt.Sprintf("How did people communicate in %d?", year),
		fmt.Sprintf("What was daily life like for common people in %d?", year),
		fmt.Sprintf("What scientific discoveries were made around %d?", year),
	}
}

// mathematicalQuestions generates unique math problems
func (g *Generator) mathematicalQuestions() []string {
	a := g.rng.Intn(9900) + 100  // 100-9999
	b := g.rng.Intn(9900) + 100  // 100-9999
	c := g.rng.Intn(49) + 2      // 2-50

	return []string{
		fmt.Sprintf("What is %d multiplied by %d?", a, b),
		fmt.Sprintf("If you divide the previous result by %d, what do you get?", c),
		fmt.Sprintf("What are the prime factors of %d?", a),
		fmt.Sprintf("Is %d a perfect square? If not, what's the nearest one?", b),
		fmt.Sprintf("Calculate %d to the power of %d.", a, c%5+2),
		fmt.Sprintf("What is the greatest common divisor of %d and %d?", a, b),
		fmt.Sprintf("Convert %d to base-%d notation.", a, c%7+2),
		fmt.Sprintf("How many ways can you partition %d into positive integers?", c),
	}
}

// geographicalQuestions generates geography questions with random parameters
func (g *Generator) geographicalQuestions() []string {
	cities := []string{
		"Tokyo", "Delhi", "Shanghai", "São Paulo", "Mexico City", "Cairo",
		"Mumbai", "Beijing", "Dhaka", "Osaka", "Karachi", "Istanbul",
		"Buenos Aires", "Kolkata", "Lagos", "Manila", "Tianjin", "Rio",
	}

	countries := []string{
		"Brazil", "Russia", "India", "China", "South Africa", "Mexico",
		"Indonesia", "Turkey", "Saudi Arabia", "Argentina", "Egypt",
		"Nigeria", "Japan", "Germany", "France", "Italy", "Canada",
	}

	city := cities[g.rng.Intn(len(cities))]
	country := countries[g.rng.Intn(len(countries))]
	distance := g.rng.Intn(4501) + 500 // 500-5000

	return []string{
		fmt.Sprintf("What is the population of %s?", city),
		fmt.Sprintf("What are the neighboring countries of %s?", country),
		fmt.Sprintf("What is the main river flowing through or near %s?", city),
		fmt.Sprintf("If you travel %dkm east from %s, where might you be?", distance, city),
		fmt.Sprintf("What is the climate type in %s?", country),
		fmt.Sprintf("What are the major exports of %s?", country),
		fmt.Sprintf("What language is primarily spoken in %s?", city),
		fmt.Sprintf("What is the time zone of %s?", city),
	}
}

// hypotheticalQuestions generates creative hypothetical scenarios
func (g *Generator) hypotheticalQuestions() []string {
	objects := []string{"cars", "trees", "buildings", "phones", "books", "computers"}
	properties := []string{"invisible", "magnetic", "telepathic", "indestructible", "sentient"}
	numbers := []int{10, 50, 100, 1000, 10000}

	obj := objects[g.rng.Intn(len(objects))]
	prop := properties[g.rng.Intn(len(properties))]
	num := numbers[g.rng.Intn(len(numbers))]
	percent := g.rng.Intn(81) + 10 // 10-90

	return []string{
		fmt.Sprintf("What would happen if all %s suddenly became %s?", obj, prop),
		fmt.Sprintf("How would society change if %d%% of people could read minds?", percent),
		fmt.Sprintf("Design a city that could accommodate %d million people in 1 square km.", num),
		fmt.Sprintf("What if gravity was %d%% stronger?", percent),
		"How would communication work if sound didn't exist?",
		fmt.Sprintf("What if %s could only last for 24 hours before disappearing?", obj),
		fmt.Sprintf("Describe an economy where %s are the primary currency.", obj),
		"What safety measures would we need if everyone could fly?",
	}
}

// technicalQuestions generates technical/programming questions
func (g *Generator) technicalQuestions() []string {
	arraySize := g.rng.Intn(991) + 10 // 10-1000
	complexities := []string{"O(n)", "O(n²)", "O(log n)", "O(n log n)"}
	complexity := complexities[g.rng.Intn(len(complexities))]
	port := g.rng.Intn(7000) + 3000 // 3000-9999

	// Generate hash similar to Python's md5(str(seed).encode()).hexdigest()[:8]
	hashInput := fmt.Sprintf("%d", g.rng.Int63())
	hash := md5.Sum([]byte(hashInput))
	hashStr := fmt.Sprintf("%x", hash)[:8]

	_ = complexity // Will be used in future enhancements

	return []string{
		fmt.Sprintf("What's the best sorting algorithm for %d nearly-sorted integers?", arraySize),
		fmt.Sprintf("How would you design a cache for %d frequently accessed items?", arraySize),
		fmt.Sprintf("Explain the trade-offs of using a hash table with %d buckets.", arraySize),
		fmt.Sprintf("What happens when you try to connect to port %d?", port),
		fmt.Sprintf("Design a URL shortener that generates codes like '%s'.", hashStr),
		fmt.Sprintf("How would you find duplicates in an array of %d elements?", arraySize),
		fmt.Sprintf("What's the space complexity of storing %d items in a binary tree?", arraySize),
		fmt.Sprintf("How would you implement rate limiting for %d requests per second?", arraySize),
	}
}

// mixedQuestions mixes different types of questions
func (g *Generator) mixedQuestions() []string {
	questions := []string{}

	strategies := []Strategy{
		StrategyHistorical,
		StrategyMathematical,
		StrategyGeographical,
		StrategyHypothetical,
		StrategyTechnical,
	}

	// Take 2 questions from each category
	for _, strategy := range strategies {
		// Create a new generator with a different seed for each strategy
		seed := g.rng.Int63()
		strategyGen := NewGenerator(seed)
		strategyQuestions := strategyGen.Generate(strategy)

		// Sample 2 random questions
		if len(strategyQuestions) >= 2 {
			// Simple random sample without replacement
			indices := g.rng.Perm(len(strategyQuestions))
			questions = append(questions, strategyQuestions[indices[0]])
			questions = append(questions, strategyQuestions[indices[1]])
		}
	}

	// Return first 8 questions
	if len(questions) > 8 {
		return questions[:8]
	}
	return questions
}

// GetQuestion returns a single question from the generated set
func (g *Generator) GetQuestion(strategy Strategy, index int) string {
	questions := g.Generate(strategy)
	if index >= 0 && index < len(questions) {
		return questions[index]
	}
	// Return random question if index out of bounds
	return questions[g.rng.Intn(len(questions))]
}

// GetRandomQuestion returns a random question from the strategy
func (g *Generator) GetRandomQuestion(strategy Strategy) string {
	questions := g.Generate(strategy)
	return questions[g.rng.Intn(len(questions))]
}
