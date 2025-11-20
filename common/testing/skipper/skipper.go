// The MIT License
//
// Copyright (c) 2025 Temporal Technologies Inc.  All rights reserved.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package skipper

import (
	"math/rand"
	"time"

	"go.temporal.io/server/common/log"
	"go.temporal.io/server/common/log/tag"
	"go.temporal.io/server/common/testing/play"
)

// Strategy defines how the skipper selects plays to execute
type Strategy string

const (
	// StrategyRandom selects plays randomly
	StrategyRandom Strategy = "random"

	// StrategyCoverageDriven prioritizes uncovered scenarios
	StrategyCoverageDriven Strategy = "coverage"

	// StrategySequential executes plays in order
	StrategySequential Strategy = "sequential"

	// StrategyWeighted uses weighted random selection based on tags
	StrategyWeighted Strategy = "weighted"
)

// Skipper intelligently selects and orchestrates play execution
type Skipper struct {
	logger   log.Logger
	library  *play.PlayLibrary
	strategy Strategy
	rng      *rand.Rand

	// Coverage tracking
	executedPlays map[string]int // play name -> execution count
	coverage      map[string]int // tag -> coverage count

	// Weights for different categories (used in weighted strategy)
	categoryWeights map[string]float64
}

// Config configures the skipper
type Config struct {
	// Strategy for play selection
	Strategy Strategy

	// Seed for random number generation (0 = use current time)
	Seed int64

	// CategoryWeights assigns importance to different play categories
	// Higher weight = more likely to be selected in weighted strategy
	CategoryWeights map[string]float64

	// MaxExecutionsPerPlay limits how many times a single play can run
	MaxExecutionsPerPlay int
}

// DefaultConfig returns default skipper configuration
func DefaultConfig() Config {
	return Config{
		Strategy: StrategyRandom,
		Seed:     0,
		CategoryWeights: map[string]float64{
			"resilience": 2.0,
			"baseline":   1.0,
			"performance": 1.5,
			"complex":    1.0,
		},
		MaxExecutionsPerPlay: 3,
	}
}

// NewSkipper creates a new skipper
func NewSkipper(logger log.Logger, library *play.PlayLibrary, config Config) *Skipper {
	seed := config.Seed
	if seed == 0 {
		seed = time.Now().UnixNano()
	}

	return &Skipper{
		logger:          logger,
		library:         library,
		strategy:        config.Strategy,
		rng:             rand.New(rand.NewSource(seed)),
		executedPlays:   make(map[string]int),
		coverage:        make(map[string]int),
		categoryWeights: config.CategoryWeights,
	}
}

// SelectPlay chooses the next play to execute based on the configured strategy
func (s *Skipper) SelectPlay() *play.Play {
	switch s.strategy {
	case StrategyRandom:
		return s.selectRandom()
	case StrategyCoverageDriven:
		return s.selectCoverageDriven()
	case StrategySequential:
		return s.selectSequential()
	case StrategyWeighted:
		return s.selectWeighted()
	default:
		return s.selectRandom()
	}
}

// selectRandom chooses a play uniformly at random
func (s *Skipper) selectRandom() *play.Play {
	allPlays := s.library.All()
	if len(allPlays) == 0 {
		return nil
	}

	idx := s.rng.Intn(len(allPlays))
	selectedPlay := allPlays[idx]

	s.logger.Info("Selected play (random)", tag.NewStringTag("play", selectedPlay.Name))
	return selectedPlay
}

// selectCoverageDriven prioritizes plays that cover less-tested scenarios
func (s *Skipper) selectCoverageDriven() *play.Play {
	allPlays := s.library.All()
	if len(allPlays) == 0 {
		return nil
	}

	// Find plays with lowest execution count
	minExecutions := -1
	var candidates []*play.Play

	for _, p := range allPlays {
		execCount := s.executedPlays[p.Name]

		if minExecutions == -1 || execCount < minExecutions {
			minExecutions = execCount
			candidates = []*play.Play{p}
		} else if execCount == minExecutions {
			candidates = append(candidates, p)
		}
	}

	// Among candidates with same execution count, prioritize by tag coverage
	// (tags that have been tested less)
	var bestPlay *play.Play
	lowestTagCoverage := -1

	for _, p := range candidates {
		tagCoverage := s.getPlayTagCoverage(p)
		if lowestTagCoverage == -1 || tagCoverage < lowestTagCoverage {
			lowestTagCoverage = tagCoverage
			bestPlay = p
		}
	}

	if bestPlay == nil && len(candidates) > 0 {
		bestPlay = candidates[0]
	}

	s.logger.Info("Selected play (coverage-driven)",
		tag.NewStringTag("play", bestPlay.Name),
		tag.NewInt("executions", s.executedPlays[bestPlay.Name]),
		tag.NewInt("tagCoverage", lowestTagCoverage))

	return bestPlay
}

