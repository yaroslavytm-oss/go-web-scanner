package engine

import "testing"

func TestNewDetector(t *testing.T) {
    // Тест на ініціалізацію детектора
    det := NewDetector(50, nil)
    if det == nil {
        t.Error("Не вдалося ініціалізувати детектор")
    }
}
