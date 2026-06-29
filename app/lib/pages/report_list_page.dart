import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:intl/intl.dart';
import '../providers/report_provider.dart';

class ReportListPage extends ConsumerStatefulWidget {
  const ReportListPage({super.key});

  @override
  ConsumerState<ReportListPage> createState() => _ReportListPageState();
}

class _ReportListPageState extends ConsumerState<ReportListPage> {
  int _page = 1;
  DateTime? _startDate;
  DateTime? _endDate;

  Map<String, dynamic> get _params => {
    'page': _page,
    'page_size': 20,
    if (_startDate != null) 'start_date': DateFormat('yyyy-MM-dd').format(_startDate!),
    if (_endDate != null) 'end_date': DateFormat('yyyy-MM-dd').format(_endDate!),
  };

  /// 将 params 序列化为排序后的字符串，作为 provider family 的稳定 key
  String get _paramsKey {
    final entries = _params.entries.toList()..sort((a, b) => a.key.compareTo(b.key));
    return entries.map((e) => '${e.key}=${e.value}').join('&');
  }

  @override
  Widget build(BuildContext context) {
    final historyAsync = ref.watch(reportHistoryProvider(_paramsKey));

    return Scaffold(
      appBar: AppBar(
        title: const Text('报工记录'),
        backgroundColor: const Color(0xFF1565C0),
        foregroundColor: Colors.white,
        actions: [
          IconButton(
            icon: const Icon(Icons.filter_list),
            onPressed: _showFilter,
          ),
        ],
      ),
      body: historyAsync.when(
        loading: () => const Center(child: CircularProgressIndicator()),
        error: (e, _) => Center(child: Text('加载失败：$e')),
        data: (list) => list.isEmpty
            ? const Center(child: Text('暂无记录'))
            : ListView.builder(
                itemCount: list.length,
                itemBuilder: (ctx, i) => _ReportCard(item: list[i] as Map<String, dynamic>),
              ),
      ),
    );
  }

  Future<void> _showFilter() async {
    await showModalBottomSheet(
      context: context,
      builder: (ctx) => _FilterSheet(
        startDate: _startDate,
        endDate: _endDate,
        onApply: (start, end) {
          setState(() {
            _startDate = start;
            _endDate = end;
            _page = 1;
          });
        },
      ),
    );
  }
}

class _ReportCard extends StatelessWidget {
  final Map<String, dynamic> item;
  const _ReportCard({required this.item});

  @override
  Widget build(BuildContext context) {
    final detail = (item['detail'] as Map?) ?? {};
    return Card(
      margin: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              mainAxisAlignment: MainAxisAlignment.spaceBetween,
              children: [
                Text(item['order_no'] ?? '', style: const TextStyle(fontWeight: FontWeight.bold)),
                Text(item['process_name'] ?? '', style: const TextStyle(color: Colors.grey)),
              ],
            ),
            const Divider(),
            Row(
              mainAxisAlignment: MainAxisAlignment.spaceAround,
              children: [
                _QtyChip(label: '投入', value: detail['received_qty']?.toString() ?? '0'),
                _QtyChip(label: '完成', value: detail['completed_qty']?.toString() ?? '0', color: Colors.green),
                _QtyChip(label: '报废', value: detail['scrap_qty']?.toString() ?? '0', color: Colors.red),
              ],
            ),
            const SizedBox(height: 4),
            Text(
              item['created_at'] != null
                  ? DateFormat('yyyy-MM-dd HH:mm').format(DateTime.parse(item['created_at']))
                  : '',
              style: const TextStyle(color: Colors.grey, fontSize: 12),
            ),
          ],
        ),
      ),
    );
  }
}

class _QtyChip extends StatelessWidget {
  final String label;
  final String value;
  final Color color;
  const _QtyChip({required this.label, required this.value, this.color = Colors.blue});

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        Text(value, style: TextStyle(fontSize: 20, fontWeight: FontWeight.bold, color: color)),
        Text(label, style: const TextStyle(fontSize: 12, color: Colors.grey)),
      ],
    );
  }
}

class _FilterSheet extends StatefulWidget {
  final DateTime? startDate;
  final DateTime? endDate;
  final void Function(DateTime? start, DateTime? end) onApply;

  const _FilterSheet({this.startDate, this.endDate, required this.onApply});

  @override
  State<_FilterSheet> createState() => _FilterSheetState();
}

class _FilterSheetState extends State<_FilterSheet> {
  DateTime? _start;
  DateTime? _end;

  @override
  void initState() {
    super.initState();
    _start = widget.startDate;
    _end = widget.endDate;
  }

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.all(24),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          const Text('筛选日期', style: TextStyle(fontSize: 18, fontWeight: FontWeight.bold)),
          const SizedBox(height: 16),
          Row(
            children: [
              Expanded(child: _datePicker('开始日期', _start, (d) => setState(() => _start = d))),
              const SizedBox(width: 16),
              Expanded(child: _datePicker('结束日期', _end, (d) => setState(() => _end = d))),
            ],
          ),
          const SizedBox(height: 24),
          Row(
            children: [
              TextButton(
                onPressed: () {
                  widget.onApply(null, null);
                  Navigator.pop(context);
                },
                child: const Text('清除'),
              ),
              const Spacer(),
              ElevatedButton(
                onPressed: () {
                  widget.onApply(_start, _end);
                  Navigator.pop(context);
                },
                child: const Text('确定'),
              ),
            ],
          ),
        ],
      ),
    );
  }

  Widget _datePicker(String label, DateTime? value, ValueChanged<DateTime?> onPick) {
    return InkWell(
      onTap: () async {
        final picked = await showDatePicker(
          context: context,
          initialDate: value ?? DateTime.now(),
          firstDate: DateTime(2020),
          lastDate: DateTime.now().add(const Duration(days: 365)),
        );
        onPick(picked);
      },
      child: InputDecorator(
        decoration: InputDecoration(labelText: label, border: const OutlineInputBorder()),
        child: Text(value != null ? DateFormat('yyyy-MM-dd').format(value) : '请选择'),
      ),
    );
  }
}
