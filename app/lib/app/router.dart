import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../features/bubble/presentation/bubble_page.dart';
import '../features/home/presentation/home_page.dart';
import '../features/benchmark/presentation/benchmark_page.dart';
import '../features/result/presentation/result_page.dart';
import '../features/settings/presentation/settings_page.dart';

final appRouterProvider = Provider<GoRouter>((ref) {
  return GoRouter(
    initialLocation: '/',
    routes: [
      GoRoute(
        path: '/bubble',
        builder: (context, state) => const BubblePage(),
      ),
      GoRoute(
        path: '/',
        builder: (context, state) => const HomePage(),
      ),
      GoRoute(
        path: '/result',
        builder: (context, state) => const ResultPage(),
      ),
      GoRoute(
        path: '/settings',
        builder: (context, state) => const SettingsPage(),
      ),
      GoRoute(
        path: '/benchmark',
        builder: (context, state) => const BenchmarkPage(),
      ),
    ],
  );
});
