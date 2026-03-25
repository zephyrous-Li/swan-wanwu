package file_extract

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/UnicomAI/wanwu/pkg/log"
	"github.com/UnicomAI/wanwu/pkg/util"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

type ZipFileExtractServiceService struct {
}

var zipFileExtractServiceService = &ZipFileExtractServiceService{}

func init() {
	AddFileExtractService(zipFileExtractServiceService)
}

func (t ZipFileExtractServiceService) ExtractFileType() string {
	return ".zip"
}

func (t ZipFileExtractServiceService) ExtractFile(ctx context.Context, localFilePath string, destDir string) (extractDir string, err error) {
	fileReader, err := zip.OpenReader(localFilePath)
	if err != nil {
		return "", err
	}
	defer func() {
		if err1 := fileReader.Close(); err1 != nil {
			log.Errorf("ZipFileExtractServiceService file close error %v", err)
		}
	}()

	for _, f := range fileReader.File {
		var decodeFileName string
		if f.Flags == 0 { //本地编码，默认GBK，转换成UTF-8
			i := bytes.NewReader([]byte(f.Name))
			decoder := transform.NewReader(i, simplifiedchinese.GB18030.NewDecoder())
			content, _ := io.ReadAll(decoder)
			decodeFileName = string(content)
		} else {
			decodeFileName = f.Name
		}
		if err := util.ValidateFileName(decodeFileName); err != nil {
			log.Warnf("skip unsafe file in zip: %s, error: %v", decodeFileName, err)
			continue
		}
		safe, safePath, err := util.IsSafePath(destDir, decodeFileName)
		if !safe {
			log.Warnf("skip unsafe path in zip: %s, error: %v", decodeFileName, err)
			continue
		}
		destFilePath := safePath
		// 检查是否为目录
		if f.FileInfo().IsDir() {
			// 创建目录
			if err := os.MkdirAll(destFilePath, f.Mode()); err != nil {
				fmt.Printf("无法创建目录: %v\n", err)
			}
			continue
		}
		log.Infof("ExtractFile file path %s", destFilePath)
		// 我们需要确保所有的文件夹都已经创建好
		if err := os.MkdirAll(filepath.Dir(destFilePath), f.Mode()); err != nil {
			return "", err
		}
		//写入文件
		if err := writeUnzipFile(f, destFilePath); err != nil {
			return "", err
		}
	}
	return destDir, nil
}

// writeUnzipFile 写入文件
func writeUnzipFile(zipFile *zip.File, destFilePath string) error {
	//打开目标文件
	destFile, err := os.OpenFile(destFilePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, zipFile.Mode())
	if err != nil {
		return err
	}
	defer func() {
		if err := destFile.Close(); err != nil {
			log.Errorf("ZipFileExtractServiceService file close error %v", err)
		}
	}()

	//打开源压缩文件
	sourceFile, err := zipFile.Open()
	if err != nil {
		return err
	}
	defer func() {
		if err := sourceFile.Close(); err != nil {
			log.Errorf("ZipFileExtractServiceService file close error %v", err)
		}
	}()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}
	return nil
}
