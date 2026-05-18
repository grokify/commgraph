package analysis

import (
	"context"
	"math"
	"sort"
	"time"

	"github.com/grokify/commgraph/entity"
	"github.com/grokify/commgraph/storage"
)

// TimeWindow represents a time period for temporal analysis.
type TimeWindow struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// ActivityPoint represents activity at a point in time.
type ActivityPoint struct {
	Time        time.Time `json:"time"`
	Count       int       `json:"count"`
	UniqueFrom  int       `json:"unique_from"`
	UniqueTo    int       `json:"unique_to"`
	TotalWeight float64   `json:"total_weight"`
}

// Burst represents a detected burst of activity.
type Burst struct {
	Start     time.Time `json:"start"`
	End       time.Time `json:"end"`
	Peak      time.Time `json:"peak"`
	PeakCount int       `json:"peak_count"`
	Total     int       `json:"total"`
	ZScore    float64   `json:"z_score"` // How many std devs above mean
}

// TemporalResults contains the results of temporal analysis.
type TemporalResults struct {
	Window       TimeWindow      `json:"window"`
	Timeline     []ActivityPoint `json:"timeline"`
	Bursts       []Burst         `json:"bursts"`
	TotalCount   int             `json:"total_count"`
	DailyAverage float64         `json:"daily_average"`
	DailyStdDev  float64         `json:"daily_std_dev"`
}

// ActorActivityResults contains activity metrics for actors over time.
type ActorActivityResults struct {
	Window  TimeWindow                  `json:"window"`
	Actors  map[entity.ActorID][]ActivityPoint `json:"actors"`
	TopSenders []ActorActivity            `json:"top_senders"`
	TopReceivers []ActorActivity          `json:"top_receivers"`
}

// ActorActivity represents an actor's activity metrics.
type ActorActivity struct {
	ActorID     entity.ActorID `json:"actor_id"`
	DisplayName string         `json:"display_name"`
	SentCount   int            `json:"sent_count"`
	RecvCount   int            `json:"recv_count"`
	FirstActive time.Time      `json:"first_active"`
	LastActive  time.Time      `json:"last_active"`
}

// Timeline builds a timeline of activity over a time range.
func (a *Analyzer) Timeline(ctx context.Context, window TimeWindow, granularity time.Duration) (*TemporalResults, error) {
	// Get interactions in the time window
	interactions, err := a.store.GetInteractions(ctx, storage.InteractionQuery{
		After:  window.Start,
		Before: window.End,
	})
	if err != nil {
		return nil, err
	}

	// Group by time bucket
	buckets := make(map[time.Time]*ActivityPoint)

	for _, interaction := range interactions {
		// Truncate to bucket
		bucket := interaction.Timestamp.Truncate(granularity)

		if buckets[bucket] == nil {
			buckets[bucket] = &ActivityPoint{
				Time: bucket,
			}
		}

		point := buckets[bucket]
		point.Count++
		point.TotalWeight += a.profile.Weight(interaction.EdgeType)
	}

	// Convert to sorted slice
	timeline := make([]ActivityPoint, 0, len(buckets))
	for _, point := range buckets {
		timeline = append(timeline, *point)
	}
	sort.Slice(timeline, func(i, j int) bool {
		return timeline[i].Time.Before(timeline[j].Time)
	})

	// Calculate stats
	totalCount := 0
	counts := make([]float64, len(timeline))
	for i, point := range timeline {
		totalCount += point.Count
		counts[i] = float64(point.Count)
	}

	mean, stdDev := meanStdDev(counts)

	// Detect bursts (periods significantly above average)
	bursts := detectBursts(timeline, mean, stdDev, 2.0) // 2 std devs threshold

	return &TemporalResults{
		Window:       window,
		Timeline:     timeline,
		Bursts:       bursts,
		TotalCount:   totalCount,
		DailyAverage: mean,
		DailyStdDev:  stdDev,
	}, nil
}

// BurstDetection identifies periods of unusually high activity.
func (a *Analyzer) BurstDetection(ctx context.Context, threshold float64, minDuration time.Duration) ([]Burst, error) {
	// Get all interactions
	interactions, err := a.store.GetInteractions(ctx, storage.InteractionQuery{})
	if err != nil {
		return nil, err
	}

	if len(interactions) == 0 {
		return nil, nil
	}

	// Find time range
	minTime := interactions[0].Timestamp
	maxTime := interactions[0].Timestamp
	for _, i := range interactions {
		if i.Timestamp.Before(minTime) {
			minTime = i.Timestamp
		}
		if i.Timestamp.After(maxTime) {
			maxTime = i.Timestamp
		}
	}

	// Build daily timeline
	window := TimeWindow{Start: minTime, End: maxTime}
	results, err := a.Timeline(ctx, window, 24*time.Hour)
	if err != nil {
		return nil, err
	}

	// Use the threshold parameter instead of fixed value
	return detectBursts(results.Timeline, results.DailyAverage, results.DailyStdDev, threshold), nil
}

