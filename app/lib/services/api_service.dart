import 'dart:convert';
import 'dart:io';
import 'package:dio/dio.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:uuid/uuid.dart';

const _baseUrl = 'http://192.168.10.10:8080/api';

class _TokenStorage {
  static const _accessTokenKey = 'access_token';
  static const _refreshTokenKey = 'refresh_token';
  static const _userKey = 'current_user';
  static const FlutterSecureStorage instance = FlutterSecureStorage();

  static Future<String?> readAccessToken() {
    return instance.read(key: _accessTokenKey);
  }

  static Future<void> save(String access, String refresh) async {
    await Future.wait([
      instance.write(key: _accessTokenKey, value: access),
      instance.write(key: _refreshTokenKey, value: refresh),
    ]);
  }

  static Future<void> saveUser(Map<String, dynamic> user) async {
    await instance.write(key: _userKey, value: jsonEncode(user));
  }

  static Future<Map<String, dynamic>?> loadUser() async {
    final raw = await instance.read(key: _userKey);
    if (raw == null || raw.isEmpty) return null;
    try {
      return jsonDecode(raw) as Map<String, dynamic>;
    } catch (_) {
      return null;
    }
  }

  static Future<void> clear() async {
    await Future.wait([
      instance.delete(key: _accessTokenKey),
      instance.delete(key: _refreshTokenKey),
      instance.delete(key: _userKey),
    ]);
  }

  static Future<bool> hasAccessToken() async {
    final token = await readAccessToken();
    return token != null && token.isNotEmpty;
  }
}

class ApiService {
  static final Dio _dio = Dio(BaseOptions(
    baseUrl: _baseUrl,
    connectTimeout: const Duration(seconds: 10),
    receiveTimeout: const Duration(seconds: 15),
  ))
    ..interceptors.add(InterceptorsWrapper(
      onRequest: (options, handler) async {
        final token = await _TokenStorage.readAccessToken();
        if (token != null) {
          options.headers['Authorization'] = 'Bearer $token';
        }
        return handler.next(options);
      },
      onError: (error, handler) {
        if (error.response?.statusCode == 401) {
          // TODO: token 刷新或跳转登录
        }
        // 提取后端返回的具体错误信息
        final data = error.response?.data;
        String? msg;
        if (data is Map) {
          msg = data['error']?.toString() ?? data['message']?.toString();
        }
        if (msg != null && msg.isNotEmpty) {
          return handler.reject(
            DioException(
              requestOptions: error.requestOptions,
              response: error.response,
              type: error.type,
              error: msg,
              message: msg,
            ),
          );
        }
        return handler.next(error);
      },
    ));

  static Future<Map<String, dynamic>> login(String name, String password) async {
    final res = await _dio.post('/auth/login', data: {'name': name, 'password': password});
    return res.data as Map<String, dynamic>;
  }

  static Future<Map<String, dynamic>> scanOrder(String internalNo) async {
    final res = await _dio.get('/app/scan', queryParameters: {'internal_no': internalNo});
    return (res.data as Map<String, dynamic>)['data'] as Map<String, dynamic>;
  }

  static Future<Map<String, dynamic>> submitReport(Map<String, dynamic> payload) async {
    final res = await _dio.post('/app/report', data: payload);
    return res.data as Map<String, dynamic>;
  }

  static Future<List<dynamic>> getReportHistory(Map<String, dynamic> params) async {
    final res = await _dio.get('/app/report/history', queryParameters: params);
    return (res.data as Map)['data'] as List? ?? [];
  }

  static Future<Map<String, dynamic>> getOrderProgress(String orderId) async {
    final res = await _dio.get('/app/orders/$orderId/progress');
    return res.data as Map<String, dynamic>;
  }

  static Future<List<dynamic>> getAIChatMessages() async {
    final res = await _dio.get('/app/ai/chat/messages');
    return (res.data as Map)['data'] as List? ?? [];
  }

  static Stream<String> streamAIChat(String message) async* {
    final res = await _dio.post<ResponseBody>(
      '/app/ai/chat/stream',
      data: {'message': message},
      options: Options(responseType: ResponseType.stream),
    );
    final body = res.data;
    if (body == null) return;

    var buffer = '';
    await for (final chunk in body.stream) {
      buffer += utf8.decode(chunk, allowMalformed: true);
      while (true) {
        final idx = buffer.indexOf('\n\n');
        if (idx < 0) break;
        final eventBlock = buffer.substring(0, idx);
        buffer = buffer.substring(idx + 2);
        final parsed = _parseSseBlock(eventBlock);
        if (parsed == null) continue;
        if (parsed.event == 'done') return;
        if (parsed.event == 'error') {
          throw Exception(parsed.data);
        }
        if (parsed.event == 'delta') {
          yield parsed.data;
        }
      }
    }
  }

  static Future<Map<String, dynamic>> presignUpload(String objectKey, String ext) async {
    final res = await _dio.post('/admin/oss/presign', data: {'object_key': objectKey, 'ext': ext});
    return res.data as Map<String, dynamic>;
  }

  static Future<void> uploadToOSS(String presignedUrl, File file) async {
    final bytes = await file.readAsBytes();
    await Dio().put(
      presignedUrl,
      data: Stream.fromIterable([bytes]),
      options: Options(headers: {
        'Content-Type': 'application/octet-stream',
        'Content-Length': bytes.length,
      }),
    );
  }

  static String generateObjectKey(String orderId, String type, String ext) {
    final uuid = const Uuid().v4().replaceAll('-', '').substring(0, 8);
    return 'reports/$orderId/$type/$uuid.$ext';
  }

  static Future<void> saveTokens(String access, String refresh) async {
    await _TokenStorage.save(access, refresh);
  }

  static Future<void> saveUser(Map<String, dynamic> user) async {
    await _TokenStorage.saveUser(user);
  }

  static Future<Map<String, dynamic>?> loadUser() async {
    return _TokenStorage.loadUser();
  }

  static Future<void> clearTokens() async {
    await _TokenStorage.clear();
  }

  static Future<bool> hasToken() async {
    return _TokenStorage.hasAccessToken();
  }
}

class _SseEvent {
  final String event;
  final String data;

  const _SseEvent(this.event, this.data);
}

_SseEvent? _parseSseBlock(String block) {
  var event = 'message';
  final dataLines = <String>[];
  for (final line in const LineSplitter().convert(block)) {
    if (line.startsWith('event:')) {
      event = line.substring(6).trim();
    } else if (line.startsWith('data:')) {
      dataLines.add(line.substring(5).trim());
    }
  }
  if (dataLines.isEmpty) return null;
  final raw = dataLines.join('\n');
  try {
    final decoded = jsonDecode(raw);
    return _SseEvent(event, decoded?.toString() ?? '');
  } catch (_) {
    return _SseEvent(event, raw);
  }
}
