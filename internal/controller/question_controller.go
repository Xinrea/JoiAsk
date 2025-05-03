package controller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"joiask-backend/internal/database"
	"joiask-backend/internal/storage"
	"joiask-backend/pkg/util"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sse"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm/clause"
)

type EmojiRecords struct {
	CardID int            `json:"card_id"`
	Emojis []*EmojiRecord `json:"emojis"`
}

type QuestionController struct {
	emojiChan    chan EmojiRecords
	clients      map[chan EmojiRecords]bool
	clientsMutex sync.Mutex
}

func NewQuestionController() *QuestionController {
	controller := &QuestionController{
		emojiChan: make(chan EmojiRecords),
		clients:   make(map[chan EmojiRecords]bool),
	}
	// Start broadcast goroutine
	go controller.broadcast()
	return controller
}

func (this *QuestionController) broadcast() {
	for emoji := range this.emojiChan {
		// Broadcast to all clients
		this.clientsMutex.Lock()
		for client := range this.clients {
			select {
			case client <- emoji:
			default:
				// Remove client if channel is full
				delete(this.clients, client)
				close(client)
			}
		}
		this.clientsMutex.Unlock()
	}
}

type QuestionRequest struct {
	OrderBy  string `form:"order_by"`
	Order    string `form:"order"`
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
	TagID    int    `form:"tag_id"`
	Search   string `form:"search"`
	Hide     bool   `form:"hide"`
	Rainbow  bool   `form:"rainbow"`
	Archive  bool   `form:"archive"`
	Publish  bool   `form:"publish"`
}

type QuestionModifyRequest struct {
	TagID     int  `json:"tag_id"`
	IsHide    bool `json:"is_hide"`
	IsRainbow bool `json:"is_rainbow"`
	IsArchive bool `json:"is_archive"`
	IsPublish bool `json:"is_publish"`
}

// Get questions
func (*QuestionController) Get(c *gin.Context) {
	// Prepare for auth
	if sessions.Default(c).Get("authed") == true {
		c.Set("authed", true)
	}
	var questionList []database.Question
	var total int64
	var request QuestionRequest
	if err := c.Bind(&request); err != nil {
		Fail(c, 400, "请求错误")
		return
	}
	var tx = database.DB.Model(&database.Question{}).Preload(clause.Associations).Order(getOrderBy(request.OrderBy) + " " + getOrder(request.Order))
	if getOrderBy(request.OrderBy) != "is_archive" {
		tx = tx.Order("is_archive asc")
	}
	if _, ok := c.GetQuery("tag_id"); ok {
		if request.TagID > 0 {
			tx = tx.Where("tag_id = ?", request.TagID)
		}
	}
	if _, ok := c.GetQuery("search"); ok {
		tx = tx.Where("content like ?", "%"+request.Search+"%")
	}
	if _, ok := c.GetQuery("hide"); ok {
		tx = tx.Where("is_hide = ?", request.Hide)
	}
	if _, ok := c.GetQuery("rainbow"); ok {
		tx = tx.Where("is_rainbow = ?", request.Rainbow)
	}
	if _, ok := c.GetQuery("archive"); ok {
		tx = tx.Where("is_archive = ?", request.Archive)
	}
	if _, ok := c.GetQuery("publish"); ok {
		// Normal users can only see published questions
		log.Debug("publish: ", request.Publish)
		if !c.GetBool("authed") {
			tx = tx.Where("is_publish = ?", true)
		} else {
			tx = tx.Where("is_publish = ?", request.Publish)
		}
	} else {
		// Normal users can only see published questions
		if !c.GetBool("authed") {
			tx = tx.Where("is_publish = ?", true)
		}
	}
	err := tx.Count(&total).Scopes(paginate(getPage(request.Page), getPageSize(request.PageSize))).Find(&questionList).Error
	if err != nil {
		log.Error(err)
		Fail(c, 500, "获取提问失败")
		return
	}
	Success(c, gin.H{
		"questions": questionList,
		"total":     total,
		"page":      getPage(request.Page),
		"page_size": getPageSize(request.PageSize),
	})
}

