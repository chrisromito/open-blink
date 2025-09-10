import logging
from pathlib import Path

import cv2
import numpy as np
from mediapipe import Image, ImageFormat
from mediapipe.tasks.python import BaseOptions, vision


logger = logging.getLogger(__name__)
logger.setLevel(logging.DEBUG)
MODEL_PATH = Path('./models/efficientdet_lite_int.tflite')


def get_detector():
    base_options = BaseOptions(model_asset_path=MODEL_PATH)
    options = vision.ObjectDetectorOptions(
        base_options=base_options,
        max_results=5,
        score_threshold=0.5,
        running_mode=vision.RunningMode.VIDEO
    )
    detector = vision.ObjectDetector.create_from_options(options)
    return detector


def test_video_detection():
    pass


def detect(input_path: str, output_path: str, debug: bool = False):
    _detector = get_detector()
    cap = cv2.VideoCapture(input_path)
    all_categories: list[set[str]] = []
    with _detector as detector:
        frame_width = int(cap.get(cv2.CAP_PROP_FRAME_WIDTH))
        frame_height = int(cap.get(cv2.CAP_PROP_FRAME_HEIGHT))
        fps = cap.get(cv2.CAP_PROP_FPS)  # Note: This might return 0 if the camera doesn't report it
        if fps == 0:  # Set a default FPS if not available
            fps = 20.0
        fourcc = cv2.VideoWriter_fourcc(*'mp4v')
        out = cv2.VideoWriter(output_path, fourcc, fps, (frame_width, frame_height))

        while cap.isOpened():
            ret, frame = cap.read()
            if not ret:
                break
            # Convert the frame to MediaPipe Image format
            mp_image: Image = Image(image_format=ImageFormat.SRGB, data=frame)

            # Get the timestamp for the current frame (in milliseconds)
            # This is important for video mode
            frame_timestamp_ms: int = int(cap.get(cv2.CAP_PROP_POS_MSEC))

            # Perform object detection
            detection_result = detector.detect_for_video(mp_image, frame_timestamp_ms)
            out_frame, categories = visualize(frame, detection_result)
            out.write(out_frame)
            all_categories.append(categories)
            if debug:
                cv2.imshow('Detection result', out_frame)

            if cv2.waitKey(1) & 0xFF == ord('q'):
                break

        cap.release()
        out.release()
        cv2.destroyAllWindows()
        return all_categories


MARGIN = 10  # pixels
ROW_SIZE = 10  # pixels
FONT_SIZE = 1
FONT_THICKNESS = 1
TEXT_COLOR = (255, 0, 0)  # red


def visualize(image, detection_result) -> tuple[np.ndarray, set[str]]:
    """Draws bounding boxes on the input image and return it.
    Args:
    image: The input RGB image.
    detection_result: The list of all "Detection" entities to be visualize.
    Returns:
    Image with bounding boxes.
    cateegories
    """
    categories: set[str] = set()
    for detection in detection_result.detections:
        # Draw bounding_box
        bbox = detection.bounding_box
        start_point = bbox.origin_x, bbox.origin_y
        end_point = bbox.origin_x + bbox.width, bbox.origin_y + bbox.height
        cv2.rectangle(image, start_point, end_point, TEXT_COLOR, 3)

        # Draw label and score
        category = detection.categories[0]
        category_name = category.category_name
        logger.info(category_name)
        probability = round(category.score, 2)
        result_text = category_name + ' (' + str(probability) + ')'
        categories.add(result_text)
        text_location = (MARGIN + bbox.origin_x,
                         MARGIN + ROW_SIZE + bbox.origin_y)

        cv2.putText(
            image,
            result_text,
            text_location,
            cv2.FONT_HERSHEY_PLAIN,
            FONT_SIZE, TEXT_COLOR, FONT_THICKNESS
        )

    return image, categories


def main():
    from pprint import pprint
    from pathlib import Path
    input_path = str(Path('./test_videos/test.mp4').resolve())
    output_path = str(Path('./test_videos/test_detection.mp4').resolve())
    categories = detect(input_path, output_path, True)
    print('done')
    pprint(categories)


if __name__ == "__main__":
    main()
