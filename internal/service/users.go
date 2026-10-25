package service

import (
	"encoding/csv"
	"fmt"
	"io"
	"strings"

	"demo1/internal/domain"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ListUsers 返回用户列表（管理员功能）。
// 系统管理员可见全部用户；辅导员只能看到管辖班级内的学生/班委账号 + 自己；未配置管辖班级则只能看到自己。
func (s *Service) ListUsers(actor domain.User, limit, offset int) ([]domain.UserListItem, error) {
	var users []domain.User
	query := s.db.Order("created_at DESC")
	if !IsSuperAdmin(actor) {
		if len(actor.ManagedClasses) > 0 {
			query = query.Where("class_name IN ? OR id = ?", actor.ManagedClasses, actor.ID)
		} else {
			query = query.Where("id = ?", actor.ID)
		}
	}
	if limit > 0 {
		query = query.Limit(limit).Offset(offset)
	}
	if err := query.Find(&users).Error; err != nil {
		return nil, err
	}
	items := make([]domain.UserListItem, 0, len(users))
	for _, u := range users {
		items = append(items, domain.UserListItem{
			ID:             u.ID,
			Role:           u.Role,
			DisplayName:    u.DisplayName,
			StudentID:      u.StudentID,
			ClassName:      u.ClassName,
			Account:        u.Account,
			ManagedClasses: u.ManagedClasses,
			IsSuperAdmin:   u.IsSuperAdmin,
			CreatedAt:      u.CreatedAt,
		})
	}
	return items, nil
}

// ImportUsersCSV 解析 CSV 批量创建学生账号
// CSV 格式：学号,姓名,班级（第一行为标题行）
func (s *Service) ImportUsersCSV(actor domain.User, reader io.Reader) domain.UserImportResult {
	r := csv.NewReader(reader)
	records, err := r.ReadAll()
	if err != nil {
		return domain.UserImportResult{Errors: []string{"无法解析 CSV 文件: " + err.Error()}}
	}
	if len(records) < 2 {
		return domain.UserImportResult{Errors: []string{"CSV 文件为空或缺少数据行"}}
	}

	// 定位列索引
	header := records[0]
	idxStudentID := -1
	idxName := -1
	idxClass := -1
	for i, col := range header {
		switch strings.TrimSpace(col) {
		case "学号":
			idxStudentID = i
		case "姓名":
			idxName = i
		case "班级":
			idxClass = i
		}
	}
	if idxStudentID == -1 {
		return domain.UserImportResult{Errors: []string{"CSV 缺少「学号」列"}}
	}
	if idxName == -1 {
		return domain.UserImportResult{Errors: []string{"CSV 缺少「姓名」列"}}
	}

	const defaultPassword = "123456"
	hash, err := HashPassword(defaultPassword)
	if err != nil {
		return domain.UserImportResult{Errors: []string{"密码加密失败: " + err.Error()}}
	}

	result := domain.UserImportResult{}
	for i := 1; i < len(records); i++ {
		row := records[i]
		studentID := strings.TrimSpace(colAt(row, idxStudentID))
		name := strings.TrimSpace(colAt(row, idxName))
		className := strings.TrimSpace(colAt(row, idxClass))

		if studentID == "" || name == "" {
			result.Errors = append(result.Errors, fmt.Sprintf("第 %d 行数据不完整，已跳过", i+1))
			continue
		}

		user := domain.User{
			ID:           NewID("u"),
			Role:         domain.RoleStudent,
			DisplayName:  name,
			StudentID:    studentID,
			ClassName:    className,
			Account:      studentID,
			PasswordHash: hash,
		}

		// OnConflict DoNothing：已存在的学号跳过
		res := s.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&user)
		if res.Error != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s(%s) 创建失败: %s", name, studentID, res.Error.Error()))
			continue
		}
		if res.RowsAffected == 0 {
			result.Skipped++
		} else {
			result.Created++
		}
	}

	// 记录审计日志
	_ = s.db.Transaction(func(tx *gorm.DB) error {
		return s.AddLog(tx, actor, "批量导入学生", fmt.Sprintf("新建 %d 人，跳过 %d 人", result.Created, result.Skipped), "")
	})

	return result
}

