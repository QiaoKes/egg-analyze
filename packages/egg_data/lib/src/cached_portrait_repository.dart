import 'dart:typed_data';

import 'package:dio/dio.dart';
import 'package:image/image.dart' as img;
import 'package:path/path.dart' as p;
import 'package:path_provider/path_provider.dart';
import 'dart:io';

class CachedPortraitRepository {
  CachedPortraitRepository({Dio? dio}) : _dio = dio ?? Dio();

  static const atlasBaseUrl = 'https://rocom.aoe.top/assets/webp/friends';

  final Dio _dio;

  Future<Uint8List?> loadPortraitBytes(String portraitKey) async {
    final normalized = _normalizeKey(portraitKey);
    if (normalized.isEmpty) {
      return null;
    }

    final file = await _cacheFile(normalized);
    if (await file.exists()) {
      return file.readAsBytes();
    }

    final response = await _dio.get<List<int>>(
      '$atlasBaseUrl/JL_$normalized.webp',
      options: Options(responseType: ResponseType.bytes),
    );
    final contentType =
        (response.headers.value(Headers.contentTypeHeader) ?? '').toLowerCase();
    final bytes = Uint8List.fromList(response.data ?? const <int>[]);

    if (!contentType.startsWith('image/') || img.decodeImage(bytes) == null) {
      return null;
    }

    await file.parent.create(recursive: true);
    await file.writeAsBytes(bytes, flush: true);
    return bytes;
  }

  Future<void> clearCache() async {
    final root = await getApplicationSupportDirectory();
    final dir = Directory(p.join(root.path, 'friend-atlas'));
    if (await dir.exists()) {
      await dir.delete(recursive: true);
    }
  }

  Future<File> _cacheFile(String normalized) async {
    final root = await getApplicationSupportDirectory();
    return File(p.join(root.path, 'friend-atlas', 'JL_$normalized.webp'));
  }

  String _normalizeKey(String value) {
    return value
        .trim()
        .replaceFirst(RegExp(r'^JL_'), '')
        .replaceFirst('.webp', '');
  }
}
