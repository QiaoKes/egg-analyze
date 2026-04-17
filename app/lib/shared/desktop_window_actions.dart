import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import 'providers.dart';

List<Widget> buildDesktopWindowActions(
  BuildContext context,
  WidgetRef ref, {
  required String currentRoute,
  List<Widget> trailing = const [],
}) {
  final controller = ref.read(desktopWindowControllerProvider);
  controller.registerRoute(currentRoute);

  if (!controller.isDesktop) {
    return trailing;
  }
  if (controller.isBubbleMode) {
    return trailing;
  }

  return [
    Semantics(
      label: '缩为悬浮球',
      button: true,
      child: Tooltip(
        message: '缩为悬浮球',
        child: IconButton(
          onPressed: () async {
            await controller.enterBubbleMode();
            if (context.mounted) {
              context.go('/bubble');
            }
          },
          icon: const Icon(Icons.circle_outlined),
        ),
      ),
    ),
    ...trailing,
  ];
}