// getPlayTagCoverage returns the total coverage count for all tags in a play
func (s *Skipper) getPlayTagCoverage(p *play.Play) int {
	total := 0
	for _, tag := range p.Tags {
		total += s.coverage[tag]
	}
	return total
}

// selectSequential chooses plays in order
func (s *Skipper) selectSequential() *play.Play {
	allPlays := s.library.All()
	if len(allPlays) == 0 {
		return nil
	}

	// Find first play that hasn't been executed yet, or cycle through
	for _, p := range allPlays {
		if s.executedPlays[p.Name] == 0 {
			s.logger.Info("Selected play (sequential)", tag.NewStringTag("play", p.Name))
			return p
		}
	}

	// All have been executed at least once, start over
	selectedPlay := allPlays[0]
	s.logger.Info("Selected play (sequential - cycle)", tag.NewStringTag("play", selectedPlay.Name))
	return selectedPlay
}

// selectWeighted uses weighted random selection based on category importance
func (s *Skipper) selectWeighted() *play.Play {
	allPlays := s.library.All()
	if len(allPlays) == 0 {
		return nil
	}

	// Calculate weights for each play
	type weightedPlay struct {
		play   *play.Play
		weight float64
	}

	var weightedPlays []weightedPlay
	totalWeight := 0.0

	for _, p := range allPlays {
		// Base weight from category tags
		weight := 1.0
		for _, tag := range p.Tags {
			if w, ok := s.categoryWeights[tag]; ok {
				weight *= w
			}
		}

		// Reduce weight for frequently executed plays
		execCount := float64(s.executedPlays[p.Name])
		if execCount > 0 {
			weight /= (execCount + 1)
		}

		weightedPlays = append(weightedPlays, weightedPlay{play: p, weight: weight})
		totalWeight += weight
	}

	// Select based on weighted probability
	target := s.rng.Float64() * totalWeight
	cumulative := 0.0

	for _, wp := range weightedPlays {
		cumulative += wp.weight
		if cumulative >= target {
			s.logger.Info("Selected play (weighted)",
				tag.NewStringTag("play", wp.play.Name),
				tag.NewFloat64("weight", wp.weight),
				tag.NewFloat64("totalWeight", totalWeight))
			return wp.play
		}
	}

	// Fallback (shouldn't reach here)
	return weightedPlays[len(weightedPlays)-1].play
}

// RecordExecution updates coverage tracking after a play executes
func (s *Skipper) RecordExecution(p *play.Play, success bool) {
	s.executedPlays[p.Name]++

	if success {
		for _, tag := range p.Tags {
			s.coverage[tag]++
		}
	}

	s.logger.Info("Recorded play execution",
		tag.NewStringTag("play", p.Name),
		tag.NewBoolTag("success", success),
		tag.NewInt("totalExecutions", s.executedPlays[p.Name]))
}

// GetCoverageReport returns current coverage statistics
func (s *Skipper) GetCoverageReport() CoverageReport {
	return CoverageReport{
		ExecutedPlays:     s.executedPlays,
		TagCoverage:       s.coverage,
		TotalPlaysInLib:   len(s.library.All()),
		UniquePlaysRun:    len(s.executedPlays),
		TotalExecutions:   s.getTotalExecutions(),
	}
}

func (s *Skipper) getTotalExecutions() int {
	total := 0
	for _, count := range s.executedPlays {
		total += count
	}
	return total
}

// CoverageReport summarizes test coverage
type CoverageReport struct {
	// ExecutedPlays maps play name to execution count
	ExecutedPlays map[string]int

	// TagCoverage maps tag to coverage count
	TagCoverage map[string]int

	// TotalPlaysInLib is the total number of plays available
	TotalPlaysInLib int

	// UniquePlaysRun is how many different plays have been executed
	UniquePlaysRun int

	// TotalExecutions is the sum of all play executions
	TotalExecutions int
}

// CoveragePercentage returns the percentage of plays that have been executed at least once
func (cr *CoverageReport) CoveragePercentage() float64 {
	if cr.TotalPlaysInLib == 0 {
		return 0
	}
	return float64(cr.UniquePlaysRun) / float64(cr.TotalPlaysInLib) * 100.0
}
