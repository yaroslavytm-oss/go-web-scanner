package engine

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"io/fs"
	"math"
	"os"
	"regexp"
	"strings"
)

type Signature struct {
	Name        string
	Regex       *regexp.Regexp
	DangerScore int
}

type Detector struct {
	signatures []Signature
	threshold  int
}

type Result struct {
	FilePath    string   `json:"file_path"`
	MD5         string   `json:"md5"`
	SHA256      string   `json:"sha256"`
	IsInfected  bool     `json:"is_infected"`
	DangerScore int      `json:"danger_score"`
	Matches     []string `json:"matches"`
}

func NewDetector(threshold int, signatures []Signature) *Detector {
	return &Detector{
		signatures: signatures,
		threshold:  threshold,
	}
}

func LoadSignatures(fsys fs.FS, path string) ([]Signature, error) {
	file, err := fs.ReadFile(fsys, path)
	if err != nil {
		return nil, err
	}
	
	var raw []struct {
		Name  string `json:"Name"`
		Regex string `json:"Regex"`
		Score int    `json:"Score"`
	}
	
	if err := json.Unmarshal(file, &raw); err != nil {
		return nil, err
	}

	var sigs []Signature
	for _, r := range raw {
		sigs = append(sigs, Signature{
			Name:        r.Name,
			Regex:       regexp.MustCompile("(?i)" + r.Regex),
			DangerScore: r.Score,
		})
	}
	return sigs, nil
}

func CalculateEntropy(data []byte) float64 {
	if len(data) == 0 { return 0.0 }
	frequencies := make(map[byte]int)
	for _, b := range data { frequencies[b]++ }
	var entropy float64
	for _, count := range frequencies {
		p := float64(count) / float64(len(data))
		entropy -= p * math.Log2(p)
	}
	return entropy
}

func hasAbnormallyLongLines(content string) bool {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		if len(line) > 5000 && !strings.Contains(line, " ") { return true }
	}
	return false
}

func (d *Detector) AnalyzeFile(path string) (Result, error) {
	res := Result{FilePath: path, Matches: make([]string, 0)}
	file, err := os.Open(path)
	if err != nil { return res, err }
	defer file.Close()

	hMD5 := md5.New()
	hSHA256 := sha256.New()
	content, err := io.ReadAll(io.TeeReader(file, io.MultiWriter(hMD5, hSHA256)))
	if err != nil { return res, err }

	res.MD5 = hex.EncodeToString(hMD5.Sum(nil))
	res.SHA256 = hex.EncodeToString(hSHA256.Sum(nil))

	totalScore := 0
	contentStr := string(content)

	if CalculateEntropy(content) > 5.9 {
		totalScore += 100
		res.Matches = append(res.Matches, "High Shannon Entropy")
	}
	if hasAbnormallyLongLines(contentStr) {
		totalScore += 60
		res.Matches = append(res.Matches, "Abnormally Long Lines")
	}
	for _, sig := range d.signatures {
		if sig.Regex.MatchString(contentStr) {
			totalScore += sig.DangerScore
			res.Matches = append(res.Matches, sig.Name)
		}
	}
	res.DangerScore = totalScore
	res.IsInfected = totalScore >= d.threshold
	return res, nil
}
