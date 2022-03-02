package database

import (
	"gorm.io/gorm"
)

// Query

type ConfigJson struct {
	Announcement string `json:"announcement"`
}

func GetConfig() (ConfigJson, error) {
	var config ConfigJson
	var c Config
	err := db.First(&c).Error
	if err != nil {
		return ConfigJson{}, err
	}
	config.Announcement = c.Announcement
	return config, nil
}

func SetConfig(a string) {
	var config Config
	db.First(&config)
	config.Announcement = a
	db.Save(&config)
}

func GetUserPasswordMD5(username string) (string, error) {
	var user Admin
	err := db.Where("username = ?", username).First(&user).Error
	if err != nil {
		return "", err
	}
	return user.Password, nil
}

type QuestionJson struct {
	ID          uint   `json:"id"`
	TagName     string `json:"tag_name"`
	Content     string `json:"content"`
	ImagesNum   int    `json:"images_num"`
	Images      string `json:"images"`
	Likes       int    `json:"likes"`
	IsHide      bool   `json:"is_hide"`
	IsPublished bool   `json:"is_published"`
	IsRainbow   bool   `json:"is_rainbow"`
	IsArchived  bool   `json:"is_archived"`
	CreatedAt   string `json:"created_at"`
}

type TableConfig struct {
	Pagination struct {
		PageSize    int `json:"pageSize"`
		Page        int `json:"page"`
		TotalLength int `json:"total_length"`
	} `json:"pagination"`
	Order struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	} `json:"order"`
	Tag int `json:"tag"`
}

func AuthAllQuestions(c TableConfig) ([]QuestionJson, int64, error) {
	var questionList []QuestionJson
	var total int64
	var tx = db.Model(&Question{}).Order(c.Order.Key + " " + c.Order.Value).Order("id desc")
	if c.Tag > 0 {
		tx = tx.Where("tag_id = ?", c.Tag)
	}
	tx.Count(&total)
	err := tx.Joins("left join tags on tags.id = questions.tag_id").
		Select("questions.id as id, tag_name, content, images_num, images, likes, is_hide, is_published, is_rainbow, is_archived, questions.created_at").
		Scopes(paginate(c.Pagination.Page, c.Pagination.PageSize)).Find(&questionList).Error
	if err != nil {
		return nil, 0, err
	}
	return questionList, total, nil
}

func GetAllQuestions(isPublic bool, page int, size int) ([]QuestionJson, error) {
	var questionList []QuestionJson
	err := db.Model(&Question{}).Scopes(public(isPublic)).
		Joins("left join tags on tags.id = questions.tag_id").
		Select("questions.id as id, tag_name, content, images_num, images, likes, is_hide, is_published, is_rainbow, is_archived, questions.created_at").
		Scopes(differArchive(), paginate(page, size)).Find(&questionList).Error
	if err != nil {
		return nil, err
	}
	return questionList, nil
}

func GetAllQuestionsByTag(isPublic bool, tagID, page int, size int) ([]QuestionJson, error) {
	var questionList []QuestionJson
	err := db.Model(&Question{}).Where("tag_id = ?", tagID).Scopes(public(isPublic)).
		Joins("left join tags on tags.id = questions.tag_id").
		Select("questions.id, tag_name, content, images_num, images, likes, is_hide, is_published, is_rainbow, is_archived, questions.created_at").
		Scopes(differArchive(), paginate(page, size)).Find(&questionList).Error
	if err != nil {
		return nil, err
	}
	return questionList, nil
}

func GetAllRainbowQuestions(isPublic bool, page int, size int) ([]QuestionJson, error) {
	var questions []QuestionJson
	err := db.Model(&Question{}).Scopes(rainbow(), public(isPublic)).
		Joins("left join tags on tags.id = questions.tag_id").
		Select("questions.id as id, tag_name, content, images_num, images, likes, is_published, is_rainbow, is_archived, questions.created_at").
		Scopes(differArchive(), paginate(page, size)).Find(&questions).Error
	if err != nil {
		return nil, err
	}
	return questions, nil
}

func DeleteQuestion(id uint) error {
	var question Question
	err := db.Where("id = ?", id).First(&question).Error
	if err != nil {
		return err
	}
	db.Delete(&question)
	return nil
}

type TagJson struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

type TagDetailJson struct {
	TagID         uint   `json:"tag_id"`
	TagName       string `json:"tag_name"`
	QuestionCount int    `json:"question_count"`
}

func GetTag(tagID uint) (TagJson, error) {
	var tag Tag
	err := db.Where("id = ?", tagID).First(&tag).Error
	if err != nil {
		return TagJson{}, err
	}
	return TagJson{
		ID:   tag.ID,
		Name: tag.TagName,
	}, nil
}

func GetTagList(isPublic bool) ([]TagJson, error) {
	var tags []Tag
	err := db.Find(&tags).Error
	if err != nil {
		return nil, err
	}
	var tagList []TagJson
	for _, tag := range tags {
		tagList = append(tagList, TagJson{ID: tag.ID, Name: tag.TagName})
	}
	return tagList, nil
}

func GetAllTags(isPublic bool) ([]TagDetailJson, error) {
	var tags []TagDetailJson
	err := db.Model(&Question{}).Scopes(public(isPublic)).Joins("left join tags on tags.id = questions.tag_id").Select("tag_id,tag_name, count(*) as question_count").Group("tag_id").Scan(&tags).Error
	if err != nil {
		return nil, err
	}
	return tags, nil
}

// Insert

func AddNewTag(tagName string) (uint, error) {
	var tag Tag
	err := db.Where("tag_name = ?", tagName).First(&tag).Error
	if err == nil {
		return tag.ID, nil
	}
	tag = Tag{TagName: tagName}
	err = db.Create(&tag).Error
	if err != nil {
		return 0, err
	}
	return tag.ID, nil
}

func AddNewQuestion(q Question) error {
	err := db.Create(&q).Error
	if err != nil {
		return err
	}
	return nil
}

func UpdateQuestion(id uint, row string, value interface{}) error {
	err := db.Model(&Question{}).Where("id = ?", id).Update(row, value).Error
	if err != nil {
		return err
	}
	return nil
}

func AddLikeRecord(ip string, qid uint) error {
	var lr LikeRecord
	err := db.Where("ip = ? and question_id = ?", ip, qid).First(&lr).Error
	if err == nil {
		return nil
	}
	lr = LikeRecord{IP: ip, QuestionID: int(qid)}
	err = db.Create(&lr).Error
	if err != nil {
		return err
	}
	// update likes
	var q Question
	err = db.Where("id = ?", qid).First(&q).Error
	if err != nil {
		return err
	}
	q.Likes++
	err = db.Save(&q).Error
	if err != nil {
		return err
	}
	return nil
}

// Delete

func paginate(page int, size int) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if page == 0 {
			page = 1
		}
		switch {
		case size > 100:
			size = 100
		case size <= 0:
			size = 10
		}

		offset := (page - 1) * size
		return db.Offset(offset).Limit(size)
	}
}

func differArchive() func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Order("is_archived asc").Order("created_at desc")
	}
}

func public(enable bool) func(db *gorm.DB) *gorm.DB {
	if enable {
		return func(db *gorm.DB) *gorm.DB {
			return db.Where("is_published = ?", true)
		}
	}
	return func(db *gorm.DB) *gorm.DB {
		return db
	}
}

func rainbow() func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("is_rainbow = ?", true)
	}
}
