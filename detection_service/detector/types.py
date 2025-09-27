from typing import TypedDict, List


class Detection(TypedDict):
    """Type definition for a single detection."""
    label: str
    confidence: float


class FrameAnnotation(TypedDict):
    """Type definition for frame annotation in the specified JSON format."""
    start_epoch_seconds: int
    end_epoch_seconds: int
    detections: List[Detection]


class BoundingBox(TypedDict):
    """Type definition for bounding box coordinates."""
    x1: int
    y1: int
    x2: int
    y2: int


class DetailedDetection(TypedDict):
    """Type definition for detailed detection with bounding box."""
    label: str
    confidence: float
    bbox: BoundingBox


class DetailedFrameAnnotation(TypedDict):
    """Type definition for detailed frame annotation with bounding boxes."""
    start_epoch_seconds: int
    end_epoch_seconds: int
    detections: List[DetailedDetection]
