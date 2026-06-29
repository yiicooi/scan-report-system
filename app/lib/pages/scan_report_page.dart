import 'dart:io';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:mobile_scanner/mobile_scanner.dart';
import 'package:image_picker/image_picker.dart';
import 'package:intl/intl.dart';
import '../providers/scan_provider.dart';
import '../providers/auth_provider.dart';
import '../providers/report_provider.dart';
import '../services/api_service.dart';

class ScanReportPage extends ConsumerStatefulWidget {
  const ScanReportPage({super.key});

  @override
  ConsumerState<ScanReportPage> createState() => _ScanReportPageState();
}

class _ScanReportPageState extends ConsumerState<ScanReportPage> {
  Map<String, dynamic>? _selectedProcess;

  final _receivedCtrl = TextEditingController();
  final _completedCtrl = TextEditingController();
  final _scrapCtrl = TextEditingController();
  List<File> _receivedImages = [];
  List<File> _completedImages = [];
  List<File> _scrapImages = [];

  @override
  void dispose() {
    _receivedCtrl.dispose();
    _completedCtrl.dispose();
    _scrapCtrl.dispose();
    super.dispose();
  }

  void _resetForm() {
    _receivedCtrl.clear();
    _completedCtrl.clear();
    _scrapCtrl.clear();
    setState(() {
      _selectedProcess = null;
      _receivedImages = [];
      _completedImages = [];
      _scrapImages = [];
    });
  }

  Future<void> _openScanner() async {
    ref.read(scanProvider.notifier).clear();
    _resetForm();
    await showModalBottomSheet(
      context: context,
      isScrollControlled: true,
      backgroundColor: Colors.black,
      builder: (_) => SizedBox(
        height: MediaQuery.of(context).size.height * 0.65,
        child: Stack(
          children: [
            MobileScanner(
              onDetect: (capture) {
                final code = capture.barcodes.first.rawValue;
                if (code != null) {
                  Navigator.pop(context);
                  ref.read(scanProvider.notifier).scan(code);
                }
              },
            ),
            Center(
              child: Container(
                width: 220,
                height: 220,
                decoration: BoxDecoration(
                  border: Border.all(color: const Color(0xFF2196F3), width: 3),
                  borderRadius: BorderRadius.circular(8),
                ),
              ),
            ),
            const Positioned(
              top: 16,
              left: 0,
              right: 0,
              child: Center(
                child: Text('将二维码置于框内扫描',
                    style: TextStyle(color: Colors.white, fontSize: 14)),
              ),
            ),
            Positioned(
              bottom: 24,
              left: 0,
              right: 0,
              child: Center(
                child: TextButton.icon(
                  onPressed: () => Navigator.pop(context),
                  icon: const Icon(Icons.close, color: Colors.white),
                  label: const Text('取消',
                      style: TextStyle(color: Colors.white)),
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }

  Future<void> _submitReport() async {
    final order = ref.read(scanProvider).order;
    if (order == null || _selectedProcess == null) return;

    final r = int.tryParse(_receivedCtrl.text) ?? 0;
    final c = int.tryParse(_completedCtrl.text) ?? 0;
    final s = int.tryParse(_scrapCtrl.text) ?? 0;
    if (r == 0 && c == 0 && s == 0) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(
            content: Text('请至少填写一项数量'),
            backgroundColor: Colors.orange),
      );
      return;
    }

    Future<List<String>> uploadImages(List<File> files, String type) async {
      final urls = <String>[];
      for (final f in files) {
        final ext = f.path.split('.').last;
        final key =
            ApiService.generateObjectKey(order['id'].toString(), type, ext);
        final presign = await ApiService.presignUpload(key, ext);
        await ApiService.uploadToOSS(presign['url'], f);
        urls.add(presign['object_url']);
      }
      return urls;
    }

    final receivedUrls = await uploadImages(_receivedImages, 'received');
    final completedUrls = await uploadImages(_completedImages, 'completed');
    final scrapUrls = await uploadImages(_scrapImages, 'scrap');

    final payload = {
      'order_id': order['id'],
      'order_process_id': _selectedProcess!['id'],
      'received_qty': r,
      'completed_qty': c,
      'scrap_qty': s,
      'received_images': receivedUrls,
      'completed_images': completedUrls,
      'scrap_images': scrapUrls,
    };

    await ref.read(reportSubmitProvider.notifier).submit(payload);
    if (!mounted) return;
    final submitState = ref.read(reportSubmitProvider);
    if (submitState.success) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(
            content: Text('报工成功'), backgroundColor: Colors.green),
      );
      ref.read(reportSubmitProvider.notifier).reset();
      // 保留当前页面，重新拉取最新工单数据（刷新统计数字）
      final internalNo = order['internal_no'] as String?;
      if (internalNo != null) {
        _receivedCtrl.clear();
        _completedCtrl.clear();
        _scrapCtrl.clear();
        setState(() {
          _receivedImages = [];
          _completedImages = [];
          _scrapImages = [];
        });
        await ref.read(scanProvider.notifier).scan(internalNo);
        // 刷新后重新匹配已选工序
        if (mounted) {
          final newOrder = ref.read(scanProvider).order;
          final newProcesses = (newOrder?['processes'] as List?) ?? [];
          final matched = newProcesses.cast<Map<String, dynamic>>().where(
            (p) => p['id'] == _selectedProcess?['id']).toList();
          setState(() {
            _selectedProcess = matched.isNotEmpty ? matched.first : null;
          });
        }
      }
    } else if (submitState.error != null) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
            content: Text('提交失败：${submitState.error}'),
            backgroundColor: Colors.red),
      );
      ref.read(reportSubmitProvider.notifier).reset();
    }
  }

