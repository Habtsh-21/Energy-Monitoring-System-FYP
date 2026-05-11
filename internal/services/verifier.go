package services

import (
	"energy-monitoring-system/internal/models"
	"fmt"
	"math"
)

// Verdict is the final cross-checked decision for a single reading.
type Verdict string

const (
	VerdictNormal    Verdict = "normal"
	VerdictSuspect   Verdict = "suspect"
	VerdictConfirmed Verdict = "confirmed"
)

// VerificationResult is the output of VerifyReading.
type VerificationResult struct {
	Verdict        Verdict  `json:"verdict"`
	MeterClaim     string   `json:"meter_claim"`      // what the meter reported
	OurDetection   string   `json:"our_detection"`    // what our analysis found
	Conflict       bool     `json:"conflict"`         // true when the two disagree
	ConflictReason string   `json:"conflict_reason,omitempty"`
	Signals        []string `json:"signals,omitempty"`
	Reason         string   `json:"reason,omitempty"`
}

// VerifyReading independently analyses a LineReadingRequest and compares
// it against the bypass status the meter itself reported. It returns a
// final Verdict and flags any conflict between the two sources.
func VerifyReading(req *models.LineReadingRequest) VerificationResult {
	// Build a synthetic LineReading so we can reuse DetectBypass.
	lr := toLineReading(req)
	detection := DetectBypass(&lr)

	ourDetection := string(detection.Severity)
	meterClaim := normaliseMeterClaim(req.BypassStatus)

	conflict := ourDetection != meterClaim
	var conflictReason string
	if conflict {
		conflictReason = fmt.Sprintf(
			"meter reported %q but independent analysis found %q",
			meterClaim, ourDetection,
		)
	}

	// Final verdict: take the more severe of the two.
	verdict := pickStricter(Verdict(ourDetection), Verdict(meterClaim))

	return VerificationResult{
		Verdict:        verdict,
		MeterClaim:     meterClaim,
		OurDetection:   ourDetection,
		Conflict:       conflict,
		ConflictReason: conflictReason,
		Signals:        detection.Signals,
		Reason:         detection.Reason,
	}
}

// toLineReading converts the inbound request into a models.LineReading
// so existing detection logic can be applied without duplication.
func toLineReading(req *models.LineReadingRequest) models.LineReading {
	poleP := req.PoleVoltageV * req.PoleCurrentA
	meterP := req.MeterVoltageV * req.MeterCurrentA

	deltaCurrentA := math.Abs(req.PoleCurrentA - req.MeterCurrentA)
	deltaVoltageV := math.Abs(req.PoleVoltageV - req.MeterVoltageV)

	var powerLossPct float64
	if poleP > 0 {
		powerLossPct = ((poleP - meterP) / poleP) * 100
	}

	return models.LineReading{
		PoleVoltageV:  req.PoleVoltageV,
		PoleCurrentA:  req.PoleCurrentA,
		MeterVoltageV: req.MeterVoltageV,
		MeterCurrentA: req.MeterCurrentA,
		DeltaCurrentA: deltaCurrentA,
		DeltaVoltageV: deltaVoltageV,
		PowerLossPct:  powerLossPct,
	}
}

// normaliseMeterClaim maps the raw string the meter sends to one of our
// three canonical severity strings.
func normaliseMeterClaim(status string) string {
	switch status {
	case "confirmed", "CONFIRMED", "bypass_confirmed":
		return string(models.SeverityConfirmed)
	case "suspect", "SUSPECT", "bypass_suspect":
		return string(models.SeveritySuspect)
	default:
		return string(models.SeverityNormal)
	}
}

// pickStricter returns whichever verdict is more severe.
func pickStricter(a, b Verdict) Verdict {
	if severityRank(a) >= severityRank(b) {
		return a
	}
	return b
}

func severityRank(v Verdict) int {
	switch v {
	case VerdictConfirmed:
		return 2
	case VerdictSuspect:
		return 1
	default:
		return 0
	}
}