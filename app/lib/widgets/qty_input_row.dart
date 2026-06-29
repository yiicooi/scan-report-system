import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

class QtyInputRow extends StatelessWidget {
  final String label;
  final TextEditingController controller;

  const QtyInputRow({super.key, required this.label, required this.controller});

  @override
  Widget build(BuildContext context) {
    return Row(
      children: [
        SizedBox(width: 80, child: Text(label)),
        Expanded(
          child: TextField(
            controller: controller,
            keyboardType: TextInputType.number,
            inputFormatters: [FilteringTextInputFormatter.digitsOnly],
            decoration: InputDecoration(
              hintText: '0',
              isDense: true,
              contentPadding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
              border: OutlineInputBorder(borderRadius: BorderRadius.circular(8)),
            ),
            textAlign: TextAlign.center,
          ),
        ),
      ],
    );
  }
}
