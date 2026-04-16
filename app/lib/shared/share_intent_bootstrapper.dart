import 'dart:async';
import 'dart:io';

import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:share_handler/share_handler.dart';

import 'providers.dart';
import '../app/router.dart';

final shareIntentBootstrapperProvider =
    Provider<ShareIntentBootstrapper>((ref) {
  final bootstrapper = ShareIntentBootstrapper(ref);
  ref.onDispose(bootstrapper.dispose);
  return bootstrapper;
});

class ShareIntentBootstrapper {
  ShareIntentBootstrapper(this.ref);

  final Ref ref;
  StreamSubscription<SharedMedia>? _subscription;
  bool _started = false;

  Future<void> start() async {
    if (_started || !Platform.isAndroid) {
      return;
    }
    _started = true;

    final handler = ShareHandler.instance;
    final initial = await handler.getInitialSharedMedia();
    if (initial != null) {
      await _handle(initial);
    }

    _subscription = handler.sharedMediaStream.listen((media) {
      unawaited(_handle(media));
    });
  }

  Future<void> _handle(SharedMedia media) async {
    final attachments = (media.attachments ?? const <SharedAttachment?>[])
        .whereType<SharedAttachment>()
        .where((item) => item.path.isNotEmpty)
        .toList(growable: false);
    final first = attachments.isEmpty ? null : attachments.first;
    if (first == null) {
      return;
    }

    final file = File(first.path);
    if (!await file.exists()) {
      return;
    }

    final bytes = await file.readAsBytes();
    await ref.read(analysisControllerProvider).analyzeBytes(
          bytes,
          label: first.path.split('/').last,
        );
    ref.read(appRouterProvider).go('/result');
  }

  void dispose() {
    _subscription?.cancel();
  }
}
