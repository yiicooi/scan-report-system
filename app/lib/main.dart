import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'pages/login_page.dart';
import 'pages/main_page.dart';
import 'providers/auth_provider.dart';

void main() {
  runApp(const ProviderScope(child: ScanReportApp()));
}

class ScanReportApp extends ConsumerWidget {
  const ScanReportApp({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final authAsync = ref.watch(authProvider);
    return MaterialApp(
      title: '扫码报工',
      debugShowCheckedModeBanner: false,
      theme: ThemeData(
        colorScheme: ColorScheme.fromSeed(seedColor: Colors.blue),
        useMaterial3: true,
      ),
      home: authAsync.when(
        // 初始化中：显示启动屏，避免闪现登录页
        loading: () => const Scaffold(
          backgroundColor: Color(0xFF1565C0),
          body: Center(
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                Icon(Icons.qr_code_scanner, size: 64, color: Colors.white),
                SizedBox(height: 16),
                Text('扫码报工',
                    style: TextStyle(
                        color: Colors.white,
                        fontSize: 22,
                        fontWeight: FontWeight.bold)),
                SizedBox(height: 32),
                CircularProgressIndicator(color: Colors.white54),
              ],
            ),
          ),
        ),
        error: (_, __) => const LoginPage(),
        data: (s) => s.isLoggedIn ? const MainPage() : const LoginPage(),
      ),
    );
  }
}

