# Overview

## Setup
1. Build the base torch image (makes subsequent runs MUCH faster):
    ```shell
    cd detection_service
    docker build -f torch.Dockerfile -t torch-base:1.0 .
    ```
2. Run the "local"/testing workflow first:
    ```shell
    docker compose --file docker-compose.local.yaml up --build
    ```

3. Open your browser: http://localhost:4000