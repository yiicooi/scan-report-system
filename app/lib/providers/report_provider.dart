import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../services/api_service.dart';

class ReportSubmitState {
  final bool submitting;
  final bool success;
  final String? error;

  const ReportSubmitState({this.submitting = false, this.success = false, this.error});
}

class ReportSubmitNotifier extends Notifier<ReportSubmitState> {
  @override
  ReportSubmitState build() => const ReportSubmitState();

  Future<void> submit(Map<String, dynamic> payload) async {
    state = const ReportSubmitState(submitting: true);
    try {
      await ApiService.submitReport(payload);
      state = const ReportSubmitState(success: true);
    } on DioException catch (e) {
      final msg = e.message?.isNotEmpty == true
          ? e.message!
          : (e.error?.toString() ?? '提交失败，请重试');
      state = ReportSubmitState(error: msg);
    } catch (e) {
      state = ReportSubmitState(error: e.toString());
    }
  }

  void reset() => state = const ReportSubmitState();
}

final reportSubmitProvider = NotifierProvider<ReportSubmitNotifier, ReportSubmitState>(
  ReportSubmitNotifier.new,
);

final reportHistoryProvider = FutureProvider.family<List<dynamic>, String>(
  (ref, paramsKey) async {
    // paramsKey 格式: "key1=val1&key2=val2"（已排序）
    final params = <String, dynamic>{};
    if (paramsKey.isNotEmpty) {
      for (final pair in paramsKey.split('&')) {
        final idx = pair.indexOf('=');
        if (idx != -1) params[pair.substring(0, idx)] = pair.substring(idx + 1);
      }
    }
    return ApiService.getReportHistory(params);
  },
);
