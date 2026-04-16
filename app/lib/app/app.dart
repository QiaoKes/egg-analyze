import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../shared/share_intent_bootstrapper.dart';
import 'router.dart';
import 'theme.dart';

class EggAnalyzeV2App extends ConsumerStatefulWidget {
  const EggAnalyzeV2App({super.key});

  @override
  ConsumerState<EggAnalyzeV2App> createState() => _EggAnalyzeV2AppState();
}

class _EggAnalyzeV2AppState extends ConsumerState<EggAnalyzeV2App> {
  @override
  void initState() {
    super.initState();
    Future.microtask(() => ref.read(shareIntentBootstrapperProvider).start());
  }

  @override
  Widget build(BuildContext context) {
    final router = ref.watch(appRouterProvider);
    return MaterialApp.router(
      title: 'Egg Analyze v2',
      theme: buildAppTheme(),
      routerConfig: router,
    );
  }
}