  @override
  Widget build(BuildContext context) {
    final scanState = ref.watch(scanProvider);
    final submitState = ref.watch(reportSubmitProvider);
    final authState = ref.watch(authProvider).asData?.value;
    final userName = authState?.user?['username'] ??
        authState?.user?['name'] ??
        '';

    final order = scanState.order;
    // 全部工序（用于工艺流程展示和上道工序统计）
    final allProcesses = (order?['processes'] as List?) ?? [];
    // 可报工工序（can_report == true）
    final reportableProcesses = allProcesses
        .where((p) => (p as Map)['can_report'] == true)
        .cast<Map<String, dynamic>>()
        .toList();
    final summary = _selectedProcess?['summary'] as Map?;

    // 计算上道工序完成数量
    String prevCompleted = '-';
    if (_selectedProcess != null) {
      final curSort = (_selectedProcess!['sort'] as num?)?.toInt() ?? 0;
      if (curSort > 1) {
        // 找 sort 小于当前且最大的工序
        Map? prevProcess;
        int bestSort = -1;
        for (final p in allProcesses) {
          final s = ((p as Map)['sort'] as num?)?.toInt() ?? 0;
          if (s > 0 && s < curSort && s > bestSort) {
            bestSort = s;
            prevProcess = p;
          }
        }
        if (prevProcess != null) {
          final prevSummary = prevProcess['summary'] as Map?;
          final val = prevSummary?['total_completed'];
          prevCompleted = val?.toString() ?? '0';
        }
      } else if (curSort == 1) {
        prevCompleted = '首道'; // 第一道工序无上道
      }
    }

    return Scaffold(
      backgroundColor: const Color(0xFFF5F5F5),
      appBar: AppBar(
        title: const Text('扫码报工'),
        backgroundColor: const Color(0xFF1565C0),
        foregroundColor: Colors.white,
        elevation: 0,
        actions: [
          if (order != null)
            IconButton(
              icon: const Icon(Icons.refresh),
              tooltip: '重新扫描',
              onPressed: () {
                ref.read(scanProvider.notifier).clear();
                _resetForm();
              },
            ),
        ],
      ),
      body: SingleChildScrollView(
        child: Column(
          children: [
            // ── 扫码按钮 ──
            Container(
              color: Colors.white,
              padding:
                  const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
              child: SizedBox(
                width: double.infinity,
                height: 48,
                child: ElevatedButton.icon(
                  onPressed: scanState.loading ? null : _openScanner,
                  icon: scanState.loading
                      ? const SizedBox(
                          width: 18,
                          height: 18,
                          child: CircularProgressIndicator(
                              color: Colors.white, strokeWidth: 2))
                      : const Icon(Icons.add, size: 20),
                  label: Text(
                      scanState.loading ? '识别中...' : '二维码扫描',
                      style: const TextStyle(fontSize: 16)),
                  style: ElevatedButton.styleFrom(
                    backgroundColor: const Color(0xFF2196F3),
                    foregroundColor: Colors.white,
                    shape: RoundedRectangleBorder(
                        borderRadius: BorderRadius.circular(6)),
                    elevation: 0,
                  ),
                ),
              ),
            ),

            if (scanState.error != null)
              Container(
                color: const Color(0xFFFFEBEE),
                padding:
                    const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
                child: Row(
                  children: [
                    const Icon(Icons.error_outline,
                        color: Colors.red, size: 16),
                    const SizedBox(width: 8),
                    Expanded(
                        child: Text(scanState.error!,
                            style: const TextStyle(
                                color: Colors.red, fontSize: 13))),
                  ],
                ),
              ),

            const Divider(height: 1, thickness: 1),

            // ── 表单区域 ──
            Container(
              color: Colors.white,
              child: Column(
                children: [
                  _FormRow(label: '报工人员', value: userName),
                  const Divider(height: 1, indent: 16),
                  _FormRow(
                    label: '零件名称',
                    value: order?['part_name'] ?? '',
                    trailing: order != null
                        ? TextButton(
                            onPressed: () => _showReportDetail(order),
                            style: TextButton.styleFrom(
                                padding: EdgeInsets.zero,
                                minimumSize: const Size(60, 32)),
                            child: const Text('报工明细',
                                style:
                                    TextStyle(color: Color(0xFF2196F3))),
                          )
                        : null,
                  ),
                  const Divider(height: 1, indent: 16),
                  _FormRow(
                    label: '图纸编号',
                    value: order?['drawing_no'] ?? '',
                  ),
                  const Divider(height: 1, indent: 16),
                  _FormRow(
                      label: '零件编号',
                      value: order?['internal_no'] ?? ''),
                  const Divider(height: 1, indent: 16),

                  // 工序下拉（仅展示可报工工序）
                  if (order != null && reportableProcesses.isEmpty)
                    Padding(
                      padding: const EdgeInsets.symmetric(
                          horizontal: 16, vertical: 14),
                      child: Row(
                        children: const [
                          Icon(Icons.info_outline,
                              color: Colors.orange, size: 18),
                          SizedBox(width: 8),
                          Expanded(
                            child: Text(
                              '您所在部门暂无此工单的可报工工序',
                              style: TextStyle(
                                  color: Colors.orange, fontSize: 13),
                            ),
                          ),
                        ],
                      ),
                    )
                  else
                    Padding(
                      padding: const EdgeInsets.symmetric(
                          horizontal: 16, vertical: 2),
                      child: DropdownButtonFormField<Map<String, dynamic>>(
                        value: _selectedProcess,
                        decoration: const InputDecoration(
                          hintText: '请选择报工工序',
                          hintStyle: TextStyle(color: Colors.grey),
                          border: InputBorder.none,
                          isDense: true,
                          contentPadding:
                              EdgeInsets.symmetric(vertical: 12),
                        ),
                        icon: const Icon(Icons.arrow_drop_down,
                            color: Color(0xFF2196F3)),
                        items: reportableProcesses
                            .map<DropdownMenuItem<Map<String, dynamic>>>(
                                (p) => DropdownMenuItem(
                                      value: p,
                                      child: Text(p['display_name'] ??
                                          p['process_name'] ??
                                          ''),
                                    ))
                            .toList(),
                        onChanged: order == null
                            ? null
                            : (val) => setState(() {
                                  _selectedProcess = val;
                                  _receivedCtrl.clear();
                                  _completedCtrl.clear();
                                  _scrapCtrl.clear();
                                }),
                      ),
                    ),
                  const Divider(height: 1),

                  // 统计行
                  if (_selectedProcess != null) ...[
                    Padding(
                      padding: const EdgeInsets.symmetric(
                          horizontal: 16, vertical: 10),
                      child: Row(
                        children: [
                          _StatCell(
                              label: '订单数量',
                              value: order?['total_qty']?.toString() ??
                                  '-'),
                          _StatCell(
                              label: '我已完成',
                              value: summary?['total_completed']
                                      ?.toString() ??
                                  '0',
                              valueColor: const Color(0xFF2196F3)),
                          _StatCell(
                              label: '上道工序完成',
                              value: prevCompleted),
                        ],
                      ),
                    ),
                    const Divider(height: 1),
                  ],
                ],
              ),
            ),

            const SizedBox(height: 8),

            // ── 数量录入 ──
            if (_selectedProcess != null) ...[
              Container(
                color: Colors.white,
                child: Column(
                  children: [
                    _QtyInputRow(
                      totalLabel: '总接收',
                      totalValue:
                          summary?['total_received']?.toString() ?? '0',
                      inputLabel: '本次接收',
                      controller: _receivedCtrl,
                      images: _receivedImages,
                      onImagesChanged: (imgs) =>
                          setState(() => _receivedImages = imgs),
                    ),
                    const Divider(height: 1, indent: 16),
                    _QtyInputRow(
                      totalLabel: '总完成',
                      totalValue:
                          summary?['total_completed']?.toString() ?? '0',
                      inputLabel: '本次完成',
                      controller: _completedCtrl,
                      images: _completedImages,
                      onImagesChanged: (imgs) =>
                          setState(() => _completedImages = imgs),
                    ),
                    const Divider(height: 1, indent: 16),
                    _QtyInputRow(
                      totalLabel: '总报废',
                      totalValue:
                          summary?['total_scrap']?.toString() ?? '0',
                      inputLabel: '本次报废',
                      controller: _scrapCtrl,
                      images: _scrapImages,
                      onImagesChanged: (imgs) =>
                          setState(() => _scrapImages = imgs),
                    ),
                  ],
                ),
              ),

              const SizedBox(height: 16),

              // 提交按钮
              Padding(
                padding: const EdgeInsets.symmetric(horizontal: 16),
                child: SizedBox(
                  width: double.infinity,
                  height: 48,
                  child: ElevatedButton(
                    onPressed:
                        submitState.submitting ? null : _submitReport,
                    style: ElevatedButton.styleFrom(
                      backgroundColor: const Color(0xFF2196F3),
                      foregroundColor: Colors.white,
                      shape: RoundedRectangleBorder(
                          borderRadius: BorderRadius.circular(6)),
                      elevation: 0,
                    ),
                    child: submitState.submitting
                        ? const SizedBox(
                            width: 22,
                            height: 22,
                            child: CircularProgressIndicator(
                                color: Colors.white, strokeWidth: 2))
                        : const Text('提交数据',
                            style: TextStyle(fontSize: 16)),
                  ),
                ),
              ),
              const SizedBox(height: 16),
            ],

            // ── 底部工单信息 ──
            if (order != null) ...[
              Container(
                color: Colors.white,
                child: Column(
                  children: [
                    Row(
                      children: [
                        Expanded(
                            child: _FormRow(
                                label: '内部单号',
                                value: order['internal_no'] ?? '')),
                        Expanded(
                            child: _FormRow(
                                label: '外部单号',
                                value: order['external_no'] ?? '')),
                      ],
                    ),
                    const Divider(height: 1, indent: 16),
                    Padding(
                      padding: const EdgeInsets.symmetric(
                          horizontal: 16, vertical: 8),
                      child: Row(
                        children: const [
                          Text('工艺流程',
                              style: TextStyle(
                                  color: Colors.grey, fontSize: 13)),
                        ],
                      ),
                    ),
                    ..._buildProcessFlow(allProcesses),
                    const SizedBox(height: 8),
                    const Divider(height: 1, indent: 16),
                    _FormRow(
                        label: '订单日期',
                        value: order['order_date'] != null
                            ? DateFormat('yyyy-MM-dd').format(
                                DateTime.tryParse(order['order_date']) ??
                                    DateTime.now())
                            : ''),
                  ],
                ),
              ),
              const SizedBox(height: 24),
            ],
          ],
        ),
      ),
    );
  }

