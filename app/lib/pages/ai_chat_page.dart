import 'dart:async';

import 'package:flutter/material.dart';
import '../services/api_service.dart';

class AiChatPage extends StatefulWidget {
  const AiChatPage({super.key});

  @override
  State<AiChatPage> createState() => _AiChatPageState();
}

class _AiChatPageState extends State<AiChatPage> {
  final _inputCtrl = TextEditingController();
  final _scrollCtrl = ScrollController();
  final List<_ChatMessage> _messages = [];
  StreamSubscription<String>? _subscription;
  bool _loading = true;
  bool _sending = false;

  @override
  void initState() {
    super.initState();
    _loadHistory();
  }

  @override
  void dispose() {
    _subscription?.cancel();
    _inputCtrl.dispose();
    _scrollCtrl.dispose();
    super.dispose();
  }

  Future<void> _loadHistory() async {
    try {
      final rows = await ApiService.getAIChatMessages();
      if (!mounted) return;
      setState(() {
        _messages
          ..clear()
          ..addAll(rows.map(_ChatMessage.fromJson));
        if (_messages.isEmpty) {
          _messages.add(const _ChatMessage(
            role: _ChatRole.assistant,
            content: '可以问我工单进度，例如：查一下外部单号 A123 做到哪了。',
          ));
        }
        _loading = false;
      });
      _scrollToBottom();
    } catch (_) {
      if (!mounted) return;
      setState(() {
        _messages
          ..clear()
          ..add(const _ChatMessage(
            role: _ChatRole.assistant,
            content: '可以问我工单进度，例如：查一下外部单号 A123 做到哪了。',
          ));
        _loading = false;
      });
    }
  }

  Future<void> _send() async {
    final text = _inputCtrl.text.trim();
    if (text.isEmpty || _sending || _loading) return;
    _inputCtrl.clear();
    setState(() {
      _sending = true;
      _messages.add(_ChatMessage(role: _ChatRole.user, content: text));
      _messages.add(const _ChatMessage(role: _ChatRole.assistant, content: ''));
    });
    _scrollToBottom();

    final assistantIndex = _messages.length - 1;
    _subscription = ApiService.streamAIChat(text).listen(
      (delta) {
        if (!mounted) return;
        setState(() {
          final old = _messages[assistantIndex];
          _messages[assistantIndex] = old.copyWith(content: old.content + delta);
        });
        _scrollToBottom();
      },
      onError: (error) {
        if (!mounted) return;
        setState(() {
          _messages[assistantIndex] = _messages[assistantIndex].copyWith(
            content: '查询失败：$error',
          );
          _sending = false;
        });
      },
      onDone: () {
        if (!mounted) return;
        setState(() => _sending = false);
      },
      cancelOnError: true,
    );
  }

  void _scrollToBottom() {
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!_scrollCtrl.hasClients) return;
      _scrollCtrl.animateTo(
        _scrollCtrl.position.maxScrollExtent,
        duration: const Duration(milliseconds: 220),
        curve: Curves.easeOut,
      );
    });
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('AI查进度'),
        backgroundColor: const Color(0xFF1565C0),
        foregroundColor: Colors.white,
      ),
      body: Column(
        children: [
          Expanded(
            child: _loading
                ? const Center(child: CircularProgressIndicator())
                : ListView.builder(
                    controller: _scrollCtrl,
                    padding: const EdgeInsets.all(16),
                    itemCount: _messages.length,
                    itemBuilder: (context, index) => _MessageBubble(message: _messages[index]),
                  ),
          ),
          SafeArea(
            top: false,
            child: Padding(
              padding: const EdgeInsets.fromLTRB(12, 8, 12, 12),
              child: Row(
                children: [
                  Expanded(
                    child: TextField(
                      controller: _inputCtrl,
                      minLines: 1,
                      maxLines: 4,
                      textInputAction: TextInputAction.send,
                      onSubmitted: (_) => _send(),
                      decoration: const InputDecoration(
                        hintText: '输入内部单号、外部单号或零件名称',
                        border: OutlineInputBorder(),
                        isDense: true,
                      ),
                    ),
                  ),
                  const SizedBox(width: 8),
                  IconButton.filled(
                    onPressed: (_sending || _loading) ? null : _send,
                    style: IconButton.styleFrom(backgroundColor: const Color(0xFF1565C0)),
                    icon: _sending
                        ? const SizedBox(
                            width: 18,
                            height: 18,
                            child: CircularProgressIndicator(strokeWidth: 2, color: Colors.white),
                          )
                        : const Icon(Icons.send),
                  ),
                ],
              ),
            ),
          ),
        ],
      ),
    );
  }
}

class _MessageBubble extends StatelessWidget {
  final _ChatMessage message;

  const _MessageBubble({required this.message});

  @override
  Widget build(BuildContext context) {
    final isUser = message.role == _ChatRole.user;
    return Align(
      alignment: isUser ? Alignment.centerRight : Alignment.centerLeft,
      child: Container(
        constraints: BoxConstraints(maxWidth: MediaQuery.of(context).size.width * 0.78),
        margin: const EdgeInsets.only(bottom: 10),
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
        decoration: BoxDecoration(
          color: isUser ? const Color(0xFF1565C0) : Colors.grey.shade100,
          borderRadius: BorderRadius.circular(10),
        ),
        child: Text(
          message.content.isEmpty ? '正在查询...' : message.content,
          style: TextStyle(color: isUser ? Colors.white : Colors.black87, height: 1.35),
        ),
      ),
    );
  }
}

enum _ChatRole { user, assistant }

class _ChatMessage {
  final _ChatRole role;
  final String content;

  const _ChatMessage({required this.role, required this.content});

  factory _ChatMessage.fromJson(dynamic json) {
    final map = json as Map;
    final role = map['role']?.toString() == 'user' ? _ChatRole.user : _ChatRole.assistant;
    return _ChatMessage(role: role, content: map['content']?.toString() ?? '');
  }

  _ChatMessage copyWith({String? content}) {
    return _ChatMessage(role: role, content: content ?? this.content);
  }
}
