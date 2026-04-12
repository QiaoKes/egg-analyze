package rocom

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
)

type Range struct {
	Min float64
	Max float64
}

type Row struct {
	ID             int
	Pet            string
	PetID          string
	EggDiameter    string
	EggWeight      string
	DiameterRange  Range
	WeightRange    Range
	DiameterCenter float64
	WeightCenter   float64
}

type Candidate struct {
	Pet         string
	PetID       string
	EggDiameter string
	EggWeight   string
	MatchCount  int
	Probability float64
	MatchType   string
	Score       float64
}

type Engine struct {
	rows []Row
}

type MeasurementPriors struct {
	Diameter Range
	Weight   Range
}

type scoredRow struct {
	row       Row
	matchType string
	score     float64
}

func NewEngine(dataset *Dataset) (*Engine, error) {
	rows := make([]Row, 0, dataset.Total)
	for _, group := range dataset.Groups {
		for _, item := range append(group.RangeItems, group.ExactItems...) {
			diameterRange, err := ParseRange(item.EggDiameter)
			if err != nil {
				return nil, fmt.Errorf("parse diameter %q: %w", item.EggDiameter, err)
			}
			weightRange, err := ParseRange(item.EggWeight)
			if err != nil {
				return nil, fmt.Errorf("parse weight %q: %w", item.EggWeight, err)
			}
			rows = append(rows, Row{
				ID:             item.ID,
				Pet:            group.Pet,
				PetID:          group.PetID,
				EggDiameter:    item.EggDiameter,
				EggWeight:      item.EggWeight,
				DiameterRange:  diameterRange,
				WeightRange:    weightRange,
				DiameterCenter: centerOfRange(diameterRange),
				WeightCenter:   centerOfRange(weightRange),
			})
		}
	}
	return &Engine{rows: rows}, nil
}

func (e *Engine) Search(diameter, weight float64, limit int) []Candidate {
	matches := make([]scoredRow, 0, len(e.rows))
	for _, row := range e.rows {
		matchType, score := evaluateRow(diameter, weight, row)
		matches = append(matches, scoredRow{row: row, matchType: matchType, score: score})
	}

	sort.Slice(matches, func(i, j int) bool {
		return matches[i].score > matches[j].score
	})

	merged := aggregateByPet(matches)
	normalized := normalizeProbabilities(merged)
	if limit > 0 && len(normalized) > limit {
		return normalized[:limit]
	}
	return normalized
}

func (e *Engine) MeasurementPriors() MeasurementPriors {
	if len(e.rows) == 0 {
		return MeasurementPriors{}
	}

	priors := MeasurementPriors{
		Diameter: Range{Min: e.rows[0].DiameterRange.Min, Max: e.rows[0].DiameterRange.Max},
		Weight:   Range{Min: e.rows[0].WeightRange.Min, Max: e.rows[0].WeightRange.Max},
	}
	for _, row := range e.rows[1:] {
		priors.Diameter.Min = math.Min(priors.Diameter.Min, row.DiameterRange.Min)
		priors.Diameter.Max = math.Max(priors.Diameter.Max, row.DiameterRange.Max)
		priors.Weight.Min = math.Min(priors.Weight.Min, row.WeightRange.Min)
		priors.Weight.Max = math.Max(priors.Weight.Max, row.WeightRange.Max)
	}
	return priors
}

func ParseRange(value string) (Range, error) {
	normalized := strings.TrimSpace(value)
	normalized = strings.ReplaceAll(normalized, "~", "-")
	normalized = strings.ReplaceAll(normalized, "–", "-")
	normalized = strings.ReplaceAll(normalized, "—", "-")
	parts := strings.Split(normalized, "-")
	if len(parts) == 1 {
		v, err := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
		if err != nil {
			return Range{}, err
		}
		return Range{Min: v, Max: v}, nil
	}
	if len(parts) != 2 {
		return Range{}, fmt.Errorf("unsupported range %q", value)
	}
	minValue, err := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	if err != nil {
		return Range{}, err
	}
	maxValue, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	if err != nil {
		return Range{}, err
	}
	if minValue > maxValue {
		minValue, maxValue = maxValue, minValue
	}
	return Range{Min: minValue, Max: maxValue}, nil
}

