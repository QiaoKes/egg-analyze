import 'dart:convert';

import 'package:egg_core/egg_core.dart';
import 'package:shared_preferences/shared_preferences.dart';

class RecentAnalysisStore {
  static const _prefsKey = 'recent_analysis_records';

  Future<List<RecentAnalysisRecord>> load() async {
    final prefs = await SharedPreferences.getInstance();
    final raw = prefs.getStringList(_prefsKey) ?? const <String>[];
    return raw
        .map((item) => RecentAnalysisRecord.fromJson(
            jsonDecode(item) as Map<String, dynamic>))
        .toList(growable: false);
  }

  Future<void> add(RecentAnalysisRecord record) async {
    final prefs = await SharedPreferences.getInstance();
    final existing = await load();
    final next = [
      record,
      ...existing.where((item) => item.id != record.id),
    ].take(10).toList(growable: false);
    await prefs.setStringList(
      _prefsKey,
      next.map((item) => jsonEncode(item.toJson())).toList(growable: false),
    );
  }

  Future<void> clear() async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.remove(_prefsKey);
  }
}
