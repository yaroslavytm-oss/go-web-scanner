package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"runtime"
	"scanner/pkg/engine"
	"scanner/pkg/server"
	"time"
)

//go:embed ui/index.html
var embedUI embed.FS

func main() {
	// Означення CLI прапорців
	mode := flag.String("mode", "ui", "Режим роботи утиліти: 'scan' (консольний CLI) або 'ui' (вбудований веб-інтерфейс)")
	path := flag.String("path", "", "Абсолютний шлях до директорії для сканування (лише для режиму 'scan')")
	addr := flag.String("addr", "127.0.0.1:8888", "Мережева адреса та порт для Web UI")
	workers := flag.Int("workers", runtime.NumCPU(), "Кількість одночасних потоків (воркерів) сканування")
	threshold := flag.Int("threshold", 50, "Поріг Danger Score (балів небезпеки) для маркування файлу інфікованим")

	flag.Parse()

	// Ініціалізація основних системних компонентів сканування
	detector := engine.NewDetector(*threshold)
	scanner := engine.NewScanner(detector, *workers)

	switch *mode {
	case "scan":
		if *path == "" {
			fmt.Println("Помилка: Параметр --path обов'язковий у режимі CLI сканування.")
			os.Exit(1)
		}
		
		fmt.Printf("[*] Запуск консольного сканування. Директорія: %s\n", *path)
		fmt.Printf("[*] Конфігурація: Потоків: %d, Поріг детекту: %d балів\n", *workers, *threshold)
		
		startTime := time.Now()
		// Сканування в CLI без обмежень по контексту
		scanner.StartScan(context.Background(), *path)
		elapsed := time.Since(startTime)

		_, scanned, found, list := scanner.GetStats()
		fmt.Println("\n================ ЗВІТ СКАНУВАННЯ ================")
		fmt.Printf("Час виконання:      %s\n", elapsed)
		fmt.Printf("Перевірено файлів:   %d\n", scanned)
		fmt.Printf("Знайдено шкідливих:  %d\n", found)
		fmt.Println("=================================================")
		
		if found > 0 {
			fmt.Println("\nСписок інфікованих об'єктів:")
			for _, file := range list {
				fmt.Printf("[-] ЗАГРОЗА: [%s] (Бали: %d) -> %s\n", file.Matches[0], file.DangerScore, file.FilePath)
				fmt.Printf("    MD5:    %s\n", file.MD5)
				fmt.Printf("    SHA256: %s\n", file.SHA256)
			}
			os.Exit(1) // Повертаємо non-zero код, якщо знайдено віруси (корисно для CI/CD або Cron)
		}
		fmt.Println("[+] Сервер чистий. Шкідливого ПЗ не виявлено.")

	case "ui":
		fmt.Printf("[*] Запуск антивірусного демона. Web UI доступний за адресою: http://%s\n", *addr)
		
		// Витягуємо підкаталог ui з embed.FS для коректного маппінгу статичного сервера
		subFS, err := fs.Sub(embedUI, "ui")
		if err != nil {
			fmt.Printf("Помилка ініціалізації вбудованих ресурсів фронтенду: %v\n", err)
			os.Exit(1)
		}

		webServer := server.NewWebServer(*addr, scanner)
		err = webServer.Start(http.FS(subFS))
		if err != nil {
			fmt.Printf("Критична помилка HTTP сервера: %v\n", err)
			os.Exit(1)
		}

	default:
		fmt.Printf("Помилка: невідомий режим роботи '%s'. Доступні варіанти: scan, ui.\n", *mode)
		os.Exit(1)
	}
}
