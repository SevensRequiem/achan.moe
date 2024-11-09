#!/bin/sh

# Exit immediately if a command exits with a non-zero status
set -e

# Install required packages
sudo add-apt-repository -y ppa:maxmind/ppa
sudo apt-get update
sudo apt-get install -y mysql-server geoipupdate

# Download the GeoLite2 database
sudo mkdir -p /usr/share/GeoIP
sudo geoipupdate

# Create the database and user
sudo mysql <<EOF
CREATE DATABASE achan;
CREATE USER 'achan'@'localhost' IDENTIFIED BY 'achan';
GRANT ALL PRIVILEGES ON achan.* TO 'achan'@'localhost';
FLUSH PRIVILEGES;
EOF

# Create the user
sudo useradd -m achan

# Install Go
wget https://golang.org/dl/go1.23.1.linux-amd64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.23.1.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.profile
source ~/.profile

# Build the server
go build

# Copy to /opt/achan
sudo mkdir -p /opt/achan
sudo cp achan.moe /opt/achan
sudo cp -r banners assets views static /opt/achan
sudo mkdir -p /opt/achan/thumbs /opt/achan/certificates /opt/achan/backups
sudo cp .env.example /opt/achan/.env
sudo cp achan.service /home/achan/achan.service
sudo systemctl --user enable achan.service
sudo chmod -R 744 /opt/achan
sudo chown -R achan:achan /opt/achan


# Start the server
sudo systemctl --user daemon-reload
sudo systemctl --user start achan.service