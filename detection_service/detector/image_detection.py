import io
from typing import Any

import torch
import torchvision.transforms as T
from torchvision.models.detection import fasterrcnn_resnet50_fpn_v2, FasterRCNN
from PIL import Image, ImageDraw

from detector.detection_types import Detection, BatchResult, DetectionResult
from shared.date_utils import get_epoch
from shared.perf_utils import block_timer
from shared.log_utils import logger


COCO_CLASSES = [
    '__background__', 'person', 'bicycle', 'car', 'motorcycle', 'airplane', 'bus',
    'train', 'truck', 'boat', 'traffic light', 'fire hydrant', 'N/A', 'stop sign',
    'parking meter', 'bench', 'bird', 'cat', 'dog', 'horse', 'sheep', 'cow',
    'elephant', 'bear', 'zebra', 'giraffe', 'N/A', 'backpack', 'umbrella', 'N/A',
    'N/A', 'handbag', 'tie', 'suitcase', 'frisbee', 'skis', 'snowboard', 'sports ball',
    'kite', 'baseball bat', 'baseball glove', 'skateboard', 'surfboard', 'tennis racket',
    'bottle', 'N/A', 'wine glass', 'cup', 'fork', 'knife', 'spoon', 'bowl',
    'banana', 'apple', 'sandwich', 'orange', 'broccoli', 'carrot', 'hot dog', 'pizza',
    'donut', 'cake', 'chair', 'couch', 'potted plant', 'bed', 'N/A', 'dining table',
    'N/A', 'N/A', 'toilet', 'N/A', 'tv', 'laptop', 'mouse', 'remote', 'keyboard',
    'cell phone', 'microwave', 'oven', 'toaster', 'sink', 'refrigerator', 'N/A', 'book',
    'clock', 'vase', 'scissors', 'teddy bear', 'hair drier', 'toothbrush'
]


def get_model() -> FasterRCNN:
    model = fasterrcnn_resnet50_fpn_v2(weights='DEFAULT')
    model.eval()  # Set to evaluation mode for inference
    return model


transform = T.Compose([T.ToTensor(), ])


def process_image(model: FasterRCNN, image_path: str | bytes | io.BytesIO) -> tuple[Image, list[Detection], set[str]]:
    image: Image = Image.open(image_path).convert("RGB")
    img_tensor = transform(image)
    with torch.no_grad():
        prediction = model([img_tensor])[0]
    boxes = prediction.get('boxes', torch.empty((0, 4)))
    labels = prediction.get('labels', torch.empty((0,), dtype=torch.long))
    scores = prediction.get('scores', torch.empty((0,)))
    threshold = 0.7
    high_conf_indices = scores > threshold
    filtered_boxes = boxes[high_conf_indices]
    filtered_labels = labels[high_conf_indices]
    filtered_scores = scores[high_conf_indices]
    detection_result = {
        'boxes': filtered_boxes.numpy(),
        'labels': filtered_labels.numpy(),
        'scores': filtered_scores.numpy()
    }
    detections = create_detections(detection_result)
    out_image, categories = draw_detections(image, detection_result)
    return out_image, detections, categories


def process_images(model: FasterRCNN, image_paths: list[str]) -> BatchResult:
    """
    Process multiple images in a batch for efficient inference.
    
    Args:
        model: The FasterRCNN model
        image_paths: List of image file paths
        
    Returns:
        List of tuples, each containing (processed_image, detections, categories)
    """
    if not image_paths:
        return []
    use_cuda = torch.cuda.is_available()
    # Move model to GPU if available
    if use_cuda:
        model = model.to('cuda')

    # Load and transform all images
    images = []
    img_tensors = []

    for image_path in image_paths:
        try:
            image = Image.open(image_path).convert("RGB")
            img_tensor = transform(image)
            images.append(image)
            img_tensors.append(img_tensor)
        except Exception as e:
            logger.error(f"Error loading image {image_path}: {e}")
            continue

    if not img_tensors:
        return []
    # Move input tensors to GPU if available
    if use_cuda:
        img_tensors = [tensor.to('cuda') for tensor in img_tensors]

    # Batch inference
    timer = block_timer(f'inference for: {len(image_paths)} images completed in')
    with torch.no_grad():
        predictions = model(img_tensors)

    timer()
    results: list[DetectionResult] = []
    threshold = 0.7

    # Process each prediction
    for i, (image, prediction) in enumerate(zip(images, predictions)):
        try:
            boxes = prediction.get('boxes', torch.empty((0, 4)))
            labels = prediction.get('labels', torch.empty((0,), dtype=torch.long))
            scores = prediction.get('scores', torch.empty((0,)))

            if use_cuda:
                boxes = boxes.cpu()
                labels = labels.cpu()
                scores = scores.cpu()

            # Filter by confidence threshold
            high_conf_indices = scores > threshold
            filtered_boxes = boxes[high_conf_indices]
            filtered_labels = labels[high_conf_indices]
            filtered_scores = scores[high_conf_indices]

            detection_data = {
                'boxes': filtered_boxes.numpy(),
                'labels': filtered_labels.numpy(),
                'scores': filtered_scores.numpy()
            }

            detections: list[Detection] = create_detections(detection_data)
            out_image, categories = draw_detections(image, detection_data)
            result: DetectionResult = DetectionResult(
                in_path=image_paths[i],
                image=out_image,
                detections=detections,
                timestamp=get_epoch()
            )

            results.append(result)

        except Exception as e:
            logger.error(f"Error processing image {image_paths[i]}: {e}")
            # Add empty result for this image to maintain alignment
            results.append(
                DetectionResult(
                    in_path=image_paths[i],
                    image=None,
                    detections=[],
                    timestamp=get_epoch()
                )
            )

    return results


def create_detections(detection_result: dict[str, Any]) -> list[Detection]:
    detections: list[Detection] = []

    boxes = detection_result['boxes']
    labels = detection_result['labels']
    scores = detection_result['scores']

    for i, (label, score) in enumerate(zip(labels, scores)):
        category_name = COCO_CLASSES[label] if label < len(COCO_CLASSES) else f"class_{label}"
        # Skip background and N/A classes
        if category_name not in ['__background__', 'N/A'] and len(boxes) > i:
            box = boxes[i].astype(int)
            detection_data = Detection(
                label=category_name,
                confidence=round(float(score), 3),
                bbox={
                    "x1": int(box[0]),
                    "y1": int(box[1]),
                    "x2": int(box[2]),
                    "y2": int(box[3])
                }
            )

            detections.append(detection_data)
    return detections


def draw_detections(frame: Image, detection_result: dict[str, Any]) -> tuple[Image, set[str]]:
    output_image = frame.copy()
    draw = ImageDraw.Draw(output_image)
    categories = set()

    boxes = detection_result['boxes']
    labels = detection_result['labels']
    scores = detection_result['scores']

    for box, label_idx, score in zip(boxes, labels, scores):
        # Extract bounding box coordinates
        # x1, y1, x2, y2 = box.astype(int)
        category_name = COCO_CLASSES[label_idx] if label_idx < len(COCO_CLASSES) else f"class_{label_idx}"

        categories.add(category_name)
        draw.rectangle(box, outline="green", width=2)
        # Add label and score
        text = f"Class {category_name}: {score:.2f}"
        draw.text((box[0], box[1] - 15), text, fill="green")  # Adjust text position as needed
    return output_image, categories
