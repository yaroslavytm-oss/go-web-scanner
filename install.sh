#!/bin/bash
set -e

echo -e "\033[0;32m==================================================\033[0m"
echo -e "\033[1;32m  Встановлення Go Anti-Malware Engine (MVP Daemon) \033[0m"
echo -e "\033[0;32m==================================================\033[0m"
echo ""

if [ "$EUID" -ne 0 ]; then
  echo -e "\033[0;31mПомилка: Цей скрипт необхідно запускати від імені root.\033[0m"
  exit 1
fi

read -p "Бажаєте запустити автоматичне встановлення антивірусного сканера? (y/n): " confirm
if [[ ! $confirm =~ ^[Yy]$ ]]; then
    echo -e "\033[0;33mВстановлення скасовано користувачем.\033[0m"
    exit 0
fi

echo -e "\n\033[0;34m[*] Крок 1: Завантаження оптимізованого Go-бінарника з GitHub...\033[0m"

ARCH=$(uname -m)
if [ "$ARCH" != "x86_64" ]; then
    echo -e "\033[0;31mПомилка: Підтримується лише архітектура x86_64.\033[0m"
    exit 1
fi

mkdir -p /tmp/scanner_install
cd /tmp/scanner_install

# Автоматичне завантаження скомпільованого релізу з твого GitHub
GITHUB_USER="yaroslavytm-oss"
RELEASE_URL="https://github.com/${GITHUB_USER}/go-web-scanner/releases/latest/download/web-scanner"

echo "[*] Завантаження з: ${RELEASE_URL}"
curl -L "${RELEASE_URL}" -o web-scanner

mv web-scanner /usr/local/bin/web-scanner
chmod +x /usr/local/bin/web-scanner

echo -e "\033[0;34m[*] Крок 2: Налаштування фонової служби systemd...\033[0m"

cat <<EOF > /etc/systemd/system/web-scanner.service
[Unit]
Description=GoWebServer Anti-Malware Engine
After=network.target

[Service]
Type=simple
User=root
ExecStart=/usr/local/bin/web-scanner --mode=ui --addr=0.0.0.0:8888 --threshold=50
Restart=always
RestartSec=5
MemoryHigh=512M
MemoryMax=768M

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable web-scanner
systemctl restart web-scanner

echo -e "\033[0;34m[*] Крок 3: Перевірка та конфігурація фаєрволу Linux...\033[0m"

if command -v ufw &> /dev/null && ufw status | grep -q "active"; then
    echo "[+] Відкриваємо порт 8888 в UFW..."
    ufw allow 8888/tcp
elif command -v firewall-cmd &> /dev/null && systemctl is-active --quiet firewalld; then
    echo "[+] Відкриваємо порт 8888 в Firewalld..."
    firewall-cmd --permanent --add-port=8888/tcp
    firewall-cmd --reload
fi

if systemctl is-active --quiet web-scanner; then
    SERVER_IP=$(curl -s ifconfig.me || echo "IP_ТВОГО_СЕРВЕРА")
    echo ""
    echo -e "\033[1;32m==================================================\033[0m"
    echo -e "\033[1;32m   УСПІШНО ВСТАНОВЛЕНО ТА ЗАПУЩЕНО!               \033[0m"
    echo -e "\033[1;32m==================================================\033[0m"
    echo -e "Веб-інтерфейс сканера доступний за посиланням:"
    echo -e "\033[1;36mhttp://${SERVER_IP}:8888\033[0m"
    echo ""
else
    echo -e "\033[0;31mСлужба не запустилася. Перевірте: systemctl status web-scanner\033[0m"
fi

rm -rf /tmp/scanner_install
