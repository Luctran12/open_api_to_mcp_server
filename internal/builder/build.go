package builder

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"open_api_to_mcp_server/pkg/openapi"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
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

//go:embed templates/main_template.go.tmpl
var templateContent string

// func BuildExecutable(spec openapi.Spec) (string, error) {
// 	// ✅ Luôn lưu vào /tmp/builds
// 	outputDir := filepath.Join(os.TempDir(), "builds")
// 	if err := os.MkdirAll(outputDir, 0755); err != nil {
// 		return "", fmt.Errorf("failed to create output dir: %w", err)
// 	}

// 	// ✅ Tên file an toàn
// 	safeName := strings.Map(func(r rune) rune {
// 		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' || r == '.' {
// 			return r
// 		}
// 		return '_'
// 	}, spec.Info.Title)

// 	mainPath := filepath.Join(outputDir, fmt.Sprintf("%s_main.go", safeName))
// 	outputPath := filepath.Join(outputDir, fmt.Sprintf("%s.exe", safeName))

// 	// ✅ Load template
// 	// ex, err := os.Executable()
// 	// if err != nil {
// 	// 	panic(err)
// 	// }
// 	//exPath := filepath.Dir(ex)
// 	//templatePath := filepath.Join(exPath, "templates/main_template.go.tmpl")
// 	tmpl, err := template.New("main").Parse(templateContent)


// 	//tmpl, err := template.ParseFiles(templatePath)
// 	if err != nil {
// 		return "", fmt.Errorf("parse template failed: %w", err)
// 	}

// 	// ✅ Chuẩn bị JSON cho template
// 	specJSON, err := json.Marshal(spec)
// 	if err != nil {
// 		return "", fmt.Errorf("marshal spec failed: %w", err)
// 	}

// 	// 👉 Dùng template.JS để không escape
// 	data := map[string]any{
// 		"SpecJSON": string(specJSON),
// 	}

// 	// ✅ Render template
// 	var buf bytes.Buffer
// 	if err := tmpl.Execute(&buf, data); err != nil {
// 		return "", fmt.Errorf("execute template failed: %w", err)
// 	}

// 	if err := os.WriteFile(mainPath, buf.Bytes(), 0644); err != nil {
// 		return "", fmt.Errorf("write main.go failed: %w", err)
// 	}

// 	// ✅ Build file .exe
// 	cmd := exec.Command("go", "build","-mod=mod" ,"-o", outputPath, mainPath)
// 	cmd.Env = append(os.Environ(), "CGO_ENABLED=0","GOOS=windows", "GOARCH=amd64")



// 	out, err := cmd.CombinedOutput()
// 	if err != nil {
// 		return "", fmt.Errorf("build failed: %v\n%s", err, string(out))
// 	}

// 	return outputPath, nil
// }

func sanitize(name string) string {
	result := strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' || r == '.' {
			return r
		}
		return '_'
	}, name)

	// nếu rỗng → đặt tên mặc định
	if result == "" {
		return "output"
	}
	return result
}


func BuildExecutable(spec openapi.Spec) (string, error) {
	// 1. Tạo thư mục tạm cho mỗi lần build
	buildDir, err := os.MkdirTemp("", "openapi-build-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp dir: %w", err)
	}
	log.Println("build dir: ", buildDir)
	// 2. Sinh file main.go
	safeName := sanitize(spec.Info.Title)
	mainPath := filepath.Join(buildDir, "main.go")
	outputPath := filepath.Join(buildDir, safeName)

	tmpl, err := template.New("main").Parse(templateContent)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	specJSON, err := json.Marshal(spec)
	if err != nil {
		return "", fmt.Errorf("failed to marshal spec: %w", err)
	}

	data := map[string]any{"SpecJSON": string(specJSON)}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	if err := os.WriteFile(mainPath, buf.Bytes(), 0644); err != nil {
		return "", fmt.Errorf("failed to write main.go: %w", err)
	}

	// 3. Initialize go module in temp directory
	cmd := exec.Command("go", "mod", "init", "temp-build")
	cmd.Dir = buildDir
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("go mod init failed: %v\n%s", err, string(out))
	}

	// Get project root
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}

	// Add replace directive to go.mod
	goModPath := filepath.Join(buildDir, "go.mod")
	f, err := os.OpenFile(goModPath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return "", fmt.Errorf("failed to open go.mod: %w", err)
	}
	defer f.Close()

	replaceDirective := fmt.Sprintf("\nreplace open_api_to_mcp_server => %s\n", wd)
	if _, err := f.WriteString(replaceDirective); err != nil {
		return "", fmt.Errorf("failed to write replace directive: %w", err)
	}

	cmd = exec.Command("go", "mod", "tidy")
	cmd.Dir = buildDir
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("go mod tidy failed: %v\n%s", err, string(out))
	}

	// 4. Build executable
	cmd = exec.Command("go", "build", "-o", outputPath, mainPath)
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0", "GOOS=linux", "GOARCH=amd64")
	cmd.Dir = buildDir // Chạy go build từ trong thư mục tạm

	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("build failed: %v\n%s", err, string(out))
	}

	// 5. Trả về đường dẫn đầy đủ
	return outputPath, nil
}

