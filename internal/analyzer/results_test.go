package analyzer

import (
	"testing"

	"egg-analyze/internal/rocom"
)

func TestBuildResultsConvertsHeightMetersToCentimeters(t *testing.T) {
	engine, err := rocom.NewEngine(&rocom.Dataset{
		Pets: []rocom.Pet{
			testAnalyzerPet(3543, "duoling", "多灵", true, []rocom.BreedingVariant{
				testAnalyzerVariant(3543001, 24, 28, 4690, 7360, 57600),
			}),
		},
	})
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}

	results := buildResults(engine, []Measurement{{
		Size:   0.26,
		Weight: 6.26,
	}}, 5)
	if len(results) != 1 {
		t.Fatalf("expected one result, got %d", len(results))
	}
	if len(results[0].Candidates) != 1 {
		t.Fatalf("expected one candidate, got %d", len(results[0].Candidates))
	}
	if results[0].Candidates[0].PetID != "3543" {
		t.Fatalf("unexpected candidate after unit conversion: %+v", results[0].Candidates[0])
	}
}

func testAnalyzerPet(id int, portraitKey, localizedName string, implemented bool, variants []rocom.BreedingVariant) rocom.Pet {
	info := &rocom.BreedingInfo{
		BreedingVariant: variants[0],
		Variants:        variants,
	}
	return rocom.Pet{
		ID:          id,
		Name:        portraitKey,
		Implemented: implemented,
		Breeding:    info,
		Localized: rocom.LocalizedPet{
			ZH: rocom.LocalizedPetName{Name: localizedName},
		},
	}
}

func testAnalyzerVariant(id int, heightLow, heightHigh, weightLow, weightHigh float64, hatchData int) rocom.BreedingVariant {
	return rocom.BreedingVariant{
		ID:         analyzerIntPointer(id),
		PetID:      analyzerIntPointer(id),
		Name:       "variant",
		HatchData:  analyzerIntPointer(hatchData),
		HeightLow:  analyzerFloatPointer(heightLow),
		HeightHigh: analyzerFloatPointer(heightHigh),
		WeightLow:  analyzerFloatPointer(weightLow),
		WeightHigh: analyzerFloatPointer(weightHigh),
	}
}

func analyzerIntPointer(value int) *int {
	return &value
}

func analyzerFloatPointer(value float64) *float64 {
	return &value
}