// detectBursts finds periods where activity exceeds threshold std devs above mean.
func detectBursts(timeline []ActivityPoint, mean, stdDev, threshold float64) []Burst {
	if len(timeline) == 0 || stdDev == 0 {
		return nil
	}

	thresholdValue := mean + threshold*stdDev
	var bursts []Burst
	var currentBurst *Burst

	for _, point := range timeline {
		zScore := (float64(point.Count) - mean) / stdDev

		if float64(point.Count) > thresholdValue {
			if currentBurst == nil {
				currentBurst = &Burst{
					Start:     point.Time,
					End:       point.Time,
					Peak:      point.Time,
					PeakCount: point.Count,
					Total:     point.Count,
					ZScore:    zScore,
				}
			} else {
				currentBurst.End = point.Time
				currentBurst.Total += point.Count
				if point.Count > currentBurst.PeakCount {
					currentBurst.Peak = point.Time
					currentBurst.PeakCount = point.Count
					currentBurst.ZScore = zScore
				}
			}
		} else {
			if currentBurst != nil {
				bursts = append(bursts, *currentBurst)
				currentBurst = nil
			}
		}
	}

	// Don't forget the last burst
	if currentBurst != nil {
		bursts = append(bursts, *currentBurst)
	}

	// Sort by z-score descending
	sort.Slice(bursts, func(i, j int) bool {
		return bursts[i].ZScore > bursts[j].ZScore
	})

	return bursts
}

// ResponseTime analyzes response times between actors.
type ResponseTimeResult struct {
	From          entity.ActorID `json:"from"`
	To            entity.ActorID `json:"to"`
	MedianMinutes float64        `json:"median_minutes"`
	MeanMinutes   float64        `json:"mean_minutes"`
	Count         int            `json:"count"`
}

// ResponseTimes calculates response times between actors.
func (a *Analyzer) ResponseTimes(ctx context.Context) ([]ResponseTimeResult, error) {
	// Get all interactions sorted by time
	interactions, err := a.store.GetInteractions(ctx, storage.InteractionQuery{})
	if err != nil {
		return nil, err
	}

	// Sort by timestamp
	sort.Slice(interactions, func(i, j int) bool {
		return interactions[i].Timestamp.Before(interactions[j].Timestamp)
	})

	// Track last message time from each actor to each other actor
	lastMessage := make(map[entity.ActorID]map[entity.ActorID]time.Time)
	responseTimes := make(map[entity.ActorID]map[entity.ActorID][]time.Duration)

	for _, interaction := range interactions {
		// Check if this is a response
		if lastMessage[interaction.To] != nil {
			if lastTime, ok := lastMessage[interaction.To][interaction.From]; ok {
				// This might be a response
				responseTime := interaction.Timestamp.Sub(lastTime)
				if responseTime > 0 && responseTime < 7*24*time.Hour { // Max 7 days
					if responseTimes[interaction.From] == nil {
						responseTimes[interaction.From] = make(map[entity.ActorID][]time.Duration)
					}
					responseTimes[interaction.From][interaction.To] = append(
						responseTimes[interaction.From][interaction.To], responseTime)
				}
			}
		}

		// Record this message
		if lastMessage[interaction.From] == nil {
			lastMessage[interaction.From] = make(map[entity.ActorID]time.Time)
		}
		lastMessage[interaction.From][interaction.To] = interaction.Timestamp
	}

	// Calculate statistics
	var results []ResponseTimeResult
	for from, toMap := range responseTimes {
		for to, times := range toMap {
			if len(times) < 2 {
				continue
			}

			// Convert to minutes
			minutes := make([]float64, len(times))
			for i, t := range times {
				minutes[i] = t.Minutes()
			}

			sort.Float64s(minutes)
			median := minutes[len(minutes)/2]
			mean := sum(minutes) / float64(len(minutes))

			results = append(results, ResponseTimeResult{
				From:          from,
				To:            to,
				MedianMinutes: median,
				MeanMinutes:   mean,
				Count:         len(times),
			})
		}
	}

	// Sort by count descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].Count > results[j].Count
	})

	return results, nil
}

// Helper functions

func meanStdDev(values []float64) (mean, stdDev float64) {
	if len(values) == 0 {
		return 0, 0
	}

	mean = sum(values) / float64(len(values))

	variance := 0.0
	for _, v := range values {
		variance += (v - mean) * (v - mean)
	}
	variance /= float64(len(values))
	stdDev = math.Sqrt(variance)

	return mean, stdDev
}

func sum(values []float64) float64 {
	total := 0.0
	for _, v := range values {
		total += v
	}
	return total
}
