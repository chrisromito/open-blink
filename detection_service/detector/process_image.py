import json
from pathlib import Path
from typing import TypeVar

from PIL import Image
from ultralytics import YOLO

from detector.detection_types import Detection, BoundingBox, BatchResult, DetectionResult
from shared.date_utils import get_epoch


class Processor:

    def __init__(self, model: YOLO):
        self.model = model

    @classmethod
    def from_path(cls, path: str | Path):
        model = YOLO(path)
        return Processor(model)

    def predict(self, image: Image, width: int = 640, height: int = 640) -> list[Detection]:
        results = self.model(image)
        return process_results(results)

    def predict_batch(self, image_paths: list[str])-> BatchResult:
        batch: BatchResult = []
        for p in image_paths:
            img = Image.open(p)
            detections = self.predict(img)
            result = DetectionResult(
                in_path=p,
                image=img,
                detections=detections,
                timestamp=get_epoch()
            )
            batch.append(result)
        return batch


def process_results(result_list) -> list[Detection]:
    """
    Result_list comes in like:
    [
        [
            {
                "name": "refrigerator",
                "class": 72,
                "confidence": 0.40313,
                "box": {
                    "x1": 76.58481,
                    "y1": 0.0,
                    "x2": 3873.19165,
                    "y2": 2494.72705
                }
            },
            {
                "name": "refrigerator",
                "class": 72,
                "confidence": 0.32549,
                "box": {
                    "x1": 558.57129,
                    "y1": 0.0,
                    "x2": 4608.0,
                    "y2": 2592.0
                }
            }
        ]
    ]

    :param result_list:
    :return:
    """
    parsed = flatten(
        [
            json.loads(result.to_json())
            for result in result_list
        ]
    )
    detections: list[Detection] = []
    for result in parsed:
        b = result.get('box')
        x1, y1, x2, y2 = b.get('x1'), b.get('y1'), b.get('x2'), b.get('y2')
        coords = [x1, y1, x2, y2]
        if x2 <= x1 or y2 <= y1:
            print(f"Skipping {coords} detection with invalid coordinate order: {b}")
            continue
        valid_coords = [isinstance(c, float) for c in coords]
        if not all(valid_coords):
            print(f'Skipping {coords} because one of them was not a float')
            continue
        detections.append(
            Detection(
                confidence=result.get('confidence'),
                label=result.get('name'),
                bbox=BoundingBox(
                    x1=x1,
                    x2=x2,
                    y1=y1,
                    y2=y2
                )
            )
        )

    return detections


T = TypeVar('T')
TList = list[T]


def flatten(your_list: list[T | TList], accum=None) -> TList:
    temp = accum or []
    for item in your_list:
        if isinstance(item, list):
            temp = [*temp, *flatten(item)]
        else:
            temp.append(item)
    return temp
