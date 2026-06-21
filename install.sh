#!/bin/bash
set -e

PORT=9999
GITHUB_USER="yaroslavytm-oss"
RELEASE_URL="https://github.com/${GITHUB_USER}/go-web-scanner/releases/latest/download/web-scanner"

echo -e "\033[0;32m==================================================\033[0m"
echo -e "\033[1;32m  Go Anti-Malware Engine Setup \033[0m"
echo -e "\033[0;32m==================================================\033[0m"

update() {
    echo "[*] Оновлення..."
    systemctl stop web-scanner 2>/dev/null || true
    curl -L "${RELEASE_URL}" -o /usr/local/bin/web-scanner
    chmod +x /usr/local/bin/web-scanner
    systemctl start web-scanner
    echo "[+] Оновлено!"
    exit 0
}

uninstall() {
    echo "[*] Видалення служби..."
    systemctl stop web-scanner 2>/dev/null || true
    systemctl disable web-scanner 2>/dev/null || true
    rm -f /etc/systemd/system/web-scanner.service
    systemctl daemon-reload
    rm -f /usr/local/bin/web-scanner
    echo "[+] Службу та бінарний файл успішно видалено."
    exit 0
}

# Обробка аргументів (якщо запуск через CLI)
if [[ "$1" == "--update" ]]; then update; fi
if [[ "$1" == "--uninstall" ]]; then uninstall; fi

# Інтерактивне меню
echo "Виберіть дію:"
echo "1) Встановити/Перевстановити службу"
echo "2) Встановити як Portable файл"
echo "3) Видалити службу та файл"
echo -n "Ваш вибір [1-3]: "
read -n 1 mode < /dev/tty
echo ""

case $mode in
    1)
        echo "[*] Встановлення служби..."
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
        echo -e "\033[1;32m[+] Готово! Служба на порту ${PORT}\033[0m"
        ;;
    2)
        echo "[*] Завантаження в поточну папку..."
        curl -L "${RELEASE_URL}" -o web-scanner
        chmod +x web-scanner
        echo -e "\033[1;36mГотово! Запуск: ./web-scanner --mode=ui --addr=0.0.0.0:${PORT}\033[0m"
        ;;
    3)
        uninstall
        ;;
    *)
        echo -e "\033[0;31mНевірний вибір. Введіть 1, 2 або 3.\033[0m"
        exit 1
        ;;
esac
