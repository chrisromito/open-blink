from typing import TypedDict
from PIL import Image


class BoundingBox(TypedDict):
    """Type definition for bounding box coordinates."""
    x1: int
    y1: int
    x2: int
    y2: int


class Detection(TypedDict):
    """Type definition for a single detection."""
    label: str
    confidence: float
    bbox: BoundingBox


class DetectionMeta(TypedDict):
    """
    JSON representation of detection results
    """
    device_id: str
    timestamp: int
    image_path: str
    meta_path: str
    detections: list[Detection]
    labels: list[str]


class DetectionResult(TypedDict):
    in_path: str
    image: Image.Image | None
    detections: list[Detection]
    timestamp: int


BatchResult = list[DetectionResult]


class DetectionMessage(DetectionResult):
    device_id: str


class DetailedDetection(TypedDict):
    """Type definition for detailed detection with bounding box."""
    label: str
    confidence: float
    bbox: BoundingBox


class DetailedFrameAnnotation(TypedDict):
    """Type definition for detailed frame annotation with bounding boxes."""
    start_epoch_seconds: int
    end_epoch_seconds: int
    detections: list[DetailedDetection]


class FrameAnnotation(TypedDict):
    """Type definition for frame annotation in the specified JSON format."""
    start_epoch_seconds: int
    end_epoch_seconds: int
    detections: list[Detection]
