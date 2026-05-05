package services

import (
	"energy-monitoring-system/internal/models"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/google/uuid"
)

type AnalyserConfig struct {
	WindowSize       int
	MinReadings      int
	PersistenceRatio float64
	ZScoreThreshold  float64
}

func DefaultAnalyserConfig() AnalyserConfig {
	return AnalyserConfig{
		WindowSize:       168,
		MinReadings:      24,
		PersistenceRatio: 0.30,
		ZScoreThreshold:  3.0,
	}
}

type RiskLevel string

const (
	RiskNone     RiskLevel = "none"
	RiskLow      RiskLevel = "low"
	RiskMedium   RiskLevel = "medium"
	RiskHigh     RiskLevel = "high"
	RiskCritical RiskLevel = "critical"
)

type SpikeEvent struct {
	ReadingID   uuid.UUID `json:"reading_id"`
	RecordedAt  time.Time `json:"recorded_at"`
	PowerLoss   float64   `json:"power_loss_pct"`
	CurrentLoss float64   `json:"current_loss_a"`
	ZScore      float64   `json:"z_score"`
}

type PatternTag string

const (
	TagPersistent    PatternTag = "persistent_loss"
	TagSpiky         PatternTag = "intermittent_spikes"
	TagHighMagnitude PatternTag = "high_magnitude"
)

type MeterAnalysisReport struct {
	MeterID        uuid.UUID          `json:"meter_id"`
	AnalysedAt     time.Time          `json:"analysed_at"`
	WindowStart    time.Time          `json:"window_start"`
	WindowEnd      time.Time          `json:"window_end"`
	WindowSize     int                `json:"window_size"`
	Risk           RiskLevel          `json:"risk"`
	Tags           []PatternTag       `json:"tags"`
	Stats          models.WindowStats `json:"stats"`
	Spikes         []SpikeEvent       `json:"spikes"`
	Findings       []string           `json:"findings"`
	Recommendation string             `json:"recommendation"`
}

type BatchReport struct {
	RunAt        time.Time              `json:"run_at"`
	TotalMeters  int                    `json:"total_meters"`
	SkippedMeters int                   `json:"skipped_meters"`
	ByCritical   int                    `json:"critical"`
	ByHigh       int                    `json:"high"`
	ByMedium     int                    `json:"medium"`
	ByLow        int                    `json:"low"`
	ByNone       int                    `json:"none"`
	Reports      []*MeterAnalysisReport `json:"reports"`
}

type LineAnalyser struct {
	cfg AnalyserConfig
}

func NewLineAnalyser(cfg AnalyserConfig) *LineAnalyser {
	return &LineAnalyser{cfg: cfg}
}

func (a *LineAnalyser) AnalyseMeter(meterID uuid.UUID) (*MeterAnalysisReport, error) {
	readings, err := models.GetRecentLineReadings(meterID, a.cfg.WindowSize)
	if err != nil {
		return nil, fmt.Errorf("fetch readings for %s: %w", meterID, err)
	}

	report := &MeterAnalysisReport{
		MeterID:    meterID,
		AnalysedAt: time.Now(),
		WindowSize: len(readings),
	}

	if len(readings) == 0 {
		report.Risk = RiskNone
		report.Findings = []string{"No line readings found for this meter."}
		report.Recommendation = "Verify the IoT collector is online."
		return report, nil
	}

	sort.Slice(readings, func(i, j int) bool {
		return readings[i].RecordedAt.Before(readings[j].RecordedAt)
	})
	report.WindowStart = readings[0].RecordedAt
	report.WindowEnd = readings[len(readings)-1].RecordedAt

	if len(readings) < a.cfg.MinReadings {
		report.Risk = RiskLow
		report.Findings = []string{fmt.Sprintf("Insufficient data: %d readings available.", len(readings))}
		report.Recommendation = "Allow more data to accumulate."
		return report, nil
	}

	stats := ComputeWindowStats(readings)
	report.Stats = stats
	report.Spikes = a.detectSpikes(readings, stats)

	risk, tags, findings, rec := a.scoreAndAssess(stats, report.Spikes)
	report.Risk = risk
	report.Tags = tags
	report.Findings = findings
	report.Recommendation = rec

	return report, nil
}

