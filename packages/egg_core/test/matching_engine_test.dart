import 'package:egg_core/egg_core.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('returns probabilities in descending order', () {
    final engine = MatchingEngine(
      PetsDataset(
        pets: [
          _pet(
            id: 3543,
            name: 'duoling',
            localizedName: '多灵',
            implemented: true,
            variants: [_variant(24, 28, 4690, 7360)],
          ),
          _pet(
            id: 3382,
            name: 'shuangdengyu',
            localizedName: '双灯鱼',
            implemented: true,
            variants: [_variant(21, 30, 5215, 7120)],
          ),
        ],
      ),
    );

    final matches = engine.search(
      heightInCentimeters: 26,
      weightInKg: 6.26,
      limit: 5,
    );

    expect(matches, hasLength(2));
    expect(matches.first.petId, 3543);
    expect(matches.first.probability, greaterThan(matches.last.probability));
  });

  test('measurement priors convert dataset height to meters', () {
    final engine = MatchingEngine(
      PetsDataset(
        pets: [
          _pet(
            id: 3543,
            name: 'duoling',
            localizedName: '多灵',
            implemented: true,
            variants: [_variant(24, 28, 4690, 7360)],
          ),
        ],
      ),
    );

    expect(engine.measurementPriors.diameter.min, closeTo(0.24, 0.001));
    expect(engine.measurementPriors.weight.min, closeTo(4.69, 0.001));
  });
}

Pet _pet({
  required int id,
  required String name,
  required String localizedName,
  required bool implemented,
  int? evolvesFromId,
  required List<BreedingVariant> variants,
}) {
  return Pet(
    id: id,
    name: name,
    localizedName: localizedName,
    implemented: implemented,
    evolvesFromId: evolvesFromId,
    breeding: BreedingInfo(
      id: variants.first.id,
      petId: variants.first.petId,
      name: variants.first.name,
      modelId: variants.first.modelId,
      hatchData: variants.first.hatchData,
      weightLow: variants.first.weightLow,
      weightHigh: variants.first.weightHigh,
      heightLow: variants.first.heightLow,
      heightHigh: variants.first.heightHigh,
      variants: variants,
    ),
  );
}

BreedingVariant _variant(
  double heightLow,
  double heightHigh,
  double weightLow,
  double weightHigh,
) {
  return BreedingVariant(
    id: 1,
    petId: 1,
    name: 'variant',
    modelId: 1,
    hatchData: 57600,
    weightLow: weightLow,
    weightHigh: weightHigh,
    heightLow: heightLow,
    heightHigh: heightHigh,
  );
}
