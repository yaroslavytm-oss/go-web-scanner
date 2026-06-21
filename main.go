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
)

// Вшиваємо UI та JSON
//go:embed ui/*
//go:embed signatures.json
var embeddedFiles embed.FS

func main() {
	mode := flag.String("mode", "ui", "Режим: 'scan' або 'ui'")
	path := flag.String("path", "", "Шлях до директорії")
	addr := flag.String("addr", "0.0.0.0:9999", "Адреса Web UI")
	workers := flag.Int("workers", runtime.NumCPU(), "Потоки")
	threshold := flag.Int("threshold", 50, "Поріг Danger Score")

	flag.Parse()

	// Завантажуємо вшиті сигнатури
	sigs, err := engine.LoadSignatures(embeddedFiles, "signatures.json")
	if err != nil {
		fmt.Printf("[!] Помилка завантаження сигнатур: %v\n", err)
		os.Exit(1)
	}

	detector := engine.NewDetector(*threshold, sigs)
	scanner := engine.NewScanner(detector, *workers)

	switch *mode {
	case "scan":
		if *path == "" {
			fmt.Println("Помилка: параметр --path обов'язковий.")
			os.Exit(1)
		}
		
		fmt.Printf("[*] Запуск сканування: %s\n", *path)
		scanner.StartScan(context.Background(), *path)
		
		_, scanned, found, list := scanner.GetStats()
		fmt.Printf("\nЗвіт: Перевірено %d файлів, знайдено %d загроз\n", scanned, found)
		for _, file := range list {
			fmt.Printf("[-] ЗАГРОЗА: %s\n", file.FilePath)
		}
		if found > 0 { os.Exit(1) }

	case "ui":
		fmt.Printf("[*] Web UI на http://%s\n", *addr)
		subFS, _ := fs.Sub(embeddedFiles, "ui")
		webServer := server.NewWebServer(*addr, scanner)
		webServer.Start(http.FS(subFS))

	default:
		fmt.Println("Невірний режим.")
	}
}
