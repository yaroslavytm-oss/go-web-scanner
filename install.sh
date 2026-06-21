#!/bin/bash
set -e

PORT=9999
GITHUB_USER="yaroslavytm-oss"
RELEASE_URL="https://github.com/${GITHUB_USER}/go-web-scanner/releases/latest/download/web-scanner"

echo -e "\033[0;32m==================================================\033[0m"
echo -e "\033[1;32m  Go Anti-Malware Engine Setup \033[0m"
echo -e "\033[0;32m==================================================\033[0m"

update() {
    systemctl stop web-scanner 2>/dev/null || true
    curl -L "${RELEASE_URL}" -o /usr/local/bin/web-scanner
    chmod +x /usr/local/bin/web-scanner
    systemctl start web-scanner
    echo "[+] Оновлено!"
    exit 0
}

if [[ "$1" == "--update" ]]; then update; fi

# Використовуємо /dev/tty для читання вибору, щоб це працювало і через curl | bash
echo "Виберіть режим встановлення:"
echo "1) Як системна служба (Daemon)"
echo "2) Тільки виконуваний файл (Portable)"
echo -n "Ваш вибір [1-2]: "
read -n 1 mode < /dev/tty
echo ""

if [ "$mode" == "1" ]; then
    echo "[*] Завантаження та встановлення служби..."
    curl -L "${RELEASE_URL}" -o /usr/local/bin/web-scanner
    chmod +x /usr/local/bin/web-scanner

    cat <<EOF > /etc/systemd/system/web-scanner.service
[Unit]
Description=GoWebServer Anti-Malware Engine
After=network.target

[Service]
ExecStart=/usr/local/bin/web-scanner --mode=ui --addr=0.0.0.0:${PORT} --threshold=50
Restart=always
User=root

[Install]
WantedBy=multi-user.target
EOF
    systemctl daemon-reload
    systemctl enable web-scanner
    systemctl restart web-scanner
    echo -e "\033[1;32m[+] Службу запущено на порті ${PORT}\033[0m"

elif [ "$mode" == "2" ]; then
    echo "[*] Завантаження файлу в поточну директорію..."
    curl -L "${RELEASE_URL}" -o web-scanner
    chmod +x web-scanner
    echo -e "\033[1;36mФайл завантажено. Запуск: ./web-scanner --mode=ui --addr=0.0.0.0:${PORT}\033[0m"
else
    echo -e "\n\033[0;31mНевірний вибір. Введіть 1 або 2.\033[0m"
    exit 1
fi
