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

type Candidate struct {
	Pet                string
	PetID              string
	PortraitKey        string
	EggHeight          string
	EggWeight          string
	MatchCount         int
	Implemented        bool
	Probability        float64
	MatchType          string
	Score              float64
	SourcePet          string
	HatchLabel         string
	MatchedMetricCount int
}

type Engine struct {
	chains []chainEntry
	priors MeasurementPriors
}

type MeasurementPriors struct {
	Diameter Range
	Weight   Range
}

type chainEntry struct {
	RootPet     Pet
	MemberCount int
	Variants    []variantEntry
}

type variantEntry struct {
	Key           string
	SourcePetID   int
	SourcePetName string
	HeightRange   Range
	WeightRange   Range
	HasHeight     bool
	HasWeight     bool
	HatchSeconds  int
}

type matchResult struct {
	score              float64
	matchedMetricCount int
}

func NewEngine(dataset *Dataset) (*Engine, error) {
	petsByID := make(map[int]Pet, len(dataset.Pets))
	for _, pet := range dataset.Pets {
		petsByID[pet.ID] = pet
	}

	rootIDCache := make(map[int]int, len(dataset.Pets))
	grouped := make(map[int]*chainEntry)
	var priors MeasurementPriors
	hasHeightPriors := false
	hasWeightPriors := false

	for _, pet := range dataset.Pets {
		rootID := resolveRootID(pet, petsByID, rootIDCache)
		rootPet, ok := petsByID[rootID]
		if !ok {
			rootPet = pet
		}

		entry, ok := grouped[rootID]
		if !ok {
			entry = &chainEntry{RootPet: rootPet}
			grouped[rootID] = entry
		}
		entry.MemberCount++

		seenVariantKeys := make(map[string]struct{}, len(entry.Variants))
		for _, existing := range entry.Variants {
			seenVariantKeys[existing.Key] = struct{}{}
		}

		for _, variant := range extractBreedingVariants(pet) {
			key := buildVariantKey(pet.ID, variant)
			if _, exists := seenVariantKeys[key]; exists {
				continue
			}

			row, ok := buildVariantEntry(pet, variant)
			if !ok {
				continue
			}

			entry.Variants = append(entry.Variants, row)
			seenVariantKeys[key] = struct{}{}

			if row.HasHeight {
				heightPriors := Range{
					Min: centimetersToMeters(row.HeightRange.Min),
					Max: centimetersToMeters(row.HeightRange.Max),
				}
				if !hasHeightPriors {
					priors.Diameter = heightPriors
					hasHeightPriors = true
				} else {
					priors.Diameter.Min = math.Min(priors.Diameter.Min, heightPriors.Min)
					priors.Diameter.Max = math.Max(priors.Diameter.Max, heightPriors.Max)
				}
			}
			if row.HasWeight {
				if !hasWeightPriors {
					priors.Weight = row.WeightRange
					hasWeightPriors = true
				} else {
					priors.Weight.Min = math.Min(priors.Weight.Min, row.WeightRange.Min)
					priors.Weight.Max = math.Max(priors.Weight.Max, row.WeightRange.Max)
				}
			}
		}
	}

	chains := make([]chainEntry, 0, len(grouped))
	for _, entry := range grouped {
		if len(entry.Variants) == 0 {
			continue
		}
		chains = append(chains, *entry)
	}

	sort.Slice(chains, func(i, j int) bool {
		return petDisplayName(chains[i].RootPet) < petDisplayName(chains[j].RootPet)
	})

	return &Engine{
		chains: chains,
		priors: priors,
	}, nil
}

