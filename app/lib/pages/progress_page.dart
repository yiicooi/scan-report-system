import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../providers/progress_provider.dart';

class ProgressPage extends ConsumerStatefulWidget {
  const ProgressPage({super.key});

  @override
  ConsumerState<ProgressPage> createState() => _ProgressPageState();
}

class _ProgressPageState extends ConsumerState<ProgressPage> {
  final _ctrl = TextEditingController();
  String? _orderId;

  @override
  void dispose() {
    _ctrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('订单进度'),
        backgroundColor: const Color(0xFF1565C0),
        foregroundColor: Colors.white,
      ),
      body: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          children: [
            Row(
              children: [
                Expanded(
                  child: TextField(
                    controller: _ctrl,
                    decoration: const InputDecoration(
                      labelText: '输入订单号',
                      border: OutlineInputBorder(),
                    ),
                  ),
                ),
                const SizedBox(width: 12),
                ElevatedButton(
                  onPressed: () => setState(() => _orderId = _ctrl.text.trim()),
                  style: ElevatedButton.styleFrom(
                    backgroundColor: const Color(0xFF1565C0),
                    foregroundColor: Colors.white,
                    padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 14),
                  ),
                  child: const Text('查询'),
                ),
              ],
            ),
            const SizedBox(height: 16),
            if (_orderId != null && _orderId!.isNotEmpty)
              Expanded(child: _ProgressView(orderId: _orderId!)),
          ],
        ),
      ),
    );
  }
}

class _ProgressView extends ConsumerWidget {
  final String orderId;
  const _ProgressView({required this.orderId});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final async = ref.watch(orderProgressProvider(orderId));
    return async.when(
      loading: () => const Center(child: CircularProgressIndicator()),
      error: (e, _) => Center(child: Text('查询失败：$e')),
      data: (data) {
        final processes = (data['processes'] as List?) ?? [];
        return ListView(
          children: [
            Card(
              child: Padding(
                padding: const EdgeInsets.all(16),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(data['internal_no'] ?? '', style: const TextStyle(fontSize: 18, fontWeight: FontWeight.bold)),
                    Text('产品：${data['product_name'] ?? ''}'),
                    Text('总数：${data['total_qty'] ?? 0} | 已完成：${data['total_completed'] ?? 0}'),
                    const SizedBox(height: 8),
                    LinearProgressIndicator(
                      value: (data['progress_pct'] ?? 0) / 100.0,
                      backgroundColor: Colors.grey[200],
                      color: const Color(0xFF1565C0),
                      minHeight: 8,
                    ),
                    const SizedBox(height: 4),
                    Text('${data['progress_pct'] ?? 0}%', style: const TextStyle(color: Colors.grey)),
                  ],
                ),
              ),
            ),
            ...processes.map((p) => _ProcessProgressCard(p: p as Map<String, dynamic>)),
          ],
        );
      },
    );
  }
}

class _ProcessProgressCard extends StatelessWidget {
  final Map<String, dynamic> p;
  const _ProcessProgressCard({required this.p});

  @override
  Widget build(BuildContext context) {
    final pct = (p['summary']?['progress_pct'] ?? 0) as num;
    return Card(
      margin: const EdgeInsets.symmetric(vertical: 4),
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              mainAxisAlignment: MainAxisAlignment.spaceBetween,
              children: [
                Text(p['process_name'] ?? '', style: const TextStyle(fontWeight: FontWeight.bold)),
                _statusChip(p['status'] ?? ''),
              ],
            ),
            const SizedBox(height: 6),
            LinearProgressIndicator(
              value: pct / 100.0,
              backgroundColor: Colors.grey[200],
              color: Colors.green,
              minHeight: 6,
            ),
            const SizedBox(height: 4),
            Text(
              '完成 ${p['summary']?['total_completed'] ?? 0} / ${p['total_qty'] ?? 0}  ($pct%)',
              style: const TextStyle(fontSize: 12, color: Colors.grey),
            ),
          ],
        ),
      ),
    );
  }

  Widget _statusChip(String status) {
    Color color;
    String label;
    switch (status) {
      case 'completed':
        color = Colors.green;
        label = '已完成';
        break;
      case 'in_progress':
        color = Colors.blue;
        label = '进行中';
        break;
      case 'pending':
        color = Colors.grey;
        label = '待处理';
        break;
      default:
        color = Colors.grey;
        label = status;
    }
    return Chip(
      label: Text(label, style: const TextStyle(color: Colors.white, fontSize: 12)),
      backgroundColor: color,
      materialTapTargetSize: MaterialTapTargetSize.shrinkWrap,
    );
  }
}
