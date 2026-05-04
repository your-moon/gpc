package examples

import "gorm.io/gorm"

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

func PreloadWithConditions(db *gorm.DB) {
	var authors []Author

	db.Preload("Posts", "published = ?", true).Find(&authors)
	db.Preload("Posts", func(db *gorm.DB) *gorm.DB {
		return db.Where("published = ?", true)
	}).Find(&authors)
	db.Preload("Posts", "published = ? AND views > ?", true, 100).Find(&authors)
	db.Preload("Comments.Post", "published = ?", true).Find(&authors)

	db.Preload("Post", "published = ?", true).Find(&authors)         // typo: Post → Posts
	db.Preload("Comments.Pos", "published = ?", true).Find(&authors) // typo in nested path
}

func PreloadWithVariables(db *gorm.DB) {
	var authors []Author

	relationName := "Posts"
	db.Preload(relationName).Find(&authors) // skipped: dynamic argument

	relations := []string{"Posts", "Comments"}
	for _, rel := range relations {
		db.Preload(rel).Find(&authors) // skipped: dynamic argument
	}
}
