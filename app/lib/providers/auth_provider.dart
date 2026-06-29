import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../services/api_service.dart';
import 'scan_provider.dart';
import 'report_provider.dart';
import 'progress_provider.dart';

class AuthState {
  final bool isLoggedIn;
  final Map<String, dynamic>? user;

  const AuthState({this.isLoggedIn = false, this.user});

  AuthState copyWith({bool? isLoggedIn, Map<String, dynamic>? user}) =>
      AuthState(isLoggedIn: isLoggedIn ?? this.isLoggedIn, user: user ?? this.user);
}

class AuthNotifier extends AsyncNotifier<AuthState> {
  @override
  Future<AuthState> build() async {
    final has = await ApiService.hasToken();
    if (!has) return const AuthState(isLoggedIn: false);
    // 恢复持久化的用户信息
    final user = await ApiService.loadUser();
    return AuthState(isLoggedIn: true, user: user);
  }

  Future<void> login(String name, String password) async {
    state = const AsyncLoading();
    state = await AsyncValue.guard(() async {
      final res = await ApiService.login(name, password);
      await ApiService.saveTokens(res['access_token'], res['refresh_token']);
      final user = res['user'] as Map<String, dynamic>?;
      if (user != null) await ApiService.saveUser(user);
      return AuthState(isLoggedIn: true, user: user);
    });
  }

  Future<void> logout() async {
    await ApiService.clearTokens();
    // 清空所有业务数据，避免换下一个人时看到旧数据
    ref.invalidate(scanProvider);
    ref.invalidate(reportSubmitProvider);
    ref.invalidate(reportHistoryProvider);
    ref.invalidate(orderProgressProvider);
    state = const AsyncData(AuthState(isLoggedIn: false));
  }
}

final authProvider = AsyncNotifierProvider<AuthNotifier, AuthState>(AuthNotifier.new);

final isLoggedInProvider = Provider<bool>((ref) {
  return ref.watch(authProvider).maybeWhen(
    data: (s) => s.isLoggedIn,
    orElse: () => false,
  );
});