func (*QuestionController) Put(c *gin.Context) {
	var request QuestionModifyRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		Fail(c, 400, "请求错误")
		return
	}
	var q database.Question
	database.DB.First(&q, c.Param("id"))
	if q.ID == 0 {
		Fail(c, 404, "提问不存在")
		return
	}
	if request.TagID == 0 {
		Fail(c, 400, "请求错误")
		return
	}
	q.TagID = request.TagID
	q.IsHide = request.IsHide
	q.IsRainbow = request.IsRainbow
	q.IsArchive = request.IsArchive
	q.IsPublish = request.IsPublish
	err := database.DB.Save(&q).Error
	if err != nil {
		log.Error(err)
		Fail(c, 500, "修改提问失败")
		return
	}
	Success(c, nil)
}

func (*QuestionController) Post(c *gin.Context) {
	var tag database.Tag
	tagID := c.PostForm("tag_id")
	database.DB.First(&tag, tagID)
	if tag.ID == 0 {
		Fail(c, 404, "话题不存在")
		return
	}
	var q database.Question
	q.TagID = int(tag.ID)
	q.Content = strings.Trim(c.PostForm("content"), " \r\n\t")
	q.IsHide = c.PostForm("hide") == "true"
	q.IsRainbow = c.PostForm("rainbow") == "true"
	mp, _ := c.MultipartForm()
	for _, v := range mp.File["files[]"] {
		f, err := v.Open()
		if err != nil {
			log.Error(err)
			Fail(c, 405, "文件上传失败")
			return
		}
		fileContent, err := io.ReadAll(f)
		if err != nil {
			log.Error(err)
			Fail(c, 405, "文件上传失败")
			return
		}
		newFileName := util.Md5v(string(fileContent)) + path.Ext(v.Filename)
		url, err := storage.Get().Upload(newFileName, bytes.NewReader(fileContent))
		if err != nil {
			log.Error(err)
			Fail(c, 500, "文件上传失败")
			return
		}
		q.ImagesNum++
		if q.Images == "" {
			q.Images = url
		} else {
			q.Images += ";" + url
		}
	}
	err := database.DB.Save(&q).Error
	if err != nil {
		log.Error(err)
		Fail(c, 500, "创建提问失败")
		return
	}
	Success(c, nil)
}

func (*QuestionController) Delete(c *gin.Context) {
	var q database.Question
	id := c.Param("id")
	err := database.DB.First(&q, id).Error
	if q.ID == 0 {
		log.Error(err)
		Fail(c, 404, "提问不存在")
		return
	}
	tx := database.DB.Begin()
	tx.Delete(&database.LikeRecord{}, "question_id", q.ID)
	tx.Delete(&q)
	if tx.Error != nil {
		log.Error(err)
		Fail(c, 500, "删除提问失败")
		tx.Rollback()
		return
	}
	tx.Commit()
	// clean images in storage
	key := "upload-img/"
	images := strings.Split(q.Images, ";")
	filenames := []string{}
	for i := range images {
		cur := images[i]
		parts := strings.SplitN(cur, key, 2)
		if len(parts) < 2 {
			continue
		}
		filenames = append(filenames, parts[1])
	}
	// it's ok to fail deleting
	for _, f := range filenames {
		_ = storage.Get().Delete(f)
	}
	Success(c, nil)
}

type EmojiRecord struct {
	Value string `json:"value"`
	Count int    `json:"count"`
}

var EmojiValid = map[string]bool{
	"👍": true,
	"👎": true,
	"🤣": true,
	"😭": true,
	"😓": true,
	"😬": true,
	"🥳": true,
	"😨": true,
	"😠": true,
	"💩": true,
	"💖": true,
	"🐵": true,
	"❓": true,
	"🫂": true,
	"🔘": true,
	"👅": true,
	"🥺": true,
	"👻": true,
	"😅": true,
	"🌹": true,
}

