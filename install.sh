#!/bin/bash
set -e

# Налаштування
PORT=9999
GITHUB_USER="yaroslavytm-oss"
RELEASE_URL="https://github.com/${GITHUB_USER}/go-web-scanner/releases/latest/download/web-scanner"

echo -e "\033[0;32m==================================================\033[0m"
echo -e "\033[1;32m  Go Anti-Malware Engine Setup \033[0m"
echo -e "\033[0;32m==================================================\033[0m"

# Функція оновлення, яка замінює бінарник "на льоту"
	update() {
    	  echo "[*] Зупинка служби..."
    	  systemctl stop web-scanner
    	  echo "[*] Завантаження нової версії..."
    	  curl -L "https://github.com/yaroslavytm-oss/go-web-scanner/releases/latest/download/web-scanner" -o /usr/local/bin/web-scanner
   	  chmod +x /usr/local/bin/web-scanner
    	  systemctl start web-scanner
    	  echo "[+] Оновлено!"
	}

if [[ "$1" == "--update" ]]; then update; fi

echo "Виберіть режим встановлення:"
echo "1) Як системна служба (Daemon)"
echo "2) Тільки виконуваний файл (Portable)"
echo -n "Ваш вибір [1-2]: "
read mode < /dev/tty

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
    
    echo -e "\033[1;36mФайл завантажено. Як користуватися:\033[0m"
    echo "./web-scanner --mode=ui --addr=0.0.0.0:${PORT}"
    echo "Більше деталей: https://github.com/${GITHUB_USER}/go-web-scanner"
else
    echo "Невірний вибір."
    exit 1
fi
