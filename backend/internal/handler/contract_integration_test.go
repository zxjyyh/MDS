package handler

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/zxjyyh/mds/internal/config"
	"github.com/zxjyyh/mds/internal/database"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// TestGenerateContractE2E 端到端验证：临时库建 personnel → GenerateContract → docx 可解包且无残留占位符。
func TestGenerateContractE2E(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tmp := filepath.Join(os.TempDir(), fmt.Sprintf("mds_ct_%d.db", time.Now().UnixNano()))
	defer os.Remove(tmp)

	db, err := database.NewDB(tmp)
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}
	defer db.Close()
	if err := database.AutoMigrate(db); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	_, err = db.Exec(`INSERT INTO personnel (name, gender, id_card, department, address, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		"张三", "unknown", "110105199001010011", "示例市示例机动车检测有限公司",
		"示例市示例镇解放路 1 号", "active", now, now)
	if err != nil {
		t.Fatalf("insert personnel: %v", err)
	}
	var id int
	if err := db.Get(&id, "SELECT id FROM personnel LIMIT 1"); err != nil {
		t.Fatalf("get id: %v", err)
	}

	cfg := config.Load()
	h := NewHandler(db, cfg, zap.NewNop())

	body := fmt.Sprintf(`{"personnelId":%d,"signDate":"2026-09-06","stationName":"","stationAddress":"福建省示例市示例区示例路 1 号","years":3}`, id)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/personnel/docs/contract", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.GenerateContract(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); !strings.Contains(ct, "wordprocessingml") {
		t.Fatalf("unexpected content-type %q", ct)
	}
	data := w.Body.Bytes()
	if len(data) == 0 {
		t.Fatal("empty docx body")
	}
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("invalid docx: %v", err)
	}
	var xml string
	for _, f := range zr.File {
		if f.Name == "word/document.xml" {
			rc, _ := f.Open()
			buf := new(bytes.Buffer)
			_, _ = io.Copy(buf, rc)
			_ = rc.Close()
			xml = buf.String()
		}
	}
	for _, want := range []string{"张三", "男", "110105199001010011", "2026年09月06日", "2029年09月05日", "示例市示例机动车检测有限公司"} {
		if !strings.Contains(xml, want) {
			t.Errorf("document.xml missing %q", want)
		}
	}
	if strings.Contains(xml, "{{") || strings.Contains(xml, "户口所在地") {
		t.Error("unexpected leftover token / 户口所在地 in generated docx")
	}
}

// TestGenerateContractErrors 校验缺身份证/地址等拦截逻辑。
func TestGenerateContractErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tmp := filepath.Join(os.TempDir(), fmt.Sprintf("mds_ct_err_%d.db", time.Now().UnixNano()))
	defer os.Remove(tmp)
	db, err := database.NewDB(tmp)
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}
	defer db.Close()
	_ = database.AutoMigrate(db)
	now := time.Now().Format("2006-01-02 15:04:05")
	_, _ = db.Exec(`INSERT INTO personnel (name, gender, department, status, created_at, updated_at)
		VALUES ('李四', 'male', '某检测站', 'active', ?, ?)`, now, now)
	var id int
	_ = db.Get(&id, "SELECT id FROM personnel LIMIT 1")

	h := NewHandler(db, config.Load(), zap.NewNop())

	cases := []struct {
		name string
		body string
	}{
		{"no personnel", `{"personnelId":0,"signDate":"2026-09-06","stationAddress":"x"}`},
		{"missing idcard+addr", fmt.Sprintf(`{"personnelId":%d,"signDate":"2026-09-06","stationAddress":"x"}`, id)},
		{"bad date", fmt.Sprintf(`{"personnelId":%d,"signDate":"2026/09/06","stationAddress":"x"}`, id)},
	}
	for _, tc := range cases {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tc.body))
		c.Request.Header.Set("Content-Type", "application/json")
		h.GenerateContract(c)
		if w.Code != http.StatusBadRequest {
			t.Errorf("[%s] want 400, got %d body=%s", tc.name, w.Code, w.Body.String())
		}
	}
}