  String _processNames(List processes) {
    return processes
        .map((p) => p['display_name'] ?? p['process_name'] ?? '')
        .where((s) => s.isNotEmpty)
        .join(' → ');
  }

  /// 工艺流程明细行列表，每道工序占一行
  List<Widget> _buildProcessFlow(List allProcesses) {
    final List<Widget> rows = [];
    for (int i = 0; i < allProcesses.length; i++) {
      final p = allProcesses[i] as Map;
      final name = (p['display_name'] ?? p['process_name'] ?? '').toString();
      final s = (p['summary'] as Map?) ?? {};
      final r = s['total_received'];
      final c = s['total_completed'];
      final sc = s['total_scrap'];
      String fmtNum(dynamic v) => (v == null || v == 0) ? '-' : v.toString();
      final stats = '接${fmtNum(r)} / 完${fmtNum(c)} / 废${fmtNum(sc)}';
      final isLast = i == allProcesses.length - 1;
      rows.add(
        Padding(
          padding: const EdgeInsets.only(left: 16, right: 16, top: 6, bottom: 2),
          child: Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              // 符号列
              Column(
                children: [
                  Container(
                    width: 22, height: 22,
                    decoration: BoxDecoration(
                      color: const Color(0xFF2196F3),
                      shape: BoxShape.circle,
                    ),
                    child: Center(
                      child: Text('${i + 1}',
                          style: const TextStyle(
                              color: Colors.white,
                              fontSize: 11,
                              fontWeight: FontWeight.bold)),
                    ),
                  ),
                  if (!isLast)
                    Container(
                        width: 2, height: 18,
                        color: const Color(0xFFBBDEFB)),
                ],
              ),
              const SizedBox(width: 10),
              Expanded(
                child: Padding(
                  padding: const EdgeInsets.only(bottom: 4),
                  child: RichText(
                    text: TextSpan(
                      children: [
                        TextSpan(
                            text: name,
                            style: const TextStyle(
                                fontSize: 13,
                                fontWeight: FontWeight.w500,
                                color: Colors.black87)),
                        TextSpan(
                            text: '  $stats',
                            style: const TextStyle(
                                fontSize: 12,
                                color: Colors.grey)),
                      ],
                    ),
                  ),
                ),
              ),
            ],
          ),
        ),
      );
    }
    if (rows.isEmpty) {
      rows.add(const Padding(
        padding: EdgeInsets.symmetric(horizontal: 16, vertical: 8),
        child: Text('暂无工序信息', style: TextStyle(color: Colors.grey, fontSize: 13)),
      ));
    }
    return rows;
  }

  void _showReportDetail(Map<String, dynamic> order) {
    final processes = (order['processes'] as List?) ?? [];
    showModalBottomSheet(
      context: context,
      isScrollControlled: true,
      shape: const RoundedRectangleBorder(
          borderRadius: BorderRadius.vertical(top: Radius.circular(16))),
      builder: (_) => DraggableScrollableSheet(
        initialChildSize: 0.6,
        minChildSize: 0.4,
        maxChildSize: 0.9,
        expand: false,
        builder: (_, ctrl) => Column(
          children: [
            const SizedBox(height: 12),
            Container(
                width: 40,
                height: 4,
                decoration: BoxDecoration(
                    color: Colors.grey[300],
                    borderRadius: BorderRadius.circular(2))),
            const SizedBox(height: 12),
            const Text('报工明细',
                style: TextStyle(
                    fontSize: 16, fontWeight: FontWeight.bold)),
            const Divider(),
            Expanded(
              child: ListView.builder(
                controller: ctrl,
                itemCount: processes.length,
                itemBuilder: (_, i) {
                  final p = processes[i] as Map<String, dynamic>;
                  final s = (p['summary'] as Map?) ?? {};
                  final pct =
                      ((s['progress_pct'] ?? 0) as num).toStringAsFixed(1);
                  return ListTile(
                    title: Text(p['display_name'] ?? ''),
                    subtitle: Text(
                        '接收 ${s['total_received'] ?? 0}  '
                        '完成 ${s['total_completed'] ?? 0}  '
                        '报废 ${s['total_scrap'] ?? 0}'),
                    trailing: Text('$pct%',
                        style: const TextStyle(
                            color: Color(0xFF2196F3),
                            fontWeight: FontWeight.bold)),
                  );
                },
              ),
            ),
          ],
        ),
      ),
    );
  }
}

