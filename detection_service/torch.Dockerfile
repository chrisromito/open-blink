# Build as: torch-base
# docker build -f torch.Dockerfile -t torch-base:1.0 .
FROM python:3.12-bookworm AS base
WORKDIR /app


# install system deps first
RUN apt-get update && apt-get install -y \
    ffmpeg \
    libsm6 \
    libxext6 \
    && rm -rf /var/lib/apt/lists/*

FROM base AS builder

WORKDIR /app
# Install any needed packages specified in requirements.txt
COPY requirements.txt requirements.txt
#RUN pip install torch torchvision --index-url https://download.pytorch.org/whl/cpu
RUN pip install --no-cache-dir -r requirements.txt

RUN echo "Image built successfully"