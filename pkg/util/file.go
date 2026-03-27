package util

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/UnicomAI/wanwu/pkg/log"
	"github.com/google/uuid"
)

const (
	kb               = 1024
	mb               = kb * 1024
	MaxScanTokenSize = 1024 * 1024 // Set the maximum token size to 1 MB
)

var specialFileExtList = []string{".tar.gz"}

type FileMergeResult struct {
	TotalSuccessCount int64
	TotalLineCount    int64
	TotalByteCount    int64
	FilePath          string
}

// ============================================================================
// Public API Functions (公开接口函数)
// ============================================================================

func FileExt(filePath string) string {
	if len(filePath) == 0 {
		return ""
	}
	cleaned := filepath.Clean(filePath)
	if strings.Contains(cleaned, "..") || filepath.IsAbs(cleaned) {
		return ""
	}
	for _, ext := range specialFileExtList {
		if strings.HasSuffix(filePath, ext) {
			return ext
		}
	}
	return filepath.Ext(filePath)
}

func NewRandomFile(fileName string) string {
	return uuid.New().String() + filepath.Ext(fileName)
}

// ToFileSizeStr fileSize单位是B，转换规则：小于1M为KB，大于等于1M，单位为M，保留两位小数
func ToFileSizeStr(fileSize int64) string {
	if fileSize < mb {
		return fmt.Sprintf("%.2f KB", float64(fileSize)/float64(kb))
	} else {
		return fmt.Sprintf("%.2f MB", float64(fileSize)/float64(mb))
	}
}

func FileExist(filePath string) (bool, error) {
	if len(filePath) == 0 {
		return false, nil
	}
	if err := ValidateFileName(filePath); err != nil {
		return false, fmt.Errorf("invalid file path: %w", err)
	}
	_, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func DirFileList(dir string, subDir bool, fullPath bool) ([]string, error) {
	var fileNameList []string
	// 读取目录
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read dir (%v) err: %v", dir, err)
	}

	// 遍历目录下的所有文件和子目录
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			// 处理错误
			log.Errorf("read dir (%v) entry err: %v", dir, err)
			continue
		}

		// 判断是否是文件
		if !info.IsDir() {
			if fullPath {
				fileNameList = append(fileNameList, dir+"/"+entry.Name())
			} else {
				fileNameList = append(fileNameList, entry.Name())
			}
		} else if !subDir { //不需要校验底层目录
			continue
		} else {
			list, err := DirFileList(dir+"/"+entry.Name(), subDir, fullPath)
			if err != nil {
				return nil, err
			} else {
				fileNameList = append(fileNameList, list...)
			}
		}
	}

	return fileNameList, nil
}

// MergeFile 合并文件
func MergeFile(filePathList []string, mergeFilePath string) (*FileMergeResult, error) {
	if err := ValidateFileName(mergeFilePath); err != nil {
		return nil, fmt.Errorf("invalid merge file path: %w", err)
	}
	for _, f := range filePathList {
		if err := ValidateFileName(f); err != nil {
			return nil, fmt.Errorf("invalid file path in list: %w", err)
		}
	}
	// 创建或打开文件
	//0644，表示文件所有者可读写，同组用户及其他用户只可读
	dir := filepath.Dir(mergeFilePath)
	exist, err := FileExist(dir)
	if err != nil {
		return nil, err
	}
	if !exist {
		err = os.MkdirAll(filepath.Dir(mergeFilePath), 0755)
		if err != nil {
			return nil, err
		}
	}

	destinationFile, err := os.OpenFile(mergeFilePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("open merge file (%v) err: %v", mergeFilePath, err)
	}
	defer func() {
		if err := destinationFile.Close(); err != nil {
			log.Errorf("close merge file (%v) err: %v", mergeFilePath, err)
		}
	}()

	var totalByteCount int64
	for _, fileInfo := range filePathList {
		byteCount, err := AppendFileStream(fileInfo, destinationFile)
		if err != nil {
			return nil, fmt.Errorf("merge file (%v) err: %v", mergeFilePath, err)
		}
		totalByteCount += byteCount
	}
	return &FileMergeResult{
		TotalByteCount: totalByteCount,
		FilePath:       mergeFilePath,
	}, nil
}

