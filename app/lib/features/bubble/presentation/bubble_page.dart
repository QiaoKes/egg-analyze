import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:window_manager/window_manager.dart';

import '../../../shared/providers.dart';

class BubblePage extends ConsumerWidget {
  const BubblePage({super.key});

  static const double _hitAreaSize = 60;
  static const double _badgeSize = 14;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final controller = ref.read(desktopWindowControllerProvider);
    final hasResult = ref.watch(currentAnalysisResultProvider) != null;

    Future<void> expand() async {
      await controller.transitionToFull(() {
        if (context.mounted) {
          context.go(hasResult ? '/result' : controller.restoreRoute);
        }
      });
    }

    return Material(
      type: MaterialType.transparency,
      child: Center(
        child: Semantics(
          label: '打开分析面板',
          button: true,
          child: MouseRegion(
            cursor: SystemMouseCursors.click,
            child: GestureDetector(
              behavior: HitTestBehavior.opaque,
              onTap: expand,
              onPanStart: (_) {
                windowManager.startDragging();
              },
              child: SizedBox(
                width: BubblePage._hitAreaSize,
                height: BubblePage._hitAreaSize,
                child: Stack(
                  clipBehavior: Clip.none,
                  children: [
                    Positioned.fill(
                      child: ClipRRect(
                        borderRadius: BorderRadius.circular(18),
                        child: Image.asset(
                          'assets/ui/bubble.png',
                          fit: BoxFit.cover,
                          filterQuality: FilterQuality.high,
                        ),
                      ),
                    ),
                    if (hasResult)
                      Positioned(
                        right: 2,
                        top: 2,
                        child: Container(
                          width: BubblePage._badgeSize,
                          height: BubblePage._badgeSize,
                          decoration: BoxDecoration(
                            color: const Color(0xFFFFB703),
                            borderRadius:
                                BorderRadius.circular(BubblePage._badgeSize / 2),
                            border: Border.all(
                              color: Colors.white,
                              width: 2,
                            ),
                          ),
                          child: const Icon(
                            Icons.bolt_rounded,
                            size: 7,
                            color: Color(0xFF5B3500),
                          ),
                        ),
                      ),
                  ],
                ),
              ),
            ),
          ),
        ),
      ),
    );
  }
}
