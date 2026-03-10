package util

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type FrontMatter struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

// ParseSkillFrontMatter 解析技能的Markdown内容，提取FrontMatter
func ParseSkillFrontMatter(content string) (*FrontMatter, error) {
	content = strings.TrimSpace(content)

	if !strings.HasPrefix(content, "---") {
		return nil, fmt.Errorf("SKILL.md file must start with front matter delimiters")
	}

	rest := content[3:]
	endIdx := strings.Index(rest, "\n---")
	if endIdx == -1 {
		return nil, fmt.Errorf("SKILL.md file must end with front matter delimiters")
	}

	frontMatterStr := strings.TrimSpace(rest[:endIdx])

	var fm FrontMatter
	if err := yaml.Unmarshal([]byte(frontMatterStr), &fm); err != nil {
		return nil, fmt.Errorf("failed to parse frontmatter: %v", err)
	}

	return &fm, nil
}

// ExtractSkillMarkdownFromZip 从ZIP文件中提取SKILL.md文件的内容，返回完整markdown内容、名称、描述
func ExtractSkillMarkdownFromZip(zipData []byte) (string, *FrontMatter, error) {
	reader, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return "", nil, fmt.Errorf("failed to read zip file: %v", err)
	}

	var skillMdFile *zip.File
	var allFiles []string
	for _, file := range reader.File {
		allFiles = append(allFiles, file.Name)
		fileName := filepath.Base(file.Name)
		if fileName == "SKILL.md" {
			skillMdFile = file
			break
		}
	}

	if skillMdFile == nil {
		return "", nil, fmt.Errorf("SKILL.md file not found in the zip archive. Files: %v", allFiles)
	}

	rc, err := skillMdFile.Open()
	if err != nil {
		return "", nil, fmt.Errorf("failed to open SKILL.md file: %v", err)
	}
	defer func() { _ = rc.Close() }()

	content, err := io.ReadAll(rc)
	if err != nil {
		return "", nil, fmt.Errorf("failed to read SKILL.md file: %v", err)
	}

	markdownContent := string(content)
	fm, err := ParseSkillFrontMatter(markdownContent)
	if err != nil {
		return "", nil, err
	}

	return markdownContent, fm, nil
}
