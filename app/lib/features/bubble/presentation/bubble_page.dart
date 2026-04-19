import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:window_manager/window_manager.dart';

import '../../../shared/desktop_window_controller.dart';
import '../../../shared/providers.dart';

class BubblePage extends ConsumerStatefulWidget {
  const BubblePage({super.key});

  static const double _hitAreaSize = 60;
  static const double _badgeSize = 14;

  @override
  ConsumerState<BubblePage> createState() => _BubblePageState();
}

class _BubblePageState extends ConsumerState<BubblePage> {
  Future<void> _expand({
    required BuildContext context,
    required DesktopWindowController controller,
    required bool hasResult,
  }) async {
    await controller.transitionToFull(() {
      if (context.mounted) {
        context.go(hasResult ? controller.restoreRoute : '/');
      }
    });
  }

  Future<void> _showBubbleMenu(
    BuildContext context, {
    required TapDownDetails details,
    required DesktopWindowController controller,
    required bool hasResult,
  }) async {
    final overlay = Overlay.of(context).context.findRenderObject() as RenderBox;
    final selected = await showMenu<_BubbleMenuAction>(
      context: context,
      position: RelativeRect.fromRect(
        Rect.fromLTWH(
          details.globalPosition.dx,
          details.globalPosition.dy,
          1,
          1,
        ),
        Offset.zero & overlay.size,
      ),
      items: [
        PopupMenuItem(
          value: _BubbleMenuAction.open,
          child: Text(hasResult ? '打开结果' : '打开主界面'),
        ),
        const PopupMenuItem(
          value: _BubbleMenuAction.exit,
          child: Text('退出程序'),
        ),
      ],
    );
    if (!mounted || !context.mounted || selected == null) {
      return;
    }
    switch (selected) {
      case _BubbleMenuAction.open:
        await _expand(
          context: context,
          controller: controller,
          hasResult: hasResult,
        );
      case _BubbleMenuAction.exit:
        await controller.exitApplication();
    }
  }

  @override
  Widget build(BuildContext context) {
    final controller = ref.read(desktopWindowControllerProvider);
    final hasResult = ref.watch(currentAnalysisResultProvider) != null;

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
              onTap: () => _expand(
                context: context,
                controller: controller,
                hasResult: hasResult,
              ),
              onSecondaryTapDown: (details) => _showBubbleMenu(
                context,
                details: details,
                controller: controller,
                hasResult: hasResult,
              ),
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

enum _BubbleMenuAction {
  open,
  exit,
}
