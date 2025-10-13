FROM pytorch/pytorch:2.8.0-cuda12.8-cudnn9-runtime AS base

WORKDIR /app

COPY requirements.txt .
RUN pip install -r requirements.txt

COPY . .
RUN mkdir -p videos
CMD ["python", "main.py"]