func (this *QuestionController) Emoji(c *gin.Context) {
	emojiToAdd := c.PostForm("emoji")
	// should check emojiToAdd valid or not
	if !EmojiValid[emojiToAdd] {
		Fail(c, 400, "无效的表情符号")
		return
	}
	// ip := c.ClientIP()
	id, _ := strconv.Atoi(c.Param("id"))
	// var lr database.LikeRecord
	// database.DB.Where("ip = ? and question_id = ?", ip, id).First(&lr)
	// if lr.ID > 0 {
	// 	Fail(c, 400, "您已经评价过了")
	// 	return
	// }
	// lr.IP = ip
	// lr.QuestionID = id
	// tx := database.DB.Begin()
	// err := tx.Create(&lr).Error
	// if err != nil {
	// 	log.Error(err)
	// 	Fail(c, 500, "评价失败")
	// 	tx.Rollback()
	// 	return
	// }
	tx := database.DB.Begin()
	var q database.Question
	err := database.DB.Where("id = ?", id).First(&q).Error
	if err != nil {
		log.Error(err)
		Fail(c, 500, "评价失败")
		tx.Rollback()
		return
	}
	var emojis []*EmojiRecord
	if len(q.Emojis) > 0 {
		err = json.Unmarshal([]byte(q.Emojis), &emojis)
		if err != nil {
			log.Error((err))
			Fail(c, 500, "解析表情失败")
			tx.Rollback()
			return
		}
	}
	added := false
	for _, e := range emojis {
		if e.Value == emojiToAdd {
			added = true
			e.Count++
			break
		}
	}
	if !added {
		emojis = append(emojis, &EmojiRecord{
			Value: emojiToAdd,
			Count: 1,
		})
	}
	updatedList, err := json.Marshal(emojis)
	if err != nil {
		log.Error(err)
		Fail(c, 500, "评价失败")
		tx.Rollback()
		return
	}
	q.Emojis = string(updatedList)
	err = tx.Save(&q).Error
	if err != nil {
		log.Error(err)
		Fail(c, 500, "评价失败")
		tx.Rollback()
		return
	}
	if tx.Commit().Error != nil {
		log.Error(err)
		Fail(c, 500, "评价失败")
		tx.Rollback()
		return
	}
	this.emojiChan <- EmojiRecords{
		CardID: id,
		Emojis: emojis,
	}
	Success(c, nil)
}

func (this *QuestionController) SSE(c *gin.Context) {
	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("X-Accel-Buffering", "no") // Disable buffering for Nginx

	// Create a channel for this client
	clientChan := make(chan EmojiRecords, 1024)

	// Add client to the connection manager
	this.clientsMutex.Lock()
	this.clients[clientChan] = true
	this.clientsMutex.Unlock()

	// Create a channel to handle client disconnection
	clientGone := c.Writer.CloseNotify()

	// Create a ticker for heartbeat with initial immediate tick
	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()

	// Create a message ID counter
	messageID := 0

	// Start streaming
	c.Stream(func(w io.Writer) bool {
		// Send initial connection message on first call
		if messageID == 0 {
			err := sse.Encode(w, sse.Event{
				Event: "connected",
				Data:  "connected",
			})
			if err != nil {
				log.Error("Failed to encode connected event:", err)
				return false
			}
			messageID++
			return true
		}

		select {
		case <-clientGone:
			log.Info("Client disconnected")
			// Remove client from connection manager
			this.clientsMutex.Lock()
			delete(this.clients, clientChan)
			this.clientsMutex.Unlock()
			close(clientChan)
			return false
		case <-heartbeat.C:
			// Send heartbeat comment
			err := sse.Encode(w, sse.Event{
				Event: "heartbeat",
				Data:  "heartbeat",
			})
			if err != nil {
				log.Error("Failed to encode heartbeat event:", err)
				return false
			}
			return true
		case emojis := <-clientChan:
			emojiJson, err := json.Marshal(emojis)
			if err != nil {
				log.Error("Failed to marshal emoji data:", err)
				return false
			}

			// Send emoji update
			err = sse.Encode(w, sse.Event{
				Id:    fmt.Sprintf("%d", messageID),
				Event: "emoji",
				Data:  string(emojiJson),
			})
			if err != nil {
				log.Error("Failed to encode emoji event:", err)
				return false
			}
			messageID++
			return true
		case <-time.After(30 * time.Second):
			// Send retry message in case of timeout
			err := sse.Encode(w, sse.Event{
				Event: "retry",
				Retry: 10000,
			})
			if err != nil {
				log.Error("Failed to encode retry event:", err)
				return false
			}
			return true
		}
	})

	// Cleanup when connection is closed
	log.Info("SSE connection closed")
}