// ──────────────────────────────────────────────
// 辅助 Widgets
// ──────────────────────────────────────────────

class _FormRow extends StatelessWidget {
  final String label;
  final String value;
  final Widget? trailing;

  const _FormRow({required this.label, required this.value, this.trailing});

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
      child: Row(
        children: [
          SizedBox(
            width: 68,
            child: Text(label,
                style:
                    const TextStyle(color: Colors.grey, fontSize: 13)),
          ),
          Expanded(
            child: Text(value,
                style: const TextStyle(fontSize: 14)),
          ),
          if (trailing != null) trailing!,
        ],
      ),
    );
  }
}

class _StatCell extends StatelessWidget {
  final String label;
  final String value;
  final Color valueColor;

  const _StatCell({
    required this.label,
    required this.value,
    this.valueColor = Colors.black87,
  });

  @override
  Widget build(BuildContext context) {
    return Expanded(
      child: Column(
        children: [
          Text(value,
              style: TextStyle(
                  fontSize: 20,
                  fontWeight: FontWeight.bold,
                  color: valueColor)),
          const SizedBox(height: 2),
          Text(label,
              style:
                  const TextStyle(fontSize: 11, color: Colors.grey)),
        ],
      ),
    );
  }
}

class _QtyInputRow extends StatelessWidget {
  final String totalLabel;
  final String totalValue;
  final String inputLabel;
  final TextEditingController controller;
  final List<File> images;
  final ValueChanged<List<File>> onImagesChanged;

