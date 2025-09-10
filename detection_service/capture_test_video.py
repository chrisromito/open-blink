from pathlib import Path

import cv2


OUT_PATH = Path('./test_videos/test.mp4')


def capture_video():
    cap = cv2.VideoCapture(0)
    frame_width = int(cap.get(cv2.CAP_PROP_FRAME_WIDTH))
    frame_height = int(cap.get(cv2.CAP_PROP_FRAME_HEIGHT))
    fps = cap.get(cv2.CAP_PROP_FPS)  # Note: This might return 0 if the camera doesn't report it
    if fps == 0:  # Set a default FPS if not available
        fps = 20.0
    fourcc = cv2.VideoWriter_fourcc(*'mp4v')  # Or 'DIVX', 'MJPG', etc.
    out = cv2.VideoWriter(
        str(OUT_PATH.resolve()),
        fourcc,
        fps,
        (frame_width, frame_height)
    )
    while cap.isOpened():
        ret, frame = cap.read()
        if ret:
            out.write(frame)
            cv2.imshow('Recording', frame)  # Optional: Display the live feed

            if cv2.waitKey(1) & 0xFF == ord('q'):  # Press 'q' to stop recording
                break
        else:
            break
    cap.release()
    out.release()
    cv2.destroyAllWindows()


def run_detection_on_test_video():
    from main import detect
    from pathlib import Path
    input_path = str(Path('./test_videos/test.mp4').resolve())
    output_path = str(Path('./test_videos/test_detection.mp4').resolve())
    categories = detect(input_path, output_path, True)
    print('done')


if __name__ == '__main__':
    capture_video()
    run_detection_on_test_video()