func DeleteDir(fileDir string) error {
	err := os.RemoveAll(fileDir)
	if err != nil {
		return fmt.Errorf("delete dir (%v) err: %v", fileDir, err)
	}
	return nil
}

func DeleteFile(file string) error {
	if err := ValidateFileName(file); err != nil {
		return fmt.Errorf("invalid file path: %w", err)
	}
	err := os.Remove(file)
	if err != nil {
		return fmt.Errorf("delete file (%v) err: %v", file, err)
	}
	return nil
}

func AppendFileStream(filePath string, destinationFile *os.File) (int64, error) {
	if err := ValidateFileName(filePath); err != nil {
		return 0, fmt.Errorf("invalid file path: %w", err)
	}
	// Open the source file for reading
	sourceFile, err := os.Open(filePath)
	if err != nil {
		return 0, fmt.Errorf("open append file (%v) err: %v", filePath, err)
	}
	defer func() {
		if err := sourceFile.Close(); err != nil {
			log.Errorf("close append file (%v) err: %v", filePath, err)
		}
	}()
	fileReader := bufio.NewReader(sourceFile)
	byteCount, err := appendFile(fileReader, destinationFile)
	if err != nil {
		return 0, fmt.Errorf("append file (%v) to (%v) err: %v", filePath, destinationFile.Name(), err)
	}
	log.Infof("append file (%v) to (%v) succeed, bytes: %v", filePath, destinationFile.Name(), byteCount)
	return byteCount, nil
}

func FileEOF(err error) bool {
	return errors.Is(err, io.EOF) || (err != nil && err.Error() == "EOF")
}

func File2Base64(filePath string, customPrefix string) (base64Str string, base64StrWithPrefix string, err error) {
	fileData, err := os.ReadFile(filePath)
	if err != nil {
		return "", "", err
	}
	return FileData2Base64(fileData, customPrefix)
}

func FileData2Base64(fileData []byte, customPrefix string) (base64Str string, base64StrWithPrefix string, err error) {
	if len(fileData) == 0 {
		return "", "", errors.New("empty file data")
	}

	base64Str = base64.StdEncoding.EncodeToString(fileData)

	var prefix string
	if customPrefix != "" {
		prefix = customPrefix
	} else {
		// 自动检测 MIME 类型
		mimeType := http.DetectContentType(fileData)
		prefix = "data:" + mimeType + ";base64"
	}
	if !strings.Contains(prefix, ",") {
		prefix += ","
	}
	base64StrWithPrefix = prefix + base64Str

	return base64Str, base64StrWithPrefix, nil
}

