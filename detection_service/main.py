import json
import logging
import paho.mqtt.client as mqtt
import queue
import sys
import threading
import time
from datetime import datetime
from os import environ

from detector.detection_service import get_images_in_directory, ImageMessage, InMessage, DetectionRepo, OutMessage, DetectionService
from detector.detection_types import BatchResult
from detector.image_detection import get_model, process_image, process_images
from pubsub.client import get_client
from pubsub.tcp_ping import ping, ping_client
from shared.log_utils import logger


def get_broker(client) -> str | None:
    broker_options: list[str] = [
        environ.get('MQTT_HOST', 'mosquitto'),
        'mosquitto',
        'host.docker.internal',
        'localhost',
        '0.0.0.0'
    ]
    for broker in broker_options:
        valid = ping(broker, 1883)
        valid_client = ping_client(client, broker, 1883)
        if valid_client or valid:
            return broker
    return None


class App:

    def __init__(self, model, client: mqtt.Client, broker: str):
        self.model = model
        self.client: mqtt.Client = client
        self.broker: str = broker
        self.in_queue: queue.Queue[InMessage] = queue.Queue()
        self.out_queue: queue.Queue[OutMessage] = queue.Queue(maxsize=100)
        # self.batch_queue: queue.Queue[BatchMessage] = queue.Queue(maxsize=10)
        self._connected: bool = False
        self.repo: DetectionRepo = DetectionRepo(self.client)

    def run(self):
        self.setup_client()
        logger.debug(f'app: setup_client')
        threads = [
            threading.Thread(
                target=self.mqtt_thread,
            ),
            threading.Thread(
                target=self.message_process_thread,
            ),
            threading.Thread(target=self.publisher_thread)
        ]
        logger.debug(f'app: starting threads')
        for t in threads:
            t.start()
        for t in threads:
            t.join()

    def setup_client(self):
        self.client.on_connect = self.on_connect
        self.client.on_message = self.on_message
        self.client.connect(self.broker or '0.0.0.0', 1883)

    def mqtt_thread(self):
        self.client.loop_forever()

    def message_process_thread(self):
        while True:
            if self.in_queue.empty():
                time.sleep(0.1)
                continue
            logger.debug(f'message_process_thread: in_queue not empty')
            msgs: list[InMessage] = [
                # self.in_queue.get(timeout=0.01)
            ]
            # # Grab up to 5 images
            for i in range(5):
                if not self.in_queue.empty():
                    msgs.append(self.in_queue.get(timeout=0.01))
            logger.debug(f'message_process_thread: processing messages')
            logger.info(msgs)
            out_messages: list[OutMessage] = DetectionService.receive_in_messages(
                msgs,
                self.model,
                self.repo
            )
            logger.debug(f'message_process_thread: out_messages -> {out_messages}')
            for out_message in out_messages:
                if self.out_queue.full():
                    self.out_queue.get_nowait()
                self.out_queue.put(out_message)
                logger.debug(f'message_process_thread: put message on out_queue')

    def publisher_thread(self):
        """
        publisher_thread reads from the out_queue and publishes to MQTT
        """
        while True:
            if not self._connected:
                time.sleep(1)
                continue
            if self.out_queue.empty():
                time.sleep(0.01)
                continue
            out_message: OutMessage = self.out_queue.get_nowait()
            self.client.publish(
                out_message.topic,
                payload=out_message.data,
                qos=out_message.qos
            )
        return None

    def on_connect(self, client, userdata, flags, reason_code, properties):
        print("Connected with result code " + str(reason_code))
        self.client.subscribe('image/#', qos=1)
        self.client.subscribe('end-stream/#', qos=1)
        self._connected = True

    # Message Handler
    def on_message(self, client, userdata, msg):
        """
        on_message manages the handoff to "in_queue"
        """
        topic = msg.topic
        logger.debug(f"Received message on topic {topic}")
        if self.in_queue.full():
            logger.debug(f'on_message: in_queue full, dropping message')
            self.in_queue.get_nowait()
        self.in_queue.put(InMessage(topic, msg.payload))

    def handle_batch_message(self, device_id: str, payload: bytes):
        try:
            dir_path = payload.decode()
            image_paths: list[str] = get_images_in_directory(dir_path)
            if len(image_paths) < 1:
                logger.error(f'handle_batch_message: no images found in {dir_path}')
                return None
            logger.debug(f'handle_batch_message: image_paths -> {image_paths}')
            batch_result: BatchResult = process_images(self.model, image_paths)
            for result in batch_result:
                if result.get('image', None) is None:
                    logger.error(f'Skipping empty result: {result}')
                    continue
                in_path = result.get('in_path')
                sans_jpg: str = in_path.replace('.jpg', '').replace('.jpeg', '')
                out_path = f"{sans_jpg}_detection.jpeg"
                with open(out_path, 'wb') as out_f:
                    result.get('image').save(out_f, format='JPEG')
                detections = result.get('detections', [])
                meta_data = {
                    'device_id': device_id,
                    'timestamp': datetime.now().timestamp() * 1000,
                    'image_path': out_path,
                    'meta_path': f'{sans_jpg}.json',
                    'detections': detections,
                    'labels': [x.get('label') for x in detections]
                }
                logger.debug(f'meta_data -> {meta_data}')
                print(f'meta_data -> {meta_data}')
                if not self.out_queue.full():
                    self.out_queue.put(meta_data)
                else:
                    self.out_queue.get_nowait()

        except Exception as err:
            logger.error(f"Error decoding batch message: {err}")
            raise err

    def process_message(self, message: ImageMessage):
        """
        Helper function for the detection thread; hands off to the model for object detection
            and pushes detection results to the output queue.
        message: ImageMessage
        """
        try:
            data = json.loads(message.payload.decode())
            logger.debug(f'process_message: message data data -> {data}')
            epoch: int = data.get('epoch', message.timestamp)
            file_name: str | None = data.get('file_name', None)
            if file_name is None:
                return None
            elif not file_name.startswith('/'):
                file_name = f'/{file_name}'
            output_image, detections, categories = process_image(self.model, file_name)
            if len(categories) == 0:
                return None
            # Write the image to disk
            sans_jpg: str = file_name.replace('.jpg', '').replace('.jpeg', '')
            out_path = f"{sans_jpg}_detection.jpeg"
            with open(out_path, 'wb') as out_f:
                output_image.save(out_f, format='JPEG')
            meta_data = {
                'device_id': message.device_id or data.get('device_id', ''),
                'timestamp': message.timestamp,
                'image_path': out_path,
                'meta_path': f'{sans_jpg}.json',
                'detections': detections,
                'labels': [x.get('label') for x in detections]
            }
            logger.debug(f'meta_data -> {meta_data}')
            if not self.out_queue.full():
                self.out_queue.put(meta_data)
            else:
                self.out_queue.get_nowait()

        except Exception as err:
            logger.error(f"Error processing frame for device {message.device_id}: {err}")
            return None


def main():
    pubsub_client = get_client()
    broker = get_broker(pubsub_client)
    if broker is None:
        broker = get_broker(pubsub_client)
    if broker is None:
        raise Exception("Could not connect to MQTT broker")
    model = get_model()
    app = App(
        model=model,
        client=pubsub_client,
        broker=broker
    )
    app.run()


if __name__ == "__main__":
    main()
