import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'scan_report_page.dart';
import 'report_list_page.dart';
import 'progress_page.dart';
import 'ai_chat_page.dart';
import 'more_page.dart';

class MainPage extends ConsumerStatefulWidget {
  const MainPage({super.key});

  @override
  ConsumerState<MainPage> createState() => _MainPageState();
}

class _MainPageState extends ConsumerState<MainPage> {
  int _index = 0;

  static const _pages = [
    ScanReportPage(),
    ReportListPage(),
    ProgressPage(),
    AiChatPage(),
    MorePage(),
  ];

  static const _items = [
    BottomNavigationBarItem(icon: Icon(Icons.qr_code_scanner), label: '扫码报工'),
    BottomNavigationBarItem(icon: Icon(Icons.list_alt), label: '报工记录'),
    BottomNavigationBarItem(icon: Icon(Icons.bar_chart), label: '进度'),
    BottomNavigationBarItem(icon: Icon(Icons.smart_toy), label: 'AI查询'),
    BottomNavigationBarItem(icon: Icon(Icons.more_horiz), label: '更多'),
  ];

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: IndexedStack(index: _index, children: _pages),
      bottomNavigationBar: BottomNavigationBar(
        currentIndex: _index,
        onTap: (i) => setState(() => _index = i),
        type: BottomNavigationBarType.fixed,
        selectedItemColor: const Color(0xFF1565C0),
        items: _items,
      ),
    );
  }
}