// FileData2FileHeader
//
//	@Description: 将字节数组转换为multipart.FileHeader
//	@Author zhangzekai
//	@Time 2026-01-21 11:11:20
//	@param filename
//	@param fileData
//	@return *multipart.FileHeader
//	@return error
func FileData2FileHeader(filename string, fileData []byte) (*multipart.FileHeader, error) {
	buf := new(bytes.Buffer)
	writer := multipart.NewWriter(buf)

	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`, "file", filename))
	header.Set("Content-Type", "application/octet-stream") // 可根据实际文件类型修改（如audio/wav）

	part, err := writer.CreatePart(header)
	if err != nil {
		return nil, fmt.Errorf("创建form字段失败: %w", err)
	}
	_, err = part.Write(fileData)
	if err != nil {
		return nil, fmt.Errorf("写入文件数据失败: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("关闭form writer失败: %w", err)
	}

	reader := multipart.NewReader(buf, writer.Boundary())
	form, err := reader.ReadForm(int64(len(fileData)) + 1024)
	if err != nil {
		return nil, fmt.Errorf("解析form数据失败: %w", err)
	}

	fileHeaders := form.File["file"]
	if len(fileHeaders) == 0 {
		return nil, fmt.Errorf("form中未找到file字段")
	}

	return fileHeaders[0], nil
}

// IsSafePath 检查用户提供的路径是否安全，防止路径遍历攻击
//
// 该函数验证 userPath 是否在 baseDir 范围内，防止恶意用户通过 "../" 等方式访问预期目录之外的文件
// 主要用于处理用户可控的文件路径输入场景，如文件上传、下载、解压缩等操作
//
// 参数:
//   - baseDir: 基础目录路径（安全边界），userPath 必须在此目录及其子目录内
//   - userPath: 用户提供的相对路径或文件名，可以是包含目录结构的相对路径（如 "subdir/file.txt"）
//
// 返回值:
//   - bool: 路径是否安全（true=安全，false=不安全）
//   - string: 安全的绝对路径（如果验证通过），可以直接用于文件操作
//   - error: 错误信息（如果路径不安全或处理失败）
//
// 安全检查包括：
//  1. 路径遍历检查：禁止 ".." 等目录遍历字符
//  2. 边界检查：确保最终路径在 baseDir 范围内
//  3. 符号链接检查：解析符号链接后再次验证是否逃逸出基础目录
//  4. 跨平台兼容：正确处理 Windows 和 Unix 系统的路径差异
//
// 使用示例:
//
//	safe, path, err := IsSafePath("/data/uploads", "user123/avatar.jpg")
//	if !safe {
//	    return errors.New("invalid file path")
//	}
//	// 使用 path 进行文件操作
func IsSafePath(baseDir, userPath string) (bool, string, error) {
	if userPath == "" {
		return false, "", fmt.Errorf("path cannot be empty")
	}

	absBase, err := filepath.Abs(baseDir)
	if err != nil {
		return false, "", fmt.Errorf("failed to get absolute base path: %w", err)
	}

	cleaned := filepath.Clean(userPath)

	// 防止路径遍历：检查清理后的路径是否包含".."
	// 注意：filepath.Clean 会处理掉多余的".."，但如果路径以".."开头，清理后仍然会有".."
	if strings.Contains(cleaned, "..") {
		return false, "", fmt.Errorf("path contains traversal sequences")
	}

	fullPath := filepath.Join(absBase, cleaned)

	// 使用 filepath.Abs 获取绝对路径，这不会要求路径存在
	absPath, err := filepath.Abs(fullPath)
	if err != nil {
		return false, "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	// 先规范化路径，处理大小写（Windows）
	normalizedPath := filepath.Clean(absPath)
	if runtime.GOOS == "windows" {
		normalizedPath = strings.ToLower(normalizedPath)
		absBase = strings.ToLower(absBase)
	}

	// 检查路径是否在基础目录内
	if !isPathWithinBase(absBase, normalizedPath) {
		return false, "", fmt.Errorf("path escapes base directory")
	}

	// 尝试解析符号链接，如果文件不存在则跳过
	var resolvedPath string
	if _, err := filepath.EvalSymlinks(absPath); err == nil {
		resolvedPath, err = filepath.EvalSymlinks(absPath)
		if err != nil {
			return false, "", fmt.Errorf("failed to resolve symlinks: %w", err)
		}

		// 重新检查解析后的路径
		if runtime.GOOS == "windows" {
			resolvedPath = strings.ToLower(resolvedPath)
		}
		if !isPathWithinBase(absBase, resolvedPath) {
			return false, "", fmt.Errorf("symlink escapes base directory")
		}
		resolvedPath = filepath.Clean(resolvedPath)
	} else {
		// 文件不存在，使用原始路径
		resolvedPath = absPath
	}

	return true, resolvedPath, nil
}

// ValidateFileName 验证文件名是否安全合法，防止文件名注入攻击
//
// 该函数用于验证单个文件名（不包含路径），确保文件名符合操作系统规范和安全要求
// 适用于验证用户上传的文件名、导出文件名、资源标识符等场景
//
// 参数:
//   - fileName: 待验证的文件名（仅文件名，不应包含路径分隔符）
//
// 返回值:
//   - error: 文件名不合法时返回错误，合法则返回 nil
//
// 验证规则：
//
//  1. 基本检查：
//     - 不能为空字符串
//     - 不能是 "." 或 ".."
//     - 长度不能超过 255 个字符（大多数文件系统限制）
//     - 不能包含路径分隔符（/ 或 \）
//
//  2. 跨平台检查：
//     - Windows: 禁止保留字符（<>:"/\|?*），不能以点或空格结尾，禁止保留设备名（CON、PRN、AUX、NUL、COM1-9、LPT1-9）
//     - Unix/Linux: 禁止空字符（\x00）
//
// 注意事项：
//   - 该函数仅验证文件名，不验证完整路径
//   - 对于完整路径验证，应使用 IsSafePath 函数
//   - 对于相对路径（包含目录结构），应先提取文件名部分再验证
//
// 使用示例:
//
//	if err := ValidateFileName(userInputFileName); err != nil {
//	    return fmt.Errorf("invalid file name: %w", err)
//	}
func ValidateFileName(fileName string) error {
	if fileName == "" {
		return fmt.Errorf("filename cannot be empty")
	}

	if fileName == "." || fileName == ".." {
		return fmt.Errorf("filename cannot be '.' or '..'")
	}

	if len(fileName) > 255 {
		return fmt.Errorf("filename too long")
	}

	// 检查路径分隔符
	if strings.ContainsAny(fileName, "/\\") {
		return fmt.Errorf("filename cannot contain path separators")
	}

	// 清理并验证
	cleaned := filepath.Clean(fileName)
	if cleaned != fileName && cleaned != "." && cleaned != ".." {
		return fmt.Errorf("filename contains invalid patterns")
	}

	// 操作系统特定检查
	if runtime.GOOS == "windows" {
		return validateWindowsFileName(fileName)
	}

	return validateUnixFileName(fileName)
}

// ============================================================================
// Internal Helper Functions (内部辅助函数)
// ============================================================================

func appendFile(reader *bufio.Reader, destinationFile *os.File) (byteCount int64, error error) {
	buf := make([]byte, MaxScanTokenSize)
	for {
		n, err := reader.Read(buf)
		if FileEOF(err) { // 检查是否到达文件末尾
			break
		}
		if err != nil {
			log.Errorf("Error reading file: %s", err)
			return -1, err
		}
		line := buf[:n]
		bytesWritten, err := destinationFile.Write(line)
		if err != nil {
			log.Errorf("appendFile error %s", err)
			return -1, err
		}
		byteCount += int64(bytesWritten)
	}
	return byteCount, nil
}

// isPathWithinBase 检查 target 路径是否在 base 目录内（包含 base 本身）
// 用于 IsSafePath 的辅助函数，确保路径不会逃逸出基础目录
func isPathWithinBase(base, target string) bool {
	if target == base {
		return true
	}

	if !strings.HasPrefix(target, base) {
		return false
	}

	if len(target) > len(base) {
		sep := string(filepath.Separator)
		if !strings.HasPrefix(target[len(base):], sep) {
			return false
		}
	}

	return true
}

func validateWindowsFileName(fileName string) error {
	// Windows保留字符
	reservedChars := `<>:"/\|?*`
	if strings.ContainsAny(fileName, reservedChars) {
		return fmt.Errorf("filename contains invalid characters")
	}

	// 检查结尾的点和空格
	if strings.HasSuffix(fileName, ".") || strings.HasSuffix(fileName, " ") {
		return fmt.Errorf("filename cannot end with dot or space")
	}

	// 检查保留的设备名
	nameWithoutExt := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	upperName := strings.ToUpper(nameWithoutExt)

	reservedNames := map[string]bool{
		"CON": true, "PRN": true, "AUX": true, "NUL": true,
		"COM1": true, "COM2": true, "COM3": true, "COM4": true,
		"COM5": true, "COM6": true, "COM7": true, "COM8": true, "COM9": true,
		"LPT1": true, "LPT2": true, "LPT3": true, "LPT4": true,
		"LPT5": true, "LPT6": true, "LPT7": true, "LPT8": true, "LPT9": true,
	}

	if reservedNames[upperName] {
		return fmt.Errorf("filename is a reserved device name")
	}

	return nil
}

func validateUnixFileName(fileName string) error {
	// Unix/Linux 基本检查
	if strings.Contains(fileName, "\x00") {
		return fmt.Errorf("filename contains null character")
	}

	// Unix文件名不能包含路径分隔符，已在主函数检查
	return nil
}