  const _QtyInputRow({
    required this.totalLabel,
    required this.totalValue,
    required this.inputLabel,
    required this.controller,
    required this.images,
    required this.onImagesChanged,
  });

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.center,
        children: [
          // 总量
          SizedBox(
            width: 68,
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(totalLabel,
                    style: const TextStyle(
                        color: Colors.grey, fontSize: 12)),
                const SizedBox(height: 2),
                Text(totalValue,
                    style: const TextStyle(
                        fontSize: 15, fontWeight: FontWeight.w500)),
              ],
            ),
          ),
          const SizedBox(width: 12),
          // 本次输入
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(inputLabel,
                    style: const TextStyle(
                        color: Color(0xFFE53935),
                        fontSize: 12,
                        fontWeight: FontWeight.w500)),
                TextField(
                  controller: controller,
                  keyboardType: TextInputType.number,
                  decoration: const InputDecoration(
                    hintText: '0',
                    isDense: true,
                    contentPadding: EdgeInsets.symmetric(vertical: 6),
                    border: UnderlineInputBorder(
                        borderSide:
                            BorderSide(color: Color(0xFF2196F3))),
                    focusedBorder: UnderlineInputBorder(
                        borderSide: BorderSide(
                            color: Color(0xFF2196F3), width: 2)),
                  ),
                ),
              ],
            ),
          ),
          const SizedBox(width: 12),
          // 图片按钮
          _ImgBtn(
            label: '${totalLabel.replaceAll('总', '')}图片',
            images: images,
            onChanged: onImagesChanged,
          ),
        ],
      ),
    );
  }
}

