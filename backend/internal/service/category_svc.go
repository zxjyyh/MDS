package service

import (
	"errors"
	"github.com/zxjyyh/mds/internal/models"

	"github.com/jmoiron/sqlx"
)

type CategoryService struct{ DB *sqlx.DB }

func (s *CategoryService) GetTree(systemType string) ([]models.DocCategory, error) {
	var all []models.DocCategory
	query := "SELECT * FROM doc_categories ORDER BY sort_order ASC, id ASC"
	args := []interface{}{}
	if systemType != "" {
		query = "SELECT * FROM doc_categories WHERE system_type=? ORDER BY sort_order ASC, id ASC"
		args = append(args, systemType)
	}
	err := s.DB.Select(&all, query, args...)
	if err != nil {
		return nil, err
	}

	// Group children by parent ID
	childrenMap := make(map[int][]models.DocCategory)
	var roots []models.DocCategory

	for i := range all {
		cat := all[i]
		cat.Children = []models.DocCategory{}
		if cat.ParentID != nil {
			childrenMap[*cat.ParentID] = append(childrenMap[*cat.ParentID], cat)
		} else {
			roots = append(roots, cat)
		}
	}

	// Recursively assign children (works for unlimited depth)
	var assignChildren func(cat *models.DocCategory)
	assignChildren = func(cat *models.DocCategory) {
		cat.Children = childrenMap[cat.ID]
		if cat.Children == nil {
			cat.Children = []models.DocCategory{}
		}
		for i := range cat.Children {
			assignChildren(&cat.Children[i])
		}
	}

	for i := range roots {
		assignChildren(&roots[i])
	}

	return roots, nil
}

func (s *CategoryService) Create(name string, parentID *int, systemType string, sortOrder int) (*models.DocCategory, error) {
	// Validate parent exists
	if parentID != nil {
		var count int
		s.DB.Get(&count, "SELECT COUNT(*) FROM doc_categories WHERE id=?", *parentID)
		if count == 0 {
			return nil, errors.New("父分类不存在")
		}
	}

	// Check name uniqueness within same parent+systemType
	var count int
	if parentID != nil {
		s.DB.Get(&count, "SELECT COUNT(*) FROM doc_categories WHERE name=? AND system_type=? AND parent_id=?", name, systemType, *parentID)
	} else {
		s.DB.Get(&count, "SELECT COUNT(*) FROM doc_categories WHERE name=? AND system_type=? AND parent_id IS NULL", name, systemType)
	}
	if count > 0 {
		return nil, errors.New("同级分类名称已存在")
	}

	res, err := s.DB.Exec("INSERT INTO doc_categories (name, parent_id, system_type, sort_order) VALUES (?,?,?,?)",
		name, parentID, systemType, sortOrder)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()

	var cat models.DocCategory
	s.DB.Get(&cat, "SELECT * FROM doc_categories WHERE id=?", id)
	return &cat, nil
}

func (s *CategoryService) Update(id int, name string, parentID *int, sortOrder int) error {
	// Check category exists
	var cat models.DocCategory
	if err := s.DB.Get(&cat, "SELECT * FROM doc_categories WHERE id=?", id); err != nil {
		return errors.New("分类不存在")
	}

	// Prevent circular reference
	if parentID != nil {
		if *parentID == id {
			return errors.New("不能将分类设置为自己的子分类")
		}
		// Check if target parent is a descendant
		if s.isDescendant(*parentID, id) {
			return errors.New("不能将父分类设置为子分类的子分类")
		}
	}

	// Check name uniqueness
	var count int
	if parentID != nil {
		s.DB.Get(&count, "SELECT COUNT(*) FROM doc_categories WHERE name=? AND system_type=? AND parent_id=? AND id!=?",
			name, cat.SystemType, *parentID, id)
	} else {
		s.DB.Get(&count, "SELECT COUNT(*) FROM doc_categories WHERE name=? AND system_type=? AND parent_id IS NULL AND id!=?",
			name, cat.SystemType, id)
	}
	if count > 0 {
		return errors.New("同级分类名称已存在")
	}

	_, err := s.DB.Exec("UPDATE doc_categories SET name=?, parent_id=?, sort_order=? WHERE id=?",
		name, parentID, sortOrder, id)
	return err
}

func (s *CategoryService) Delete(id int) error {
	// Check if has children
	var childCount int
	s.DB.Get(&childCount, "SELECT COUNT(*) FROM doc_categories WHERE parent_id=?", id)
	if childCount > 0 {
		return errors.New("该分类下有子分类，无法删除")
	}

	// Check if has documents
	var docCount int
	s.DB.Get(&docCount, "SELECT COUNT(*) FROM documents WHERE category_id=?", id)
	if docCount > 0 {
		return errors.New("该分类下有文档，无法删除")
	}

	_, err := s.DB.Exec("DELETE FROM doc_categories WHERE id=?", id)
	return err
}

func (s *CategoryService) isDescendant(ancestorID, targetID int) bool {
	rows, err := s.DB.Query(`
		WITH RECURSIVE descendants AS (
			SELECT id, parent_id FROM doc_categories WHERE parent_id=?
			UNION ALL
			SELECT c.id, c.parent_id FROM doc_categories c
			INNER JOIN descendants d ON c.parent_id = d.id
		)
		SELECT 1 FROM descendants WHERE id=?`, ancestorID, targetID)
	if err != nil {
		return false
	}
	defer rows.Close()
	return rows.Next()
}

// GetByID returns a flat list for a given system type (used for document filter dropdowns)
func (s *CategoryService) GetByID(id int) (*models.DocCategory, error) {
	var cat models.DocCategory
	err := s.DB.Get(&cat, "SELECT * FROM doc_categories WHERE id=?", id)
	return &cat, err
}

// helper alias
func (s *CategoryService) GetFlatList(systemType string) []models.DocCategory {
	var list []models.DocCategory
	s.DB.Select(&list, "SELECT * FROM doc_categories WHERE system_type=? ORDER BY sort_order ASC, id ASC", systemType)
	return list
}