// hasSeverity loads the recent readings for a meter and returns true
// if at least one reading triggers a non-None bypass severity.
// This is a cheap pre-flight check; full analysis is only run when true.
func (a *LineAnalyser) hasSeverity(meterID uuid.UUID) (bool, error) {
	readings, err := models.GetRecentLineReadings(meterID, a.cfg.WindowSize)
	if err != nil {
		return false, err
	}
	for i := range readings {
		res := DetectBypass(&readings[i])
		if res.Severity != models.SeverityNone {
			return true, nil
		}
	}
	return false, nil
}

func (a *LineAnalyser) RunNightlyBatch() (*BatchReport, error) {
	ids, err := models.GetAllActiveMeterIDs()
	if err != nil {
		return nil, fmt.Errorf("fetch active meters: %w", err)
	}

	batch := &BatchReport{
		RunAt:       time.Now(),
		TotalMeters: len(ids),
		Reports:     make([]*MeterAnalysisReport, 0),
	}

	for _, id := range ids {
		// Gate: only analyse meters where bypass severity is detected.
		detected, err := a.hasSeverity(id)
		if err != nil {
			// Log and skip on fetch error; don't treat as a severity event.
			batch.SkippedMeters++
			continue
		}
		if !detected {
			// No severity signals — skip full analysis entirely.
			batch.SkippedMeters++
			continue
		}

		r, err := a.AnalyseMeter(id)
		if err != nil {
			r = &MeterAnalysisReport{
				MeterID:        id,
				AnalysedAt:     time.Now(),
				Risk:           RiskNone,
				Findings:       []string{fmt.Sprintf("Analysis error: %v", err)},
				Recommendation: "Check logs.",
			}
		}
		batch.Reports = append(batch.Reports, r)
		updateBatchCounts(batch, r.Risk)
	}

	sort.Slice(batch.Reports, func(i, j int) bool {
		return riskOrdinal(batch.Reports[i].Risk) > riskOrdinal(batch.Reports[j].Risk)
	})

	return batch, nil
}

func updateBatchCounts(b *BatchReport, r RiskLevel) {
	switch r {
	case RiskCritical:
		b.ByCritical++
	case RiskHigh:
		b.ByHigh++
	case RiskMedium:
		b.ByMedium++
	case RiskLow:
		b.ByLow++
	default:
		b.ByNone++
	}
}

func riskOrdinal(r RiskLevel) int {
	switch r {
	case RiskCritical:
		return 4
	case RiskHigh:
		return 3
	case RiskMedium:
		return 2
	case RiskLow:
		return 1
	default:
		return 0
	}
}

func (a *LineAnalyser) detectSpikes(readings []models.LineReading, stats models.WindowStats) []SpikeEvent {
	if stats.StdDevPowerLossPct == 0 {
		return nil
	}
	var spikes []SpikeEvent
	for _, r := range readings {
		z := (r.PowerLossPct - stats.MeanPowerLossPct) / stats.StdDevPowerLossPct
		if z >= a.cfg.ZScoreThreshold {
			spikes = append(spikes, SpikeEvent{
				ReadingID:   r.ID,
				RecordedAt:  r.RecordedAt,
				PowerLoss:   r.PowerLossPct,
				CurrentLoss: r.DeltaCurrentA,
				ZScore:      math.Round(z*100) / 100,
			})
		}
	}
	return spikes
}

func (a *LineAnalyser) scoreAndAssess(stats models.WindowStats, spikes []SpikeEvent) (RiskLevel, []PatternTag, []string, string) {
	score := 0
	var tags []PatternTag
	var findings []string

	if stats.PersistenceRate >= a.cfg.PersistenceRatio {
		tags = append(tags, TagPersistent)
		findings = append(findings, "Persistent loss detected.")
		score += 3
	}

	if stats.MeanPowerLossPct >= models.PowerLossHighThreshold {
		tags = append(tags, TagHighMagnitude)
		findings = append(findings, "High average power loss.")
		score += 4
	} else if stats.MeanPowerLossPct >= models.PowerLossLowThreshold {
		score += 2
	}

	if len(spikes) > 0 {
		tags = append(tags, TagSpiky)
		findings = append(findings, fmt.Sprintf("%d spike event(s) detected.", len(spikes)))
		score += 2
	}

	risk := RiskNone
	rec := "Line is healthy."
	switch {
	case score >= 7:
		risk = RiskCritical
		rec = "URGENT - dispatch field crew."
	case score >= 4:
		risk = RiskHigh
		rec = "Schedule field inspection."
	case score >= 2:
		risk = RiskMedium
		rec = "Monitor closely."
	case score >= 1:
		risk = RiskLow
		rec = "Isolated weak signals."
	}

	return risk, tags, findings, rec
}