class _ImgBtn extends StatelessWidget {
  final String label;
  final List<File> images;
  final ValueChanged<List<File>> onChanged;

  const _ImgBtn(
      {required this.label,
      required this.images,
      required this.onChanged});

  Future<void> _pick(BuildContext context) async {
    final picker = ImagePicker();
    showModalBottomSheet(
      context: context,
      builder: (_) => SafeArea(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            ListTile(
              leading: const Icon(Icons.camera_alt),
              title: const Text('拍照'),
              onTap: () async {
                Navigator.pop(context);
                final xf = await picker.pickImage(
                    source: ImageSource.camera, imageQuality: 80);
                if (xf != null) onChanged([...images, File(xf.path)]);
              },
            ),
            ListTile(
              leading: const Icon(Icons.photo_library),
              title: const Text('从相册选择'),
              onTap: () async {
                Navigator.pop(context);
                final xf = await picker.pickImage(
                    source: ImageSource.gallery, imageQuality: 80);
                if (xf != null) onChanged([...images, File(xf.path)]);
              },
            ),
            if (images.isNotEmpty)
              ListTile(
                leading: const Icon(Icons.delete_outline,
                    color: Colors.red),
                title: const Text('清除全部图片',
                    style: TextStyle(color: Colors.red)),
                onTap: () {
                  Navigator.pop(context);
                  onChanged([]);
                },
              ),
          ],
        ),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    return GestureDetector(
      onTap: () => _pick(context),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Stack(
            clipBehavior: Clip.none,
            children: [
              const Icon(Icons.camera_alt_outlined,
                  color: Color(0xFF2196F3), size: 28),
              if (images.isNotEmpty)
                Positioned(
                  right: -4,
                  top: -4,
                  child: Container(
                    width: 15,
                    height: 15,
                    decoration: const BoxDecoration(
                        color: Colors.red, shape: BoxShape.circle),
                    child: Center(
                      child: Text('${images.length}',
                          style: const TextStyle(
                              color: Colors.white, fontSize: 9)),
                    ),
                  ),
                ),
            ],
          ),
          const SizedBox(height: 2),
          Text(label,
              style: const TextStyle(
                  color: Color(0xFF2196F3), fontSize: 11)),
        ],
      ),
    );
  }
}
