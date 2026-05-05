package services

import (
	"energy-monitoring-system/internal/models"
	"math"
)

func ComputeWindowStats(readings []models.LineReading) models.WindowStats {
	n := len(readings)
	if n == 0 {
		return models.WindowStats{}
	}

	s := models.WindowStats{Count: n}
	for _, r := range readings {
		s.MeanPowerLossPct += r.PowerLossPct
		s.MeanDeltaCurrentA += r.DeltaCurrentA
		s.MeanDeltaVoltageV += r.DeltaVoltageV

		if r.PowerLossPct > s.MaxPowerLossPct {
			s.MaxPowerLossPct = r.PowerLossPct
		}
		if r.DeltaCurrentA > s.MaxDeltaCurrentA {
			s.MaxDeltaCurrentA = r.DeltaCurrentA
		}
		if r.DeltaVoltageV > s.MaxDeltaVoltageV {
			s.MaxDeltaVoltageV = r.DeltaVoltageV
		}
		if r.PowerLossPct >= models.PowerLossLowThreshold ||
			r.DeltaCurrentA >= models.LineCurrentLowThreshold ||
			r.DeltaVoltageV >= models.LineVoltageLowThreshold {
			s.SuspiciousCount++
		}
	}

	fn := float64(n)
	s.MeanPowerLossPct /= fn
	s.MeanDeltaCurrentA /= fn
	s.MeanDeltaVoltageV /= fn
	s.PersistenceRate = float64(s.SuspiciousCount) / fn

	for _, r := range readings {
		s.StdDevPowerLossPct += math.Pow(r.PowerLossPct-s.MeanPowerLossPct, 2)
		s.StdDevDeltaCurrentA += math.Pow(r.DeltaCurrentA-s.MeanDeltaCurrentA, 2)
	}
	s.StdDevPowerLossPct = math.Sqrt(s.StdDevPowerLossPct / fn)
	s.StdDevDeltaCurrentA = math.Sqrt(s.StdDevDeltaCurrentA / fn)
	s.TrendSlopePctPerHour, s.TrendR2 = powerLossTrend(readings)

	return s
}

func powerLossTrend(readings []models.LineReading) (slope, r2 float64) {
	n := float64(len(readings))
	if n < 2 {
		return 0, 0
	}
	origin := readings[0].RecordedAt
	var sx, sy, sxy, sx2 float64
	for _, r := range readings {
		x := r.RecordedAt.Sub(origin).Hours()
		y := r.PowerLossPct
		sx += x
		sy += y
		sxy += x * y
		sx2 += x * x
	}
	denom := n*sx2 - sx*sx
	if denom == 0 {
		return 0, 0
	}
	slope = (n*sxy - sx*sy) / denom
	intercept := (sy - slope*sx) / n
	meanY := sy / n
	var ssTot, ssRes float64
	for _, r := range readings {
		x := r.RecordedAt.Sub(origin).Hours()
		predicted := slope*x + intercept
		ssTot += math.Pow(r.PowerLossPct-meanY, 2)
		ssRes += math.Pow(r.PowerLossPct-predicted, 2)
	}
	if ssTot > 0 {
		r2 = 1 - ssRes/ssTot
	}
	return slope, r2
}
