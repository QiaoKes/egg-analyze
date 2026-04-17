import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:window_manager/window_manager.dart';

import '../../../shared/providers.dart';

class BubblePage extends ConsumerWidget {
  const BubblePage({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final controller = ref.read(desktopWindowControllerProvider);
    final hasResult = ref.watch(currentAnalysisResultProvider) != null;

    Future<void> expand() async {
      await controller.exitBubbleMode();
      if (!context.mounted) {
        return;
      }
      context.go(hasResult ? controller.restoreRoute : '/');
    }

    return Scaffold(
      backgroundColor: Colors.transparent,
      body: Center(
        child: Stack(
          clipBehavior: Clip.none,
          children: [
            const SizedBox(width: 68, height: 68),
            Positioned.fill(
              child: DragToMoveArea(
                child: Container(color: Colors.transparent),
              ),
            ),
            Center(
              child: Tooltip(
                message: hasResult ? '打开最近分析结果' : '打开分析面板',
                child: SizedBox(
                  width: 60,
                  height: 60,
                  child: Semantics(
                    label: '打开分析面板',
                    button: true,
                    child: FilledButton(
                      onPressed: expand,
                      style: FilledButton.styleFrom(
                        shape: const CircleBorder(),
                        padding: EdgeInsets.zero,
                        backgroundColor: Colors.transparent,
                        shadowColor: const Color(0x40294AA7),
                        elevation: 10,
                      ),
                      child: Ink(
                        decoration: const BoxDecoration(
                          shape: BoxShape.circle,
                          image: DecorationImage(
                            image: AssetImage('assets/ui/bubble.png'),
                            fit: BoxFit.cover,
                          ),
                        ),
                        child: Stack(
                          clipBehavior: Clip.none,
                          children: [
                            const SizedBox(width: 60, height: 60),
                            if (hasResult)
                              Positioned(
                                right: 2,
                                top: 2,
                                child: Container(
                                  width: 14,
                                  height: 14,
                                  decoration: BoxDecoration(
                                    color: const Color(0xFFFFB703),
                                    borderRadius: BorderRadius.circular(7),
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
            ),
          ],
        ),
      ),
    );
  }
}
