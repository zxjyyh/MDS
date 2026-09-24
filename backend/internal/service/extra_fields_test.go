package service

import (
	"testing"

	"github.com/zxjyyh/mds/internal/scope"
)

// 新增字段的读写往返测试（人员：安检岗位 / 环保岗位；设备：检定单位 / 检定费用）。
//
// 为什么单独写这个：给表加列再往 INSERT/UPDATE 里补占位符，最容易出现
// 「列数 / 问号数 / 参数个数」三者不匹配，而这种错误**编译期查不出来**，
// 只有真正执行到那条 SQL 才报 "N values for M columns"。
// 既有测试里 personnel.Create 只走成功路径一次、equipment.Create 只走了被拒的分支，
// UPDATE 更是没走到过 SQL，所以这里专门把增改读三条路径都跑一遍。
func TestPersonnelPositionsRoundTrip(t *testing.T) {
	db := openCertClearTestDB(t)
	seedScopeFixture(t, db)
	svc := &PersonnelService{DB: db}
	all := scope.All()

	const (
		safety = "引车员,外观检验员"
		env    = "登录员、OBD查验员" // 选项本身含顿号，所以分隔符必须用半角逗号
	)

	p, err := svc.Create(all, map[string]interface{}{
		"name": "岗位往返", "gender": "男", "status": "active", "department": "甲检测站",
		"securityPositions": safety,
		"envPositions":      env,
	}, 1)
	if err != nil {
		t.Fatalf("创建人员失败（检查 INSERT 的列/占位符/参数是否对齐）: %v", err)
	}

	got, err := svc.GetByID(all, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.SecurityPositions == nil || *got.SecurityPositions != safety {
		t.Fatalf("安检岗位写入后读回不一致: %v", got.SecurityPositions)
	}
	if got.EnvPositions == nil || *got.EnvPositions != env {
		t.Fatalf("环保岗位写入后读回不一致: %v", got.EnvPositions)
	}

	// 列表走的是 SELECT p.*，顺带确认新列也能带出来
	list, _, err := svc.List(all, "", "", "", 1, 50)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for i := range list {
		if list[i].ID == p.ID {
			found = true
			if list[i].SecurityPositions == nil || *list[i].SecurityPositions != safety {
				t.Fatalf("列表未带出安检岗位: %v", list[i].SecurityPositions)
			}
		}
	}
	if !found {
		t.Fatal("列表中找不到刚创建的人员")
	}

	// 更新：清空安检岗位、改写环保岗位
	if err := svc.Update(all, p.ID, map[string]interface{}{
		"name": "岗位往返", "gender": "男", "status": "active",
		"securityPositions": nil,
		"envPositions":      "引车员",
	}); err != nil {
		t.Fatalf("更新人员失败（检查 UPDATE 的列/占位符/参数是否对齐）: %v", err)
	}
	got2, err := svc.GetByID(all, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got2.SecurityPositions != nil {
		t.Fatalf("安检岗位应已清空，实际为 %q", *got2.SecurityPositions)
	}
	if got2.EnvPositions == nil || *got2.EnvPositions != "引车员" {
		t.Fatalf("环保岗位未更新成功: %v", got2.EnvPositions)
	}
}

func TestEquipmentCalibrationFieldsRoundTrip(t *testing.T) {
	db := openCertClearTestDB(t)
	seedScopeFixture(t, db)
	svc := &EquipmentService{DB: db}
	all := scope.All()

	// 费用刻意用自由文本：可能一次送检多项、每项一个价
	const (
		unit = "示例计量检定院"
		fee  = "外观 300，制动 500"
	)

	e, err := svc.Create(all, map[string]interface{}{
		"name": "设备往返", "serialNumber": "SN-RT", "status": "in_use", "department": "甲检测站",
		"calibrationUnit": unit,
		"calibrationFee":  fee,
	})
	if err != nil {
		t.Fatalf("创建设备失败（检查 INSERT 的列/占位符/参数是否对齐）: %v", err)
	}

	got, err := svc.GetByID(all, e.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.CalibrationUnit == nil || *got.CalibrationUnit != unit {
		t.Fatalf("检定单位写入后读回不一致: %v", got.CalibrationUnit)
	}
	if got.CalibrationFee == nil || *got.CalibrationFee != fee {
		t.Fatalf("检定费用写入后读回不一致: %v", got.CalibrationFee)
	}

	// 注意：这两个 Update 都是"整行覆盖"，name / serialNumber / status 都是 NOT NULL，
	// 所以即便只改新字段，也必须把这三列带上（前端也是提交整个表单）。
	if err := svc.Update(all, e.ID, map[string]interface{}{
		"name": "设备往返", "serialNumber": "SN-RT", "status": "in_use",
		"calibrationUnit": "改后的检定院",
		"calibrationFee":  nil,
	}); err != nil {
		t.Fatalf("更新设备失败（检查 UPDATE 的列/占位符/参数是否对齐）: %v", err)
	}
	got2, err := svc.GetByID(all, e.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got2.CalibrationUnit == nil || *got2.CalibrationUnit != "改后的检定院" {
		t.Fatalf("检定单位未更新成功: %v", got2.CalibrationUnit)
	}
	if got2.CalibrationFee != nil {
		t.Fatalf("检定费用应已清空，实际为 %q", *got2.CalibrationFee)
	}
}
