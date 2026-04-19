import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../shared/providers.dart';
import '../shared/share_intent_bootstrapper.dart';
import 'router.dart';
import 'theme.dart';

class EggAnalyzeV2App extends ConsumerStatefulWidget {
  const EggAnalyzeV2App({super.key});

  @override
  ConsumerState<EggAnalyzeV2App> createState() => _EggAnalyzeV2AppState();
}

class _EggAnalyzeV2AppState extends ConsumerState<EggAnalyzeV2App> {
  bool _desktopBootstrapped = false;

  @override
  void initState() {
    super.initState();
    Future.microtask(() => ref.read(shareIntentBootstrapperProvider).start());
    Future.microtask(() async {
      final controller = ref.read(desktopWindowControllerProvider);
      await controller.initialize();
      if (!_desktopBootstrapped && controller.isDesktop && controller.isBubbleMode) {
        _desktopBootstrapped = true;
        ref.read(appRouterProvider).go('/bubble');
      }
    });
  }

  @override
  Widget build(BuildContext context) {
    final router = ref.watch(appRouterProvider);
    return MaterialApp.router(
      title: '洛克王国精灵蛋分析',
      theme: buildAppTheme(),
      debugShowCheckedModeBanner: false,
      routerConfig: router,
    );
  }
}
