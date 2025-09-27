import logging
from typing import List, Set, Tuple, Optional, Dict, Any, Generator
import cv2
import numpy as np
import torch
import torchvision.transforms as transforms
from torchvision.models import detection
from PIL import Image

from detection_service.detector.types import FrameAnnotation, DetailedFrameAnnotation


class VideoObjectDetector:
    """
    A PyTorch-based model class for object detection on videos.
    """

    # COCO class names (80 classes + background)
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

    def __init__(
            self,
            model_name: str = 'fasterrcnn_resnet50_fpn',
            score_threshold: float = 0.5,
            device: Optional[str] = None,
            default_fps: float = 20.0
    ):
        """
        Initialize the PyTorch video object detector.
        
        Args:
            model_name: Name of the torchvision model to use
            score_threshold: Minimum confidence score for detections
            device: Device to run inference on ('cpu', 'cuda', or None for auto)
            default_fps: Default FPS to use if video doesn't provide it
        """
        self.model_name = model_name
        self.score_threshold = score_threshold
        self.default_fps = default_fps
        self.logger = logging.getLogger(__name__)

        # Set device
        if device is None:
            self.device = torch.device('cuda' if torch.cuda.is_available() else 'cpu')
        else:
            self.device = torch.device(device)

        self.logger.info(f"Using device: {self.device}")

        # Initialize model
        self.model = self._load_model()

        # Image preprocessing transform
        self.transform = transforms.Compose(
            [
                transforms.ToTensor()
            ]
        )

        # Visual settings for drawing
        self.margin = 10
        self.row_size = 10
        self.font_size = 1
        self.font_thickness = 1
        self.text_color = (255, 0, 0)  # Red
        self.bbox_thickness = 3

    def _load_model(self) -> torch.nn.Module:
        """Load and configure the PyTorch detection model."""
        if self.model_name == 'fasterrcnn_resnet50_fpn':
            model = detection.fasterrcnn_resnet50_fpn(pretrained=True)
        elif self.model_name == 'fasterrcnn_mobilenet_v3_large_fpn':
            model = detection.fasterrcnn_mobilenet_v3_large_fpn(pretrained=True)
        elif self.model_name == 'retinanet_resnet50_fpn':
            model = detection.retinanet_resnet50_fpn(pretrained=True)
        elif self.model_name == 'ssd300_vgg16':
            model = detection.ssd300_vgg16(pretrained=True)
        elif self.model_name == 'ssdlite320_mobilenet_v3_large':
            model = detection.ssdlite320_mobilenet_v3_large(pretrained=True)
        else:
            raise ValueError(f"Unsupported model: {self.model_name}")

        model.to(self.device)
        model.eval()
        return model

    def _preprocess_frame(self, frame: np.ndarray) -> torch.Tensor:
        """
        Preprocess frame for model input.
        
        Args:
            frame: Input frame as numpy array (BGR format from OpenCV)
            
        Returns:
            Preprocessed tensor
        """
        # Convert BGR to RGB
        frame_rgb = cv2.cvtColor(frame, cv2.COLOR_BGR2RGB)

        # Convert to PIL Image
        pil_image = Image.fromarray(frame_rgb)

        # Apply transforms
        tensor = self.transform(pil_image)

        return tensor

    def detect_frame(self, frame: np.ndarray) -> Dict[str, Any]:
        """
        Perform object detection on a single frame.
        
        Args:
            frame: Input frame as numpy array
            
        Returns:
            Detection results dictionary with boxes, labels, and scores
        """
        # Preprocess frame
        input_tensor = self._preprocess_frame(frame)
        input_batch = input_tensor.unsqueeze(0).to(self.device)

        # Perform inference
        with torch.no_grad():
            predictions = self.model(input_batch)

        # Extract results
        prediction = predictions[0]

        # Filter by score threshold
        keep_indices = prediction['scores'] > self.score_threshold

        result = {
            'boxes': prediction['boxes'][keep_indices].cpu().numpy(),
            'labels': prediction['labels'][keep_indices].cpu().numpy(),
            'scores': prediction['scores'][keep_indices].cpu().numpy()
        }

        return result

    def visualize_detections(
            self,
            frame: np.ndarray,
            detection_result: Dict[str, Any]
    ) -> Tuple[np.ndarray, Set[str]]:
        """
        Draw bounding boxes and labels on the frame.
        
        Args:
            frame: Input frame
            detection_result: Detection results from PyTorch model
            
        Returns:
            Tuple of (annotated_frame, detected_categories)
        """
        annotated_frame = frame.copy()
        categories = set()

        boxes = detection_result['boxes']
        labels = detection_result['labels']
        scores = detection_result['scores']

        for box, label, score in zip(boxes, labels, scores):
            # Extract bounding box coordinates
            x1, y1, x2, y2 = box.astype(int)

            # Draw bounding box
            cv2.rectangle(
                annotated_frame,
                (x1, y1),
                (x2, y2),
                self.text_color,
                self.bbox_thickness
            )

            # Get category name and confidence
            category_name = self.COCO_CLASSES[label] if label < len(self.COCO_CLASSES) else f"class_{label}"
            confidence = round(score, 2)
            result_text = f"{category_name} ({confidence})"
            categories.add(result_text)

            # Draw label
            text_location = (
                x1 + self.margin,
                y1 - self.margin if y1 - self.margin > 10 else y1 + self.row_size + self.margin
            )

            cv2.putText(
                annotated_frame,
                result_text,
                text_location,
                cv2.FONT_HERSHEY_PLAIN,
                self.font_size,
                self.text_color,
                self.font_thickness
            )

            self.logger.debug(f"Detected: {result_text}")

        return annotated_frame, categories

    def _create_annotation(
            self,
            detection_result: Dict[str, Any],
            timestamp_seconds: float,
            frame_duration_seconds: float = None,
            include_bboxes: bool = False
    ) -> FrameAnnotation | DetailedFrameAnnotation:
        """
        Create annotation in the specified JSON format.
        
        Args:
            detection_result: Detection results from model
            timestamp_seconds: Current frame timestamp in seconds
            frame_duration_seconds: Duration of the frame (1/fps)
            include_bboxes: Whether to include bounding box coordinates
            
        Returns:
            Annotation dictionary in specified format
        """
        detections = []

        boxes = detection_result['boxes']
        labels = detection_result['labels']
        scores = detection_result['scores']

        for i, (label, score) in enumerate(zip(labels, scores)):
            category_name = self.COCO_CLASSES[label] if label < len(self.COCO_CLASSES) else f"class_{label}"
            # Skip background and N/A classes
            if category_name not in ['__background__', 'N/A']:
                detection_data = {
                    "label": category_name,
                    "confidence": round(float(score), 3)
                }

                if include_bboxes and len(boxes) > i:
                    box = boxes[i].astype(int)
                    detection_data["bbox"] = {
                        "x1": int(box[0]),
                        "y1": int(box[1]),
                        "x2": int(box[2]),
                        "y2": int(box[3])
                    }

                detections.append(detection_data)

        # Calculate end timestamp
        if frame_duration_seconds is not None:
            end_timestamp = timestamp_seconds + frame_duration_seconds
        else:
            end_timestamp = timestamp_seconds + (1.0 / self.default_fps)

        annotation = {
            "start_epoch_seconds": int(timestamp_seconds),
            "end_epoch_seconds": int(end_timestamp),
            "detections": detections
        }

        return annotation

    def process_video(
            self,
            input_path: str,
            output_path: Optional[str] = None,
            show_preview: bool = False,
            yield_annotations: bool = True,
            include_bboxes: bool = False
    ) -> Generator[FrameAnnotation | DetailedFrameAnnotation, None, Optional[List[Set[str]]]]:
        """
        Process an entire video file for object detection with dynamic annotation yielding.
        
        Args:
            input_path: Path to input video file
            output_path: Path to save output video (optional)
            show_preview: Whether to show preview window during processing
            yield_annotations: Whether to yield annotations (if False, returns list like before)
            include_bboxes: Whether to include bounding box coordinates in annotations
            
        Yields:
            Annotation dictionary for each frame in the specified JSON format
            
        Returns:
            If yield_annotations is False, returns list of detected categories for each frame
        """
        cap = cv2.VideoCapture(input_path)
        if not cap.isOpened():
            raise ValueError(f"Cannot open video file: {input_path}")

        # Get video properties
        frame_width = int(cap.get(cv2.CAP_PROP_FRAME_WIDTH))
        frame_height = int(cap.get(cv2.CAP_PROP_FRAME_HEIGHT))
        fps = cap.get(cv2.CAP_PROP_FPS)
        total_frames = int(cap.get(cv2.CAP_PROP_FRAME_COUNT))

        if fps == 0:
            fps = self.default_fps
            self.logger.warning(f"FPS not available, using default: {fps}")

        frame_duration = 1.0 / fps

        self.logger.info(f"Processing video: {frame_width}x{frame_height}, {fps} FPS, {total_frames} frames")

        # Initialize video writer if output path is provided
        out = None
        if output_path:
            fourcc = cv2.VideoWriter_fourcc(*'mp4v')
            out = cv2.VideoWriter(output_path, fourcc, fps, (frame_width, frame_height))

        all_categories = []
        frame_count = 0

        try:
            while cap.isOpened():
                ret, frame = cap.read()
                if not ret:
                    break

                frame_count += 1

                # Calculate timestamp in seconds
                timestamp_seconds = (frame_count - 1) * frame_duration

                self.logger.debug(f"Processing frame {frame_count}/{total_frames} at {timestamp_seconds:.3f}s")

                # Perform detection
                detection_result = self.detect_frame(frame)

                if yield_annotations:
                    # Create and yield annotation
                    annotation: FrameAnnotation | DetailedFrameAnnotation = self._create_annotation(
                        detection_result,
                        timestamp_seconds,
                        frame_duration,
                        include_bboxes
                    )

                    # Only yield if there are detections
                    if annotation['detections']:
                        yield annotation

                # Create visualization
                annotated_frame, categories = self.visualize_detections(frame, detection_result)
                all_categories.append(categories)

                # Write to output video if specified
                if out is not None:
                    out.write(annotated_frame)

                # Show preview if requested
                if show_preview:
                    cv2.imshow('Object Detection', annotated_frame)
                    if cv2.waitKey(1) & 0xFF == ord('q'):
                        break

        finally:
            # Cleanup
            cap.release()
            if out is not None:
                out.release()
            cv2.destroyAllWindows()

        self.logger.info(f"Processed {frame_count} frames")

        # Return categories list if not yielding annotations
        if not yield_annotations:
            return all_categories

    def process_frame_batch(self, frames: List[np.ndarray]) -> List[Dict[str, Any]]:
        """
        Process a batch of frames for better performance.
        
        Args:
            frames: List of input frames
            
        Returns:
            List of detection results for each frame
        """
        # Preprocess all frames
        input_tensors = []
        for frame in frames:
            tensor = self._preprocess_frame(frame)
            input_tensors.append(tensor)

        # Stack into batch
        input_batch = torch.stack(input_tensors).to(self.device)

        # Perform batch inference
        with torch.no_grad():
            predictions = self.model(input_batch)

        # Process results for each frame
        results = []
        for prediction in predictions:
            # Filter by score threshold
            keep_indices = prediction['scores'] > self.score_threshold

            result = {
                'boxes': prediction['boxes'][keep_indices].cpu().numpy(),
                'labels': prediction['labels'][keep_indices].cpu().numpy(),
                'scores': prediction['scores'][keep_indices].cpu().numpy()
            }
            results.append(result)

        return results

    def get_supported_categories(self) -> List[str]:
        """Get list of categories that the model can detect."""
        return [cls for cls in self.COCO_CLASSES if cls != 'N/A' and cls != '__background__']

    def get_model_info(self) -> Dict[str, Any]:
        """Get information about the loaded model."""
        return {
            'model_name': self.model_name,
            'device': str(self.device),
            'score_threshold': self.score_threshold,
            'num_classes': len(self.COCO_CLASSES),
            'categories': self.get_supported_categories()
        }

    def set_score_threshold(self, threshold: float):
        """Update the score threshold for detections."""
        if not 0.0 <= threshold <= 1.0:
            raise ValueError("Score threshold must be between 0.0 and 1.0")
        self.score_threshold = threshold
        self.logger.info(f"Updated score threshold to {threshold}")

    def __repr__(self) -> str:
        return f"VideoObjectDetector(model={self.model_name}, device={self.device}, threshold={self.score_threshold})"