func evaluateRow(diameter, weight float64, row Row) (string, float64) {
	dIn := inRange(diameter, row.DiameterRange)
	wIn := inRange(weight, row.WeightRange)
	dPoint := isPointRange(row.DiameterRange)
	wPoint := isPointRange(row.WeightRange)

	precise := dPoint && wPoint &&
		nearlyEqual(diameter, row.DiameterRange.Min) &&
		nearlyEqual(weight, row.WeightRange.Min)
	if precise {
		return "precise", 1200
	}

	if dPoint && wPoint {
		dDiff := math.Abs(diameter - row.DiameterRange.Min)
		wDiff := math.Abs(weight - row.WeightRange.Min)

		if dDiff <= 0.01 && wDiff <= 0.1 {
			dNorm := dDiff / 0.01
			wNorm := wDiff / 0.1
			distance := math.Sqrt(dNorm*dNorm + wNorm*wNorm)
			return "tolerance1", 1100 - distance*120
		}

		if dDiff <= 0.02 && wDiff <= 0.2 {
			dNorm := dDiff / 0.02
			wNorm := wDiff / 0.2
			distance := math.Sqrt(dNorm*dNorm + wNorm*wNorm)
			return "tolerance2", 900 - distance*140
		}
	}

	if dIn && wIn {
		dHalf := span(row.DiameterRange) / 2
		wHalf := span(row.WeightRange) / 2
		dBaseTol := 0.02
		wBaseTol := 0.4
		dZ := math.Abs(diameter-row.DiameterCenter) / (dHalf + dBaseTol)
		wZ := math.Abs(weight-row.WeightCenter) / (wHalf + wBaseTol)
		dScore := gaussian(dZ)
		wScore := gaussian(wZ)
		score := math.Pow(dScore, 0.58) * math.Pow(wScore, 0.42)
		precisionBoost := 1 + 0.16*(1/(1+span(row.DiameterRange)*12)) + 0.12*(1/(1+span(row.WeightRange)*2))
		score *= clamp(precisionBoost, 1, 1.28)
		return "matched", score
	}

	score := 1 / (1 + distanceToRange(diameter, row.DiameterRange)/0.05 + distanceToRange(weight, row.WeightRange)/1.0)
	return "nearest", score
}

func aggregateByPet(rows []scoredRow) []Candidate {
	grouped := make(map[string][]scoredRow)
	for _, row := range rows {
		grouped[row.row.Pet] = append(grouped[row.row.Pet], row)
	}

	merged := make([]Candidate, 0, len(grouped))
	for _, list := range grouped {
		sort.Slice(list, func(i, j int) bool {
			return list[i].score > list[j].score
		})
		petScore := 0.0
		for idx, item := range list {
			petScore += item.score * math.Pow(0.58, float64(idx))
		}
		best := list[0]
		merged = append(merged, Candidate{
			Pet:         best.row.Pet,
			PetID:       best.row.PetID,
			EggDiameter: best.row.EggDiameter,
			EggWeight:   best.row.EggWeight,
			MatchCount:  len(list),
			MatchType:   best.matchType,
			Score:       petScore,
		})
	}

	sort.Slice(merged, func(i, j int) bool {
		return merged[i].Score > merged[j].Score
	})
	return merged
}

func normalizeProbabilities(items []Candidate) []Candidate {
	sum := 0.0
	for _, item := range items {
		sum += item.Score
	}
	if sum <= 0 {
		return items
	}
	normalized := make([]Candidate, 0, len(items))
	for _, item := range items {
		item.Probability = math.Round((item.Score/sum)*100*100) / 100
		normalized = append(normalized, item)
	}
	return normalized
}

func nearlyEqual(left, right float64) bool {
	return math.Abs(left-right) <= 1e-9
}

func inRange(value float64, r Range) bool {
	return value >= r.Min && value <= r.Max
}

func isPointRange(r Range) bool {
	return nearlyEqual(r.Min, r.Max)
}

func span(r Range) float64 {
	return r.Max - r.Min
}

func centerOfRange(r Range) float64 {
	return (r.Min + r.Max) / 2
}

func distanceToRange(value float64, r Range) float64 {
	if value < r.Min {
		return r.Min - value
	}
	if value > r.Max {
		return value - r.Max
	}
	return 0
}

func gaussian(z float64) float64 {
	return math.Exp(-0.5 * z * z)
}

func clamp(value, minValue, maxValue float64) float64 {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}
