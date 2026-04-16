import 'dart:convert';
import 'dart:io';

import 'package:flutter_test/flutter_test.dart';

void main() {
  test('benchmark fixture manifests are available', () async {
    final file = File('test_fixtures/benchmarks.json');

    expect(await file.exists(), isTrue);

    final entries = jsonDecode(await file.readAsString()) as List<dynamic>;

    expect(entries, isNotEmpty);
  });
}
