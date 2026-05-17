package httpapi

import (
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"demo1/internal/service"

	"github.com/gin-gonic/gin"
)

const maxMaterialFileSize = 10 << 20

type materialUploadResponse struct {
	Name        string `json:"name"`
	URL         string `json:"url"`
	Size        int64  `json:"size"`
	ContentType string `json:"contentType"`
}

func (s *Server) uploadMaterial(c *gin.Context) {
	header, err := c.FormFile("file")
	if err != nil {
		writeError(c, err)
		return
	}
	if header.Size <= 0 {
		writeError(c, errors.New("上传文件不能为空"))
		return
	}
	if header.Size > maxMaterialFileSize {
		writeError(c, fmt.Errorf("单个材料文件不能超过 %dMB", maxMaterialFileSize>>20))
		return
	}

	contentType, ext, err := detectMaterialType(header)
	if err != nil {
		writeError(c, err)
		return
	}
	if err := os.MkdirAll(s.cfg.UploadDir, 0o755); err != nil {
		writeError(c, err)
		return
	}

	storedName := service.NewID("mat") + ext
	targetPath := filepath.Join(s.cfg.UploadDir, storedName)
	source, err := header.Open()
	if err != nil {
		writeError(c, err)
		return
	}
	defer source.Close()

	target, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		writeError(c, err)
		return
	}
	defer target.Close()

	if _, err := io.Copy(target, source); err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusCreated, materialUploadResponse{
		Name:        cleanOriginalFilename(header.Filename),
		URL:         "/api/materials/" + storedName,
		Size:        header.Size,
		ContentType: contentType,
	})
}

func (s *Server) downloadMaterial(c *gin.Context) {
	name := strings.TrimSpace(c.Param("name"))
	if name == "" || name != filepath.Base(name) {
		writeError(c, errors.New("材料文件不存在"))
		return
	}
	c.File(filepath.Join(s.cfg.UploadDir, name))
}

func detectMaterialType(header *multipart.FileHeader) (string, string, error) {
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext == ".jpeg" {
		ext = ".jpg"
	}
	if ext != ".pdf" && ext != ".png" && ext != ".jpg" {
		return "", "", errors.New("仅支持上传 PDF、PNG、JPG 格式材料")
	}

	file, err := header.Open()
	if err != nil {
		return "", "", err
	}
	defer file.Close()

	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && !errors.Is(err, io.EOF) {
		return "", "", err
	}
	contentType := http.DetectContentType(buffer[:n])
	switch contentType {
	case "application/pdf":
		if ext != ".pdf" {
			return "", "", errors.New("文件内容与扩展名不匹配")
		}
		return contentType, ext, nil
	case "image/png":
		if ext != ".png" {
			return "", "", errors.New("文件内容与扩展名不匹配")
		}
		return contentType, ext, nil
	case "image/jpeg":
		if ext != ".jpg" {
			return "", "", errors.New("文件内容与扩展名不匹配")
		}
		return contentType, ext, nil
	default:
		return "", "", errors.New("仅支持上传 PDF、PNG、JPG 格式材料")
	}
}

func cleanOriginalFilename(name string) string {
	name = strings.TrimSpace(filepath.Base(name))
	if name == "" || name == "." {
		return "证明材料"
	}
	return name
}
