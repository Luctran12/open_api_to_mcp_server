package builder

import (
	"bytes"
	"encoding/json"
	"fmt"
	"text/template"
	"open_api_to_mcp_server/pkg/openapi"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unicode"
)

// func BuildExecutable(spec openapi.Spec) (string, error) {
// 	// 1️⃣ Tạo thư mục tạm cho build
// 	buildDir, err := os.MkdirTemp("", "build-*")

// 	// 2️⃣ Sinh file main.go từ template
// 	mainPath := filepath.Join(buildDir, "main.go")

// 	tmpl, _ := template.ParseFiles("templates/main_template.go.tmpl")
// 	buf := new(bytes.Buffer)

// 	specBytes, _ := json.Marshal(spec)
// 	escaped := strings.ReplaceAll(string(specBytes), "`", "` + \"`\" + `")

// 	tmpl.Execute(buf, map[string]string{"SpecJSON": escaped}) // change cfg to spec
// 	os.WriteFile(mainPath, buf.Bytes(), 0644)

// 	// 3️⃣ Gọi go build -> đây là nơi file exe được tạo ra!
// 	outputPath := filepath.Join(buildDir, spec.Info.Title+".exe")
// 	cmd := exec.Command("go", "build", "-o", outputPath, mainPath)
// 	cmd.Env = append(os.Environ(), "GOOS=windows", "GOARCH=amd64") // build cho Windows
// 	out, err := cmd.CombinedOutput()
// 	if err != nil {
// 		return "", fmt.Errorf("build failed: %v\n%s", err, string(out))
// 	}

// 	// 4️⃣ Trả về đường dẫn tới file exe đã build xong
// 	return outputPath, nil
// }

// func BuildExecutable(spec openapi.Spec) (string, error) {

// 	// ✅ Luôn lưu vào /tmp/builds
// 	outputDir := "/tmp/builds"
// 	if err := os.MkdirAll(outputDir, 0755); err != nil {
// 		return "", fmt.Errorf("failed to create output dir: %w", err)
// 	}

// 	// file main tạm nằm trong /tmp/builds
// 	mainPath := filepath.Join(outputDir, "main.go")

// 	// 1️⃣ Tạo thư mục tạm cho build
// 	//buildDir, err := os.MkdirTemp("", "build-*")
// 	// if err != nil {
// 	// 	return "", fmt.Errorf("create temp dir failed: %w", err)
// 	// }

// 	// 2️⃣ Sinh file main.go từ template
// 	//mainPath := filepath.Join(buildDir, "main.go")

// 	tmpl, err := template.ParseFiles("templates/main_template.go.tmpl")
// 	if err != nil {
// 		return "", fmt.Errorf("parse template failed: %w", err)
// 	}

// 	buf := new(bytes.Buffer)

// 	specBytes, err := json.Marshal(spec)
// 	if err != nil {
// 		return "", fmt.Errorf("marshal spec failed: %w", err)
// 	}

// 	// ✅ Escape chuỗi JSON để nhúng an toàn trong Go source code
// 	escaped := strings.ReplaceAll(string(specBytes), "`", "` + \"`\" + `")

// 	err = tmpl.Execute(buf, map[string]string{
// 		"SpecJSON": escaped,
// 	})
// 	if err != nil {
// 		return "", fmt.Errorf("execute template failed: %w", err)
// 	}

// 	if err := os.WriteFile(mainPath, buf.Bytes(), 0644); err != nil {
// 		return "", fmt.Errorf("write main.go failed: %w", err)
// 	}

// 	// 3️⃣ Gọi go build để sinh file .exe
// 	outputPath := filepath.Join(outputDir, fmt.Sprintf("%s.exe", spec.Info.Title))
// 	cmd := exec.Command("go", "build", "-o", outputPath, mainPath)
// 	cmd.Env = append(os.Environ(), "GOOS=windows", "GOARCH=amd64")

// 	out, err := cmd.CombinedOutput()
// 	if err != nil {
// 		return "", fmt.Errorf("build failed: %v %s", err, string(out))
// 	}

// 	return outputPath, nil
// }

func BuildExecutable(spec openapi.Spec) (string, error) {
	// ✅ Luôn lưu vào /tmp/builds
	outputDir := filepath.Join(os.TempDir(), "builds")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output dir: %w", err)
	}

	// ✅ Tên file an toàn
	safeName := strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' || r == '.' {
			return r
		}
		return '_'
	}, spec.Info.Title)

	mainPath := filepath.Join(outputDir, fmt.Sprintf("%s_main.go", safeName))
	outputPath := filepath.Join(outputDir, fmt.Sprintf("%s.exe", safeName))

	// ✅ Load template
	tmpl, err := template.ParseFiles("templates/main_template.go.tmpl")
	if err != nil {
		return "", fmt.Errorf("parse template failed: %w", err)
	}

	// ✅ Chuẩn bị JSON cho template
	specJSON, err := json.Marshal(spec)
	if err != nil {
		return "", fmt.Errorf("marshal spec failed: %w", err)
	}

	// 👉 Dùng template.JS để không escape
	data := map[string]any{
		"SpecJSON": string(specJSON),
	}

	// ✅ Render template
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute template failed: %w", err)
	}

	if err := os.WriteFile(mainPath, buf.Bytes(), 0644); err != nil {
		return "", fmt.Errorf("write main.go failed: %w", err)
	}

	// ✅ Build file .exe
	cmd := exec.Command("go", "build", "-o", outputPath, mainPath)
	cmd.Env = append(os.Environ(), "GOOS=windows", "GOARCH=amd64")

	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("build failed: %v\n%s", err, string(out))
	}

	return outputPath, nil
}



