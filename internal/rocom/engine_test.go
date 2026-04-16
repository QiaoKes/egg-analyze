package rocom

import (
	"math"
	"strings"
	"testing"
)

func TestMeasurementPriorsConvertWeightToKilograms(t *testing.T) {
	engine, err := NewEngine(&Dataset{
		Pets: []Pet{
			testPet(3543, "duoling", "多灵", true, nil, []BreedingVariant{
				testVariant(3543001, 24, 28, 4690, 7360, 57600),
			}),
		},
	})
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}

	priors := engine.MeasurementPriors()
	if math.Abs(priors.Diameter.Min-0.24) > 0.001 || math.Abs(priors.Diameter.Max-0.28) > 0.001 {
		t.Fatalf("unexpected height priors: %+v", priors.Diameter)
	}
	if math.Abs(priors.Weight.Min-4.69) > 0.001 || math.Abs(priors.Weight.Max-7.36) > 0.001 {
		t.Fatalf("unexpected weight priors: %+v", priors.Weight)
	}
}

func TestSearchUsesEvolutionChainAndPrefersImplementedResults(t *testing.T) {
	rootID := 3543
	engine, err := NewEngine(&Dataset{
		Pets: []Pet{
			testPet(3543, "duoling", "多灵", true, nil, []BreedingVariant{
				testVariant(3543001, 29, 34, 6000, 8200, 57600),
			}),
			testPet(3544, "duoling_stage2", "多灵二阶段", true, &rootID, []BreedingVariant{
				testVariant(3544001, 24, 28, 4690, 7360, 57600),
			}),
			testPet(3187, "mimiyan", "迷迷眼", false, nil, []BreedingVariant{
				testVariant(3187001, 24, 28, 4690, 7360, 57600),
			}),
		},
	})
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}

	matches := engine.Search(26, 6.26, 5)
	if len(matches) != 1 {
		t.Fatalf("expected only implemented matches, got %d", len(matches))
	}

	got := matches[0]
	if got.PetID != "3543" || got.Pet != "多灵" {
		t.Fatalf("unexpected root match: %+v", got)
	}
	if got.PortraitKey != "duoling" {
		t.Fatalf("unexpected portrait key: got %q want %q", got.PortraitKey, "duoling")
	}
	if !strings.Contains(got.MatchType, "多灵二阶段") {
		t.Fatalf("expected match type to mention source stage, got %q", got.MatchType)
	}
	if got.EggHeight != "24-28cm" {
		t.Fatalf("unexpected height label: %q", got.EggHeight)
	}
	if got.EggWeight != "4.69-7.36kg" {
		t.Fatalf("unexpected weight label: %q", got.EggWeight)
	}
	if math.Abs(got.Score-0.38498) > 0.0001 {
		t.Fatalf("unexpected score: got %.5f want %.5f", got.Score, 0.38498)
	}
	if got.Probability <= 0 {
		t.Fatalf("expected positive probability, got %.2f", got.Probability)
	}
}

func TestSearchFallsBackToUnimplementedWhenNeeded(t *testing.T) {
	engine, err := NewEngine(&Dataset{
		Pets: []Pet{
			testPet(3187, "mimiyan", "迷迷眼", false, nil, []BreedingVariant{
				testVariant(3187001, 25, 35, 5600, 8600, 57600),
			}),
		},
	})
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}

	matches := engine.Search(26, 6.26, 5)
	if len(matches) != 1 {
		t.Fatalf("expected one fallback match, got %d", len(matches))
	}
	if matches[0].Implemented {
		t.Fatalf("expected unimplemented fallback match, got %+v", matches[0])
	}
}

func TestSearchSortsByProbabilityDescending(t *testing.T) {
	engine, err := NewEngine(&Dataset{
		Pets: []Pet{
			testPet(3543, "duoling", "多灵", true, nil, []BreedingVariant{
				testVariant(3543001, 24, 28, 4690, 7360, 57600),
			}),
			testPet(3382, "shuangdengyu", "双灯鱼", true, nil, []BreedingVariant{
				testVariant(3382001, 21, 30, 5215, 7120, 57600),
			}),
		},
	})
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}

	matches := engine.Search(26, 6.26, 5)
	if len(matches) != 2 {
		t.Fatalf("expected two matches, got %d", len(matches))
	}
	if matches[0].Probability < matches[1].Probability {
		t.Fatalf("matches not sorted by probability desc: %.2f < %.2f", matches[0].Probability, matches[1].Probability)
	}
	if matches[0].PetID != "3543" {
		t.Fatalf("expected best probability first, got %+v", matches[0])
	}
}

func testPet(id int, portraitKey, localizedName string, implemented bool, evolvesFromID *int, variants []BreedingVariant) Pet {
	info := &BreedingInfo{
		BreedingVariant: variants[0],
		Variants:        variants,
	}
	return Pet{
		ID:            id,
		Name:          portraitKey,
		Implemented:   implemented,
		EvolvesFromID: evolvesFromID,
		Breeding:      info,
		Localized: LocalizedPet{
			ZH: LocalizedPetName{Name: localizedName},
		},
	}
}

func testVariant(id int, heightLow, heightHigh, weightLow, weightHigh float64, hatchData int) BreedingVariant {
	return BreedingVariant{
		ID:         intPointer(id),
		PetID:      intPointer(id),
		Name:       "variant",
		HatchData:  intPointer(hatchData),
		HeightLow:  floatPointer(heightLow),
		HeightHigh: floatPointer(heightHigh),
		WeightLow:  floatPointer(weightLow),
		WeightHigh: floatPointer(weightHigh),
	}
}

func intPointer(value int) *int {
	return &value
}

func floatPointer(value float64) *float64 {
	return &value
}
