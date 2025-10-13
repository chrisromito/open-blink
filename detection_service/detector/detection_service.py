import json
import os
from dataclasses import dataclass
from datetime import datetime
from pathlib import Path

import paho.mqtt.client as mqtt

from detector.detection_types import DetectionResult, DetectionMeta, BatchResult, DetectionMessage
from detector.image_detection import process_image, process_images
from shared.date_utils import get_epoch
from shared.json_utils import write_json
from shared.perf_utils import block_timer
from shared.log_utils import logger


@dataclass
class ImageMessage:
    device_id: str
    payload: bytes
    timestamp: int


@dataclass
class BatchMessage:
    device_id: str
    payload: bytes


@dataclass
class InMessage:
    topic: str
    payload: bytes


@dataclass
class OutMessage:
    topic: str
    data: str | bytes
    qos: int = 1


class DetectionRepo:

    def __init__(self, client: mqtt.Client):
        self.client = client

    def result_to_out_message(self, device_id: str, detection_result: DetectionResult) -> OutMessage:
        logger.debug(f'Repo: result_to_out_message for {device_id}: {detection_result}')
        self.write_image(detection_result)
        meta: DetectionMeta = self.result_to_meta(device_id, detection_result)
        self.write_meta(meta)
        return OutMessage(
            topic=f'detection/{device_id}',
            data=json.dumps(meta)
        )

    def result_to_meta(self, device_id: str, detection_result: DetectionResult) -> DetectionMeta:
        detections = detection_result.get('detections', [])
        return DetectionMeta(
            device_id=device_id,
            timestamp=int(datetime.now().timestamp() * 1000),
            image_path=get_image_path(detection_result),
            meta_path=get_meta_path(detection_result),
            detections=detections,
            labels=[x.get('label') for x in detections]
        )

    def write_meta(self, detection_meta: DetectionMeta):
        out_path = detection_meta.get('meta_path')
        logger.debug(f'write_meta ({out_path}): {detection_meta}')
        write_json(out_path, detection_meta)

    def write_image(self, detection_result: DetectionResult):
        out_path = get_image_path(detection_result)
        logger.debug(f'write_image: {out_path}')
        detection_result.get('image').save(out_path)


