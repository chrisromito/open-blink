#!/usr/bin/env bash
cd $HOME/thumb/open-blink
echo "Setting permissions on run.sh..."
chmod +x ./run.sh
echo "Setting up systemd service"
cd ./devicecapture
sudo cp ./systemd/open-blink-server.service /etc/systemd/system/open-blink-server.service
echo "Reloading systemctl..."
sudo systemctl daemon-reload
# Enable services on boot
echo "Enabling services"
sudo systemctl enable open-blink-server.service
echo "Starting open-blink-server.service..."
sudo systemctl start open-blink-server.service
echo "    Success"
echo "Installation complete"