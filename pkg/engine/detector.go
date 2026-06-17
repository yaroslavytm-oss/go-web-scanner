package engine

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"regexp"
)

// Signature представляє правило детектування
type Signature struct {
	Name        string
	Regex       *regexp.Regexp
	DangerScore int
}

// Detector містить набір сигнатур для евристичного аналізу
type Detector struct {
	signatures []Signature
	threshold  int
}

// NewDetector ініціалізує евристичний двигун базовими правилами
func NewDetector(threshold int) *Detector {
	// Базовий набір сигнатур для MVP (детект веб-шеллів та бекдорів)
	rawSigns := []struct {
		Name  string
		Expr  string
		Score int
	}{
		{"Suspicious Eval with Request", `eval\s*\(\s*\$(?:_POST|_GET|_REQUEST|_COOKIE)`, 80},
		{"Base64 Decode with Exec", `(?:exec|shell_exec|system|passthru)\s*\(\s*base64_decode`, 90},
		{"PHP Alternative Syntax Execution", `\$\s*_\s*(?:POST|GET|REQUEST)\s*\[.*\]\s*\(\s*\$\s*_\s*(?:POST|GET|REQUEST)`, 95},
		{"Obfuscated Global Variables Access", `\$GLOBALS\s*\[\s*['"][a-zA-Z0-9_\x7f-\xff]+['"]\s*\]\s*\(`, 70},
		{"Silent Error Execution Operator", `@\s*(?:eval|exec|system|passthru)\s*\(`, 40},
		{"Dynamic Function Call from Variable", `\$[a-zA-Z_\x7f-\xff][a-zA-Z0-9_\x7f-\xff]*\s*\(\s*\$(?:_POST|_GET|_REQUEST)`, 75},
		{"Suspicious Include / Require", `(?:include|require)(?:_once)?\s*\(?\s*['"]data\s*:.*base64`, 85},
		{"PHP Backdoor Functions Usage", `(?:passthru|shell_exec|system|popen|proc_open|assert)\s*\(`, 25},
	}

	var compiled []Signature
	for _, rs := range rawSigns {
		re := regexp.MustCompile("(?i)" + rs.Expr) // Case-insensitive
		compiled = append(compiled, Signature{Name: rs.Name, Regex: re, DangerScore: rs.Score})
	}

	return &Detector{
		signatures: compiled,
		threshold:  threshold,
	}
}

// Result містить інформацію про результати аналізу файлу
type Result struct {
	FilePath    string   `json:"file_path"`
	MD5         string   `json:"md5"`
	SHA256      string   `json:"sha256"`
	IsInfected  bool     `json:"is_infected"`
	DangerScore int      `json:"danger_score"`
	Matches     []string `json:"matches"`
}

// AnalyzeFile виконує перевірку файлу за конвеєром (Хеші -> Евристика)
func (d *Detector) AnalyzeFile(path string) (Result, error) {
	res := Result{FilePath: path, Matches: make([]string, 0)}

	file, err := os.Open(path)
	if err != nil {
		return res, err
	}
	defer file.Close()

	// Обчислення хешів та читання контенту за один прохід через MultiWriter
	hMD5 := md5.New()
	hSHA256 := sha256.New()
	
	// Обмеження пам'яті: зчитуємо файл повністю (макс розмір вже перевірено у WalkDir)
	content, err := io.ReadAll(io.TeeReader(file, io.MultiWriter(hMD5, hSHA256)))
	if err != nil {
		return res, err
	}

	res.MD5 = hex.EncodeToString(hMD5.Sum(nil))
	res.SHA256 = hex.EncodeToString(hSHA256.Sum(nil))

	// Евристичний аналіз регулярними виразами
	contentStr := string(content)
	totalScore := 0

	for _, sig := range d.signatures {
		if sig.Regex.MatchString(contentStr) {
			totalScore += sig.DangerScore
			res.Matches = append(res.Matches, sig.Name)
		}
	}

	res.DangerScore = totalScore
	if totalScore >= d.threshold {
		res.IsInfected = true
	}

	return res, nil
}
