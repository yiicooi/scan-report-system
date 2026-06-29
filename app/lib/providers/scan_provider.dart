import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../services/api_service.dart';

class ScanState {
  final bool loading;
  final Map<String, dynamic>? order;
  final String? error;

  const ScanState({this.loading = false, this.order, this.error});

  ScanState copyWith({bool? loading, Map<String, dynamic>? order, String? error}) =>
      ScanState(
        loading: loading ?? this.loading,
        order: order ?? this.order,
        error: error ?? this.error,
      );
}

class ScanNotifier extends Notifier<ScanState> {
  @override
  ScanState build() => const ScanState();

  Future<void> scan(String qrCode) async {
    state = state.copyWith(loading: true, error: null);
    try {
      final order = await ApiService.scanOrder(qrCode);
      state = state.copyWith(loading: false, order: order);
    } on DioException catch (e) {
      final msg = e.message?.isNotEmpty == true
          ? e.message!
          : (e.response?.statusCode == 404 ? '工单不存在' : '扫码失败，请重试');
      state = state.copyWith(loading: false, error: msg, order: null);
    } catch (e) {
      state = state.copyWith(loading: false, error: e.toString(), order: null);
    }
  }

  void clear() {
    state = const ScanState();
  }
}

final scanProvider = NotifierProvider<ScanNotifier, ScanState>(ScanNotifier.new);
