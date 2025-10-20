# Build as: torch-base
# docker build -f torch.Dockerfile -t torch-base:1.0 .
FROM python:3.12-bookworm

# Set the working directory to /app
WORKDIR /app

# Install any needed packages specified in requirements.txt
COPY requirements.txt requirements.txt
RUN pip install torch torchvision --index-url https://download.pytorch.org/whl/cpu
RUN pip install opencv-python paho-mqtt pillow

COPY . .

RUN echo "Image built successfully"