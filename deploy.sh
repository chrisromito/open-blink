#!/usr/bin/env bash
cd detection_service
docker build -f torch.Dockerfile -t torch-base:1.0 .
cd ..
docker compose build runtime-base go-base
docker compose up --build -d