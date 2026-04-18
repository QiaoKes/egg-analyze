import 'dart:typed_data';
import 'dart:math' as math;

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'providers.dart';

final portraitBytesProvider =
    FutureProvider.family<Uint8List?, String>((ref, portraitKey) {
  return ref.read(portraitRepositoryProvider).loadPortraitBytes(portraitKey);
});

class PortraitImage extends ConsumerWidget {
  const PortraitImage({
    super.key,
    required this.portraitKey,
    required this.label,
    this.size = 96,
  });

  final String portraitKey;
  final String label;
  final double size;

  double get _portraitHeight => size * 1.18;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final image = ref.watch(portraitBytesProvider(portraitKey));
    return ClipRRect(
      borderRadius: BorderRadius.circular(18),
      child: Container(
        width: size,
        height: _portraitHeight,
        color: const Color(0xFFE8EEF9),
        child: image.when(
          data: (bytes) {
            if (bytes == null) {
              return _Fallback(label: label);
            }
            return Padding(
              padding: EdgeInsets.fromLTRB(
                size * 0.04,
                size * 0.05,
                size * 0.04,
                size * 0.02,
              ),
              child: Image.memory(
                bytes,
                fit: BoxFit.contain,
                alignment: Alignment.bottomCenter,
              ),
            );
          },
          error: (_, __) => _Fallback(label: label),
          loading: () =>
              const Center(child: CircularProgressIndicator.adaptive()),
        ),
      ),
    );
  }
}

class _Fallback extends StatelessWidget {
  const _Fallback({required this.label});

  final String label;

  @override
  Widget build(BuildContext context) {
    final trimmed = label.trim();
    final text = trimmed.isEmpty
        ? '?'
        : trimmed.substring(0, math.min(2, trimmed.length));
    return Center(
      child: Text(
        text,
        style: Theme.of(context).textTheme.titleMedium,
      ),
    );
  }
}