// ChangePassword 修改当前用户密码。
// currentToken 为触发改密请求的当前会话令牌：改密后会吊销该用户的其他有效会话，
// 仅保留当前会话，避免旧 token 在密码泄露后仍可用（纵深防御）。
func (s *Service) ChangePassword(user domain.User, oldPassword, newPassword, currentToken string) error {
	oldPassword = Trim(oldPassword)
	newPassword = Trim(newPassword)
	if oldPassword == "" || newPassword == "" {
		return ErrWrongPassword
	}
	if len(newPassword) < 6 {
		return fmt.Errorf("新密码长度至少 6 位")
	}

	// 验证旧密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(oldPassword)); err != nil {
		return ErrWrongPassword
	}

	hash, err := HashPassword(newPassword)
	if err != nil {
		return err
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&domain.User{}).Where("id = ?", user.ID).
			Update("password_hash", hash).Error; err != nil {
			return err
		}
		// 吊销该用户的其他有效会话（保留当前会话），避免旧 token 仍可用。
		now := s.now()
		revoke := tx.Model(&domain.Session{}).
			Where("user_id = ? AND revoked_at IS NULL", user.ID)
		if currentToken != "" {
			revoke = revoke.Where("token_hash <> ?", s.TokenHash(currentToken))
		}
		if err := revoke.Update("revoked_at", now).Error; err != nil {
			return err
		}
		return s.AddLog(tx, user, "修改密码", roleLabel(user.Role), "已吊销其他登录会话")
	})
}

// UpdateUserRole 修改用户角色。
// 辅导员：只能把管辖班级内的账号在 student ↔ reviewer 间调整；
// 系统管理员：还可以任命/撤销辅导员（admin），但系统管理员本身的角色不可改动。
func (s *Service) UpdateUserRole(actor domain.User, targetID, newRole string) error {
	newRole = Trim(newRole)
	if newRole != domain.RoleStudent && newRole != domain.RoleReviewer && newRole != domain.RoleAdmin {
		return fmt.Errorf("只能设置为 student、reviewer 或 admin 角色")
	}

	var target domain.User
	if err := s.db.First(&target, "id = ?", targetID).Error; err != nil {
		return err
	}
	if target.IsSuperAdmin {
		return fmt.Errorf("系统管理员的角色不可修改")
	}
	if !IsSuperAdmin(actor) {
		if newRole == domain.RoleAdmin {
			return ErrForbidden
		}
		// 辅导员只能管理管辖班级内的账号
		if !classesContain(actor.ManagedClasses, target.ClassName) {
			return ErrForbidden
		}
	}
	if target.Role == newRole {
		return nil // 无需更改
	}

	oldRole := target.Role
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&target).Update("role", newRole).Error; err != nil {
			return err
		}
		return s.AddLog(tx, actor, "修改角色",
			target.DisplayName,
			fmt.Sprintf("从 %s 改为 %s", roleLabel(oldRole), roleLabel(newRole)),
		)
	})
}

func colAt(row []string, index int) string {
	if index >= 0 && index < len(row) {
		return row[index]
	}
	return ""
}

// UpdateUserClasses 指定辅导员的管辖班级。仅系统管理员可操作，辅导员无权自改。
func (s *Service) UpdateUserClasses(actor domain.User, targetID string, classes []string) (domain.User, error) {
	if !IsSuperAdmin(actor) {
		return domain.User{}, ErrForbidden
	}
	cleaned := make([]string, 0, len(classes))
	seen := map[string]bool{}
	for _, item := range classes {
		item = Trim(item)
		if item == "" || seen[item] {
			continue
		}
		seen[item] = true
		cleaned = append(cleaned, item)
	}
	var target domain.User
	if err := s.db.First(&target, "id = ?", targetID).Error; err != nil {
		return domain.User{}, err
	}
	if target.Role != domain.RoleAdmin {
		return domain.User{}, fmt.Errorf("只能设置辅导员账号的管辖班级")
	}
	if target.IsSuperAdmin {
		return domain.User{}, fmt.Errorf("系统管理员不受管辖班级限制")
	}
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		target.ManagedClasses = cleaned
		if err := tx.Model(&target).Select("managed_classes").Updates(&target).Error; err != nil {
			return err
		}
		return s.AddLog(tx, actor, "设置管辖班级", target.DisplayName, strings.Join(cleaned, "、"))
	}); err != nil {
		return domain.User{}, err
	}
	return target, nil
}
