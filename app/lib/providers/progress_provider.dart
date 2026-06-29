import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../services/api_service.dart';

final orderProgressProvider =
    FutureProvider.family<Map<String, dynamic>, String>((ref, orderId) async {
  return ApiService.getOrderProgress(orderId);
});
