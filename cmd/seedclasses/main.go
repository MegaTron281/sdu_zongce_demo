package main

import (
	"log"

	"demo1/internal/config"
	"demo1/internal/domain"
	"demo1/internal/repository"
	"demo1/internal/service"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// 测试账号矩阵：4 个班级，每班 1 班委 + ≥2 学生；2 名辅导员各管辖 2 个班级。
// 幂等：已存在的账号只校正班级/角色/管辖班级，不覆盖密码。
type accountSpec struct {
	ID             string
	Role           string
	DisplayName    string
	StudentID      string
	ClassName      string
	Account        string
	ManagedClasses []string // 仅 admin 使用
	IsSuperAdmin   bool     // 仅系统管理员使用
}

var accountSpecs = []accountSpec{
	// 公信23（已有 金瑞开 / 巩浩，补一名学生）
	{ID: "user-stu-23b", Role: domain.RoleStudent, DisplayName: "刘晓", StudentID: "202300271019", ClassName: "公信23", Account: "202300271019"},
	// 公信24
	{ID: "user-rev-24", Role: domain.RoleReviewer, DisplayName: "孙倩", StudentID: "202300241001", ClassName: "公信24", Account: "202300241001"},
	{ID: "user-stu-24a", Role: domain.RoleStudent, DisplayName: "周宇", StudentID: "202300241002", ClassName: "公信24", Account: "202300241002"},
	{ID: "user-stu-24b", Role: domain.RoleStudent, DisplayName: "吴桐", StudentID: "202300241003", ClassName: "公信24", Account: "202300241003"},
	// 公信25
	{ID: "user-rev-25", Role: domain.RoleReviewer, DisplayName: "郑凯", StudentID: "202300251001", ClassName: "公信25", Account: "202300251001"},
	{ID: "user-stu-25a", Role: domain.RoleStudent, DisplayName: "冯雪", StudentID: "202300251002", ClassName: "公信25", Account: "202300251002"},
	{ID: "user-stu-25b", Role: domain.RoleStudent, DisplayName: "许亮", StudentID: "202300251003", ClassName: "公信25", Account: "202300251003"},
	// 公信26
	{ID: "user-rev-26", Role: domain.RoleReviewer, DisplayName: "何静", StudentID: "202300261001", ClassName: "公信26", Account: "202300261001"},
	{ID: "user-stu-26a", Role: domain.RoleStudent, DisplayName: "卢鑫", StudentID: "202300261002", ClassName: "公信26", Account: "202300261002"},
	{ID: "user-stu-26b", Role: domain.RoleStudent, DisplayName: "高媛", StudentID: "202300261003", ClassName: "公信26", Account: "202300261003"},
}

var adminSpecs = []accountSpec{
	{ID: "user-admin-1", Role: domain.RoleAdmin, DisplayName: "辅导员张苗", ClassName: "政治学与公共管理学院", Account: "zhangmiao", ManagedClasses: []string{"公信23", "公信24"}},
	{ID: "user-admin-2", Role: domain.RoleAdmin, DisplayName: "辅导员王强", ClassName: "政治学与公共管理学院", Account: "wangqiang", ManagedClasses: []string{"公信25", "公信26"}},
}

// 系统管理员：唯一最高权限身份，负责指定各辅导员的管辖班级，不受管辖班级限制。
var superAdminSpec = accountSpec{
	ID: "user-super", Role: domain.RoleAdmin, DisplayName: "系统管理员",
	ClassName: "学生工作办公室", Account: "superadmin", IsSuperAdmin: true,
}

func main() {
	cfg := config.Load()
	db, err := repository.Open(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}
	if err := seed(db); err != nil {
		log.Fatalf("seed accounts: %v", err)
	}
	log.Println("测试账号矩阵就绪：4 个班级 × (1 班委 + ≥2 学生) + 2 名辅导员（各管辖 2 个班级）+ 1 名系统管理员")
}

func seed(db *gorm.DB) error {
	hash, err := service.HashPassword("123456")
	if err != nil {
		return err
	}
	return db.Transaction(func(tx *gorm.DB) error {
		specs := append(append(append([]accountSpec{}, accountSpecs...), adminSpecs...), superAdminSpec)
		for _, spec := range specs {
			var user domain.User
			err := tx.First(&user, "account = ?", spec.Account).Error
			switch {
			case err == nil:
				// 已存在：校正角色与班级，不动密码
				updates := map[string]any{"role": spec.Role, "display_name": spec.DisplayName, "updated_at": tx.NowFunc()}
				if spec.ClassName != "" {
					updates["class_name"] = spec.ClassName
				}
				updates["is_super_admin"] = spec.IsSuperAdmin
				if err := tx.Model(&user).Updates(updates).Error; err != nil {
					return err
				}
			case service.IsNotFound(err):
				user = domain.User{
					ID: spec.ID, Role: spec.Role, DisplayName: spec.DisplayName,
					StudentID: spec.StudentID, ClassName: spec.ClassName,
					Account: spec.Account, PasswordHash: hash, IsSuperAdmin: spec.IsSuperAdmin,
				}
				if err := tx.Create(&user).Error; err != nil {
					return err
				}
			default:
				return err
			}
			// 管辖班级只对非超管的 admin 生效
			if spec.Role == domain.RoleAdmin && !spec.IsSuperAdmin && spec.ManagedClasses != nil {
				user.ManagedClasses = spec.ManagedClasses
				if err := tx.Model(&user).Select("managed_classes").Updates(&user).Error; err != nil {
					return err
				}
			}
		}
		// 保障活跃综测周期存在（与 seed 保持一致）
		var count int64
		if err := tx.Model(&domain.AssessmentCycle{}).Where("active = ?", true).Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&domain.AssessmentCycle{
				ID: "2026", Name: "2026 综测周期", Active: true,
			}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
