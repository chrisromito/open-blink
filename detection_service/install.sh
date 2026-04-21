#!/usr/bin/env bash
sudo apt install -y python3-picamera2 --no-install-recommends
sudo apt install -y iptables
cd $HOME/thumb/open-blink/detection_service
echo "Setting up venv..."
python3 -m venv venv
source ./venv/bin/activate
pip install -r requirements.in
echo "Setting permissions on run.sh..."
chmod +x ./run.sh
echo "Setting up systemd service"
sudo cp ./systemd/open-blink-detection.service /etc/systemd/system/open-blink-detection.service
echo "Reloading systemctl..."
sudo systemctl daemon-reload
# Enable services on boot
echo "Enabling services"
sudo systemctl enable open-blink-detection.service
echo "Starting open-blink-detection.service..."
sudo systemctl start open-blink-detection.service
echo "    Success"
echo "Installation complete"