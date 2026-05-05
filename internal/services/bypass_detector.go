package services

import (
	"energy-monitoring-system/internal/models"
	"fmt"
)

func DetectBypass(lr *models.LineReading) models.BypassResult {
	res := models.BypassResult{
		PowerLoss:   lr.PowerLossPct,
		CurrentLoss: lr.DeltaCurrentA,
		VoltageDrop: lr.DeltaVoltageV,
		Severity:    models.SeverityNone,
	}

	highCount := 0
	lowCount := 0

	// Signal 1: power loss
	if lr.PowerLossPct >= models.PowerLossHighThreshold {
		res.Signals = append(res.Signals, fmt.Sprintf("power loss %.2f%% (≥%.0f%% high)", lr.PowerLossPct, models.PowerLossHighThreshold))
		highCount++
	} else if lr.PowerLossPct >= models.PowerLossLowThreshold {
		res.Signals = append(res.Signals, fmt.Sprintf("power loss %.2f%% (≥%.0f%% suspicion)", lr.PowerLossPct, models.PowerLossLowThreshold))
		lowCount++
	}

	// Signal 2: current delta
	if lr.DeltaCurrentA >= models.LineCurrentHighThreshold {
		res.Signals = append(res.Signals, fmt.Sprintf("current delta %.3f A (≥%.1f A high)", lr.DeltaCurrentA, models.LineCurrentHighThreshold))
		highCount++
	} else if lr.DeltaCurrentA >= models.LineCurrentLowThreshold {
		res.Signals = append(res.Signals, fmt.Sprintf("current delta %.3f A (≥%.1f A suspicion)", lr.DeltaCurrentA, models.LineCurrentLowThreshold))
		lowCount++
	}

	// Signal 3: voltage drop
	if lr.DeltaVoltageV >= models.LineVoltageHighThreshold {
		res.Signals = append(res.Signals, fmt.Sprintf("voltage drop %.2f V (≥%.0f V high)", lr.DeltaVoltageV, models.LineVoltageHighThreshold))
		highCount++
	} else if lr.DeltaVoltageV >= models.LineVoltageLowThreshold {
		res.Signals = append(res.Signals, fmt.Sprintf("voltage drop %.2f V (≥%.0f V suspicion)", lr.DeltaVoltageV, models.LineVoltageLowThreshold))
		lowCount++
	}

	if highCount >= 2 || (highCount >= 1 && lowCount >= 1) {
		res.Severity = models.SeverityConfirmed
	} else if highCount >= 1 || lowCount >= 1 {
		res.Severity = models.SeveritySuspect
	}

	if res.Severity != models.SeverityNone {
		res.Reason = fmt.Sprintf("Pole %.1fV/%.3fA -> Meter %.1fV/%.3fA. Triggered: %v",
			lr.PoleVoltageV, lr.PoleCurrentA, lr.MeterVoltageV, lr.MeterCurrentA, res.Signals)
	}

	return res
}
