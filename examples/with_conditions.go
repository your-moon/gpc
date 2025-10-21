package examples

import "gorm.io/gorm"

// Examples showing Preload with conditions (args parameter)

type Post struct {
	Title   string
	Content string
}

type Comment struct {
	Text string
	Post Post
}

type Author struct {
	Name     string
	Posts    []Post
	Comments []Comment
}

// PreloadWithConditions demonstrates Preload with additional arguments
func PreloadWithConditions(db *gorm.DB) {
	var authors []Author

	// ✅ Preload with conditions - first arg is still validated
	db.Preload("Posts", "published = ?", true).Find(&authors)

	// ✅ Preload with function conditions
	db.Preload("Posts", func(db *gorm.DB) *gorm.DB {
		return db.Where("published = ?", true)
	}).Find(&authors)

	// ✅ Multiple conditions
	db.Preload("Posts", "published = ? AND views > ?", true, 100).Find(&authors)

	// ✅ Nested preload with conditions
	db.Preload("Comments.Post", "published = ?", true).Find(&authors)

	// ❌ Typo in relation name - still caught even with conditions
	// Error: invalid preload: Post not found in Author
	db.Preload("Post", "published = ?", true).Find(&authors)

	// ❌ Typo in nested relation - still caught
	// Error: invalid preload: Comments.Pos not found in Author
	db.Preload("Comments.Pos", "published = ?", true).Find(&authors)
}

// PreloadWithVariables shows cases that won't be validated
func PreloadWithVariables(db *gorm.DB) {
	var authors []Author

	// ⚠️ Variable relation names cannot be validated at compile time
	relationName := "Posts"
	db.Preload(relationName).Find(&authors)

	// ⚠️ Dynamic relation names - skipped by linter
	relations := []string{"Posts", "Comments"}
	for _, rel := range relations {
		db.Preload(rel).Find(&authors)
	}
}
