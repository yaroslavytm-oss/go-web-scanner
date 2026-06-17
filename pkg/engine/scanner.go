package engine

import (
	"context"
	"io/fs"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Scanner Config
const (
	MaxFileSize     = 10 * 1024 * 1024 // 10 MB
	ThrottleDelayMS = 10               // Пауза у мс для зниження навантаження на I/O
)

type Scanner struct {
	detector      *Detector
	numWorkers    int
	filesScanned  int64
	virusesFound  int64
	infectedFiles []Result
	mu            sync.RWMutex
	isScanning    int32
}

func NewScanner(detector *Detector, workers int) *Scanner {
	if workers <= 0 {
		workers = 2
	}
	return &Scanner{
		detector:      detector,
		numWorkers:    workers,
		infectedFiles: make([]Result, 0),
	}
}

// GetStats повертає поточну статистику процесу
func (s *Scanner) GetStats() (bool, int64, int64, []Result) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	// Глибоке копіювання результатів для запобігання race conditions при читанні з API
	copiedResults := make([]Result, len(s.infectedFiles))
	copy(copiedResults, s.infectedFiles)

	return atomic.LoadInt32(&s.isScanning) == 1,
		atomic.LoadInt64(&s.filesScanned),
		atomic.LoadInt64(&s.virusesFound),
		copiedResults
}

// Reset очищує метрики перед новим запуском
func (s *Scanner) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.infectedFiles = make([]Result, 0)
	atomic.StoreInt64(&s.filesScanned, 0)
	atomic.StoreInt64(&s.virusesFound, 0)
}

// StartScan запускає багатопотоковий обхід та сканування директорії
func (s *Scanner) StartScan(ctx context.Context, rootPath string) {
	if !atomic.CompareAndSwapInt32(&s.isScanning, 0, 1) {
		return // Вже сканується
	}
	
	s.Reset()

	defer atomic.StoreInt32(&s.isScanning, 0)

	jobs := make(chan string, 1000)
	var wg sync.WaitGroup

	// Ініціалізація пулу воркерів
	for i := 0; i < s.numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range jobs {
				select {
				case <-ctx.Done():
					return
				default:
					// Захист Disk I/O: мікропауза перед кожною операцією читання
					time.Sleep(time.Duration(ThrottleDelayMS) * time.Millisecond)

					res, err := s.detector.AnalyzeFile(path)
					if err != nil {
						continue
					}

					atomic.AddInt64(&s.filesScanned, 1)
					if res.IsInfected {
						atomic.AddInt64(&s.virusesFound, 1)
						s.mu.Lock()
						s.infectedFiles = append(s.infectedFiles, res)
						s.mu.Unlock()
					}
				}
			}
		}()
	}

	// Обхід папок (продюсер)
	_ = filepath.WalkDir(rootPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		
		select {
		case <-ctx.Done():
			return filepath.SkipDir
		default:
			if !d.IsDir() {
				ext := strings.ToLower(filepath.Ext(path))
				// MVP фільтрує тільки файли інтерпретатора PHP
				if ext == ".php" || ext == ".phtml" || ext == ".php5" {
					info, err := d.Info()
					if err == nil && info.Size() <= MaxFileSize {
						jobs <- path
					}
				}
			}
		}
		return nil
	})

	close(jobs)
	wg.Wait() // Очікування завершення обробки черги воркерами
}
