import 'dart:io';
import 'package:flutter/material.dart';
import 'package:image_picker/image_picker.dart';

class ImagePickerGroup extends StatelessWidget {
  final String label;
  final List<File> images;
  final ValueChanged<List<File>> onChanged;

  const ImagePickerGroup({
    super.key,
    required this.label,
    required this.images,
    required this.onChanged,
  });

  Future<void> _pickImage(ImageSource source) async {
    final picker = ImagePicker();
    final xf = await picker.pickImage(source: source, imageQuality: 80);
    if (xf != null) {
      onChanged([...images, File(xf.path)]);
    }
  }

  void _remove(int index) {
    final list = [...images];
    list.removeAt(index);
    onChanged(list);
  }

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          children: [
            Text(label, style: const TextStyle(fontWeight: FontWeight.w500)),
            const Spacer(),
            IconButton(
              icon: const Icon(Icons.camera_alt, size: 20),
              onPressed: () => _pickImage(ImageSource.camera),
              tooltip: '拍照',
            ),
            IconButton(
              icon: const Icon(Icons.photo_library, size: 20),
              onPressed: () => _pickImage(ImageSource.gallery),
              tooltip: '相册',
            ),
          ],
        ),
        if (images.isNotEmpty)
          SizedBox(
            height: 80,
            child: ListView.separated(
              scrollDirection: Axis.horizontal,
              itemCount: images.length,
              separatorBuilder: (_, __) => const SizedBox(width: 8),
              itemBuilder: (ctx, i) => Stack(
                children: [
                  ClipRRect(
                    borderRadius: BorderRadius.circular(8),
                    child: Image.file(images[i], width: 80, height: 80, fit: BoxFit.cover),
                  ),
                  Positioned(
                    right: 0,
                    top: 0,
                    child: GestureDetector(
                      onTap: () => _remove(i),
                      child: Container(
                        decoration: const BoxDecoration(
                          color: Colors.red,
                          shape: BoxShape.circle,
                        ),
                        child: const Icon(Icons.close, color: Colors.white, size: 16),
                      ),
                    ),
                  ),
                ],
              ),
            ),
          ),
      ],
    );
  }
}