func (e *Engine) Search(height, weight float64, limit int) []Candidate {
	matches := make([]Candidate, 0, len(e.chains))
	for _, chain := range e.chains {
		var best *Candidate
		for _, variant := range chain.Variants {
			evaluation, ok := evaluateVariantMatch(height, weight, variant)
			if !ok {
				continue
			}

			candidate := Candidate{
				Pet:                petDisplayName(chain.RootPet),
				PetID:              strconv.Itoa(chain.RootPet.ID),
				PortraitKey:        strings.TrimSpace(chain.RootPet.Name),
				EggHeight:          formatRange(variant.HeightRange, variant.HasHeight, "cm"),
				EggWeight:          formatRange(variant.WeightRange, variant.HasWeight, "kg"),
				MatchCount:         chain.MemberCount,
				Implemented:        chain.RootPet.Implemented,
				MatchType:          buildMatchType(chain.RootPet, variant, evaluation.matchedMetricCount),
				Score:              evaluation.score,
				SourcePet:          variant.SourcePetName,
				HatchLabel:         formatDurationLabel(variant.HatchSeconds),
				MatchedMetricCount: evaluation.matchedMetricCount,
			}

			if best == nil || candidate.Score < best.Score || (candidate.Score == best.Score && candidate.Pet < best.Pet) {
				copy := candidate
				best = &copy
			}
		}

		if best != nil {
			matches = append(matches, *best)
		}
	}

	if implemented := filterImplemented(matches); len(implemented) > 0 {
		matches = implemented
	}

	matches = assignProbabilities(matches)

	sort.Slice(matches, func(i, j int) bool {
		if matches[i].Probability != matches[j].Probability {
			return matches[i].Probability > matches[j].Probability
		}
		if matches[i].Score != matches[j].Score {
			return matches[i].Score < matches[j].Score
		}
		return matches[i].Pet < matches[j].Pet
	})

	if limit > 0 && len(matches) > limit {
		return matches[:limit]
	}
	return matches
}

func (e *Engine) MeasurementPriors() MeasurementPriors {
	return e.priors
}

func resolveRootID(pet Pet, petsByID map[int]Pet, cache map[int]int) int {
	if cached, ok := cache[pet.ID]; ok {
		return cached
	}

	visited := make([]int, 0, 4)
	current := pet
	seen := make(map[int]struct{})
	for current.EvolvesFromID != nil {
		if _, exists := seen[current.ID]; exists {
			break
		}
		seen[current.ID] = struct{}{}
		visited = append(visited, current.ID)

		parent, ok := petsByID[*current.EvolvesFromID]
		if !ok {
			break
		}
		current = parent
	}

	cache[current.ID] = current.ID
	for _, id := range visited {
		cache[id] = current.ID
	}
	return current.ID
}

func extractBreedingVariants(pet Pet) []BreedingVariant {
	if pet.Breeding == nil {
		return nil
	}

	var variants []BreedingVariant
	if len(pet.Breeding.Variants) > 0 {
		variants = pet.Breeding.Variants
	} else {
		variants = []BreedingVariant{pet.Breeding.BreedingVariant}
	}

	filtered := make([]BreedingVariant, 0, len(variants))
	for _, variant := range variants {
		if variant.HeightLow == nil && variant.HeightHigh == nil && variant.WeightLow == nil && variant.WeightHigh == nil {
			continue
		}
		filtered = append(filtered, variant)
	}
	return filtered
}

func buildVariantKey(petID int, variant BreedingVariant) string {
	parts := []string{
		strconv.Itoa(petID),
		intPointerString(variant.ID),
		floatPointerString(variant.HeightLow),
		floatPointerString(variant.HeightHigh),
		floatPointerString(variant.WeightLow),
		floatPointerString(variant.WeightHigh),
		intPointerString(variant.HatchData),
	}
	return strings.Join(parts, ":")
}

func buildVariantEntry(pet Pet, variant BreedingVariant) (variantEntry, bool) {
	heightRange, hasHeight := buildRange(variant.HeightLow, variant.HeightHigh, identity)
	weightRange, hasWeight := buildRange(variant.WeightLow, variant.WeightHigh, gramsToKilograms)
	if !hasHeight && !hasWeight {
		return variantEntry{}, false
	}

	return variantEntry{
		Key:           buildVariantKey(pet.ID, variant),
		SourcePetID:   pet.ID,
		SourcePetName: petDisplayName(pet),
		HeightRange:   heightRange,
		WeightRange:   weightRange,
		HasHeight:     hasHeight,
		HasWeight:     hasWeight,
		HatchSeconds:  intValue(variant.HatchData),
	}, true
}

func buildRange(low, high *float64, transform func(float64) float64) (Range, bool) {
	switch {
	case low == nil && high == nil:
		return Range{}, false
	case low != nil && high != nil:
		minValue := transform(*low)
		maxValue := transform(*high)
		if minValue > maxValue {
			minValue, maxValue = maxValue, minValue
		}
		return Range{Min: minValue, Max: maxValue}, true
	case low != nil:
		value := transform(*low)
		return Range{Min: value, Max: value}, true
	default:
		value := transform(*high)
		return Range{Min: value, Max: value}, true
	}
}