class DetectionService:

    def __init__(self, device_id: str, model, repo: DetectionRepo):
        self.device_id = device_id
        self.model = model
        self.repo: DetectionRepo = repo

    @classmethod
    def receive_message(cls, msg: InMessage, model, repo) -> list[DetectionResult]:
        if msg.topic.startswith('end-stream/'):
            # Batch message
            device_id: str = msg.topic.replace('end-stream/', '')
            instance = cls(device_id, model, repo)
            # return instance.receive_batch_message(
            #     BatchMessage(
            #         device_id=device_id,
            #         payload=msg.payload
            #     )
            # )
            logger.debug(f'receive_message: batch message')
            return []
        elif msg.topic.startswith('image/'):
            device_id: str = msg.topic.replace('image/', '')
            instance = cls(device_id, model, repo)
            data = json.loads(msg.payload.decode())
            timestamp = data.get('timestamp', int(datetime.now().timestamp() * 1000))
            result = instance.receive_image_message(
                ImageMessage(
                    device_id=device_id,
                    payload=msg.payload,
                    timestamp=timestamp
                )
            )
            return [result]
        return []

    @classmethod
    def receive_in_messages(cls, msgs: list[InMessage], model, repo) -> list[OutMessage]:
        messages = [m for m in msgs if m.topic.startswith('image/')]
        if len(messages) < 1:
            logger.debug(f'receive_in_messages: no image messages found, returning empty list')
            return []
        # Group Messages by device ID
        logger.debug(f'DetectionService -> receive_in_messages: {messages}')
        receiver_timer = block_timer(f'\n\nreceive_in_messages: grouping messages by device ID')
        device_message_map: dict[str, list[ImageMessage]] = {}
        for message in messages:
            data = json.loads(message.payload.decode())
            timestamp = data.get('timestamp', int(datetime.now().timestamp() * 1000))
            device_id = data.get('device_id')
            if device_id not in device_message_map:
                device_message_map[device_id] = []
            device_message_map[device_id].append(
                ImageMessage(
                    device_id=device_id,
                    payload=message.payload,
                    timestamp=timestamp
                )
            )
        out_messages: list[OutMessage] = []
        # Process images, grouped by device ID for output cohesion
        for device_id, image_messages in device_message_map.items():
            instance = cls(device_id, model, repo)
            detection_results: list[DetectionResult] = instance.receive_image_messages(image_messages)
            group_messages: list[OutMessage] = [
                repo.result_to_out_message(device_id, dr)
                for dr in detection_results
            ]
            logger.debug(f'DetectionService -> group_messages: {group_messages}')
            out_messages = out_messages + group_messages
        receiver_timer()
        return out_messages

    def receive_image_message(self, msg: ImageMessage) -> DetectionMessage:
        """
        Perform inference on an Image & return the DetectionResult
        """
        data = json.loads(msg.payload.decode())
        file_name: str = data.get('file_name')
        output_image, detections, _ = process_image(self.model, file_name)
        return DetectionMessage(
            in_path=file_name,
            device_id=self.device_id,
            image=output_image,
            detections=detections,
            timestamp=get_epoch()
        )

    def receive_image_messages(self, msgs: list[ImageMessage]) -> list[DetectionMessage]:
        """
        Perform inference o list of Images & return the DetectionResults
        """
        logger.debug(f'DetectionService -> receive_image_messages: {msgs}')
        data_list = [
            json.loads(msg.payload.decode())
            for msg in msgs
        ]
        batch_results: BatchResult = process_images(self.model, [x.get('file_name') for x in data_list])
        results: list[DetectionMessage] = []
        for result in batch_results:
            detections = result.get('detections', [])
            if result.get('image', None) is None or len(detections) < 1:
                continue
            results.append(
                DetectionMessage(
                    in_path=result.get('in_path'),
                    device_id=self.device_id,
                    image=result.get('image'),
                    detections=result.get('detections'),
                    timestamp=result.get('timestamp')
                )
            )
        return results

    def receive_batch_message(self, msg: BatchMessage) -> list[DetectionMessage]:
        """
        Introspect a BatchMessage & perform inference on all image files in the respective directory
        """
        dir_path = msg.payload.decode()
        image_paths: list[str] = get_images_in_directory(dir_path)
        results: list[DetectionMessage] = []
        for image_path in image_paths:
            batch_result: BatchResult = process_images(self.model, [image_path])
            for result in batch_result:
                if result.get('image', None) is None:
                    continue
                results.append(
                    DetectionMessage(
                        in_path=result.get('in_path'),
                        device_id=self.device_id,
                        image=result.get('image'),
                        detections=result.get('detections'),
                        timestamp=get_epoch()
                    )
                )
        return results


def get_meta_path(detection_result: DetectionResult):
    original_path = strip_file_extension(detection_result.get('in_path'))
    return original_path + '_meta.json'


def get_image_path(detection_result: DetectionResult):
    original_path = strip_file_extension(detection_result.get('in_path'))
    return original_path + '_detection.jpeg'


def get_files_in_directory(directory_path: str) -> list[str]:
    files = []
    for entry in os.listdir(directory_path):
        full_path = os.path.join(directory_path, entry)
        if os.path.isfile(full_path) and (full_path.endswith('.jpg') or full_path.endswith('.jpeg')):
            files.append(full_path)
    return files


def get_images_in_directory(directory_path: str) -> list[str]:
    file_paths = get_files_in_directory(directory_path)
    return [fp for fp in file_paths if fp.endswith('jpg') or fp.endswith('jpeg')]


def strip_file_extension(file_path: str) -> str:
    ext = Path(file_path).suffix
    return file_path.replace(ext, '')
