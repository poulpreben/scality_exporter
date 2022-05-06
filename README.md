# README

## 1. Create service account

```bash
sudo groupadd -r scality_exporter
sudo useradd -r -m -g scality_exporter scality_exporter
```

## 2. Create `systemd` unit

```bash
sudo bash -c 'cat <<EOF > /etc/systemd/system/scality_exporter.service
[Unit]
Description=Scality Exporter
After=network.target

[Service]
User=scality_exporter
Group=scality_exporter
Type=simple
ExecStart=/usr/local/bin/scality_exporter

[Install]
WantedBy=multi-user.target
EOF'
```

## 3. Install the exporter

```bash
sudo mv scality_exporter /usr/local/bin/
sudo chown scality_exporter /usr/local/bin/scality_exporter
```

## 4. Start the service

```bash
sudo systemctl daemon-reload
sudo systemctl start scality_exporter
```