func evaluateVariantMatch(height, weight float64, variant variantEntry) (matchResult, bool) {
	score := 0.0
	matchedMetricCount := 0

	if variant.HasHeight {
		heightMatch, ok := evaluateMetric(height, variant.HeightRange)
		if !ok {
			return matchResult{}, false
		}
		score += heightMatch
		matchedMetricCount++
	}

	if variant.HasWeight {
		weightMatch, ok := evaluateMetric(weight, variant.WeightRange)
		if !ok {
			return matchResult{}, false
		}
		score += weightMatch
		matchedMetricCount++
	}

	if matchedMetricCount == 0 {
		return matchResult{}, false
	}

	return matchResult{
		score:              score,
		matchedMetricCount: matchedMetricCount,
	}, true
}

func evaluateMetric(value float64, r Range) (float64, bool) {
	if value < r.Min || value > r.Max {
		return 0, false
	}

	center := (r.Min + r.Max) / 2
	span := math.Max(r.Max-r.Min, 1)
	centerPenalty := math.Abs(value-center) / math.Max(span/2, 1)
	spanPenalty := (span / math.Max(center, 1)) * 0.35
	return centerPenalty + spanPenalty, true
}

func filterImplemented(items []Candidate) []Candidate {
	result := make([]Candidate, 0, len(items))
	for _, item := range items {
		if item.Implemented {
			result = append(result, item)
		}
	}
	return result
}

func assignProbabilities(items []Candidate) []Candidate {
	if len(items) == 0 {
		return items
	}

	weights := make([]float64, len(items))
	sum := 0.0
	for i, item := range items {
		weight := math.Exp(-item.Score)
		weights[i] = weight
		sum += weight
	}

	if sum <= 0 {
		return items
	}

	for i := range items {
		items[i].Probability = math.Round((weights[i]/sum)*100*100) / 100
	}
	return items
}

func buildMatchType(root Pet, variant variantEntry, matchedMetricCount int) string {
	prefix := "双维命中"
	if matchedMetricCount == 1 {
		prefix = "单维命中"
	}

	if variant.SourcePetName == "" || variant.SourcePetName == petDisplayName(root) {
		return prefix + "最低阶段"
	}
	return fmt.Sprintf("%s %s，结果回落到最低阶段", prefix, variant.SourcePetName)
}

func formatRange(r Range, ok bool, unit string) string {
	if !ok {
		return "暂无数据"
	}
	if nearlyEqual(r.Min, r.Max) {
		return formatNumber(r.Min) + unit
	}
	return formatNumber(r.Min) + "-" + formatNumber(r.Max) + unit
}

func formatNumber(value float64) string {
	if nearlyEqual(value, math.Round(value)) {
		return strconv.FormatInt(int64(math.Round(value)), 10)
	}
	return strconv.FormatFloat(value, 'f', 2, 64)
}

func formatDurationLabel(seconds int) string {
	if seconds <= 0 {
		return ""
	}
	if seconds%86400 == 0 {
		return strconv.Itoa(seconds/86400) + " 天"
	}
	hours := float64(seconds) / 3600
	if nearlyEqual(hours, math.Round(hours)) {
		return strconv.Itoa(int(math.Round(hours))) + " 小时"
	}
	return formatNumber(hours) + " 小时"
}

func petDisplayName(pet Pet) string {
	name := strings.TrimSpace(pet.Localized.ZH.Name)
	if name != "" {
		return name
	}
	name = strings.TrimSpace(pet.Name)
	if name != "" {
		return name
	}
	return strconv.Itoa(pet.ID)
}

func gramsToKilograms(value float64) float64 {
	return value / 1000
}

func centimetersToMeters(value float64) float64 {
	return value / 100
}

func identity(value float64) float64 {
	return value
}

func nearlyEqual(left, right float64) bool {
	return math.Abs(left-right) <= 1e-9
}

func intPointerString(value *int) string {
	if value == nil {
		return "na"
	}
	return strconv.Itoa(*value)
}

func floatPointerString(value *float64) string {
	if value == nil {
		return "na"
	}
	return strconv.FormatFloat(*value, 'f', -1, 64)
}

func intValue(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}
