# Initialize with different models
from detection_service.detector.model import VideoObjectDetector
from detection_service.shared.json_utils import write_json


def process_video(video_path: str, output_path: str, annotation_path: str | None = None, show_preview: bool = False):
    detector = VideoObjectDetector(model_name='fasterrcnn_resnet50_fpn', score_threshold=0.7)
    # Process video
    if annotation_path is None:
        annotation_path = video_path.replace('.mp4', '') + '_annotations.json'
    annotations = []
    for annotation in detector.process_video(video_path, output_path=output_path, yield_annotations=True, include_bboxes=True):
        annotations.append(annotation)
    write_json(annotation_path, annotations)
    # Get model information
    print(detector.get_model_info())
    # Change threshold dynamically
    detector.set_score_threshold(0.3)
