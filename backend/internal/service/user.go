package service

import (
	"context"
	"encoding/json"
	"fmt"
	"go-task-api/internal/cache"
	"go-task-api/internal/common"
	"go-task-api/internal/model"
	"go-task-api/internal/repository"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// UserService 负责承接用户相关业务逻辑。
type UserService struct {
	repo repository.UserRepository
}

// CreateUserRequest 定义用户注册请求参数。
type CreateUserRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8,max=32"`
}

type LoginResponse struct {
	Token string      `json:"token"`
	User  *model.User `json:"user"`
}

// NewUserService 创建用户服务实例。
func NewUserService(repo repository.UserRepository) *UserService {
	return &UserService{repo: repo}
}

// Register 校验参数、加密密码并创建用户。
func (s *UserService) Register(req *CreateUserRequest, ctx context.Context) (*model.User, error) {
	existingUser, err := s.repo.FindByEmail(req.Email, ctx)
	if err == nil && existingUser != nil {
		return nil, common.BadRequest("该邮箱已注册")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, common.InternalError(fmt.Sprintf("密码加密失败: %v", err))
	}

	user := model.User{
		Email:    req.Email,
		Password: string(hashedPassword),
	}

	if err := s.repo.Create(&user, ctx); err != nil {
		return nil, common.InternalError(fmt.Sprintf("用户注册失败: %v", err))
	}

	return &user, nil
}

// FindByEmail 按邮箱查询用户。
func (s *UserService) FindByEmail(email string, ctx context.Context) (*model.User, error) {
	user, err := s.repo.FindByEmail(email, ctx)
	if err != nil {
		return nil, common.InternalError(fmt.Sprintf("邮箱查询失败: %v", err))
	}
	return user, nil
}

// login 用户登录方法
func (s *UserService) Login(req *CreateUserRequest, ctx context.Context) (*LoginResponse, error) {
	user, err := s.repo.FindByEmail(req.Email, ctx)
	if err != nil {
		return nil, common.Unauthorized("邮箱或密码错误")
	}
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password))

	if err != nil {
		return nil, common.Unauthorized("邮箱或密码错误")
	}

	var loginRes LoginResponse

	loginRes.Token, err = common.GenerateToken(user.ID)
	loginRes.User = user

	return &loginRes, err
}

func (s *UserService) GetByID(id uint64, ctx context.Context) (*model.User, error) {
	key := fmt.Sprintf("user:%d", id)

	// cache aside 先查缓存
	val, err := cache.Redis.Get(ctx, key).Result()
	if err == nil {
		var user model.User

		if err := json.Unmarshal([]byte(val), &user); err == nil {
			return &user, nil
		}

		// 缓存数据损坏，删除后继续查DB
		_ = cache.Redis.Del(ctx, key).Err()
	}

	// cache miss  / redis 异常 -> 查询 DB
	user, err := s.repo.GetByID(id, ctx)

	if err != nil {
		return nil, common.NotFound("用户不存在")
	}

	// DB 查询成功 -> 回填缓存
	if data, err := json.Marshal(user); err == nil {
		_ = cache.Redis.Set(ctx, key, data, 10*time.Minute).Err()
	}
	return user, nil
}

// 上传文件
func (s *UserService) UploadAvatar(fileHeader *multipart.FileHeader, userID uint64, ctx context.Context) (string, error) {
	// 判断文件大小
	fileSize := fileHeader.Size
	if fileSize > 2*1024*1024 {
		return "", common.BadRequest("文件超出大小")
	}

	// 判断文件类型
	fileType := fileHeader.Header.Get("Content-Type")
	if fileType != "image/png" && fileType != "image/jpeg" && fileType != "image/jpg" {
		return "", common.BadRequest("不支持上传该类型文件")
	}

	// 第3步：拿扩展名，拼文件名
	ext := filepath.Ext(fileHeader.Filename) // 比如用户上传 "照片.png" → ext = ".png"
	filename := fmt.Sprintf("%d_%d%s", userID, time.Now().Unix(), ext)
	// userID=42, 时间戳=1693456789, ext=".png" → filename = "42_1693456789.png"

	// 第4步：拼完整保存路径
	dst := filepath.Join("uploads", "avatars", filename)
	// → "uploads/avatars/42_1693456789.png"

	os.MkdirAll("uploads/avatars", 0755)

	// 打开上传的文件（拆开信封，拿到内容）
	src, err := fileHeader.Open()
	if err != nil {
		return "", common.InternalError(fmt.Sprintf("打开上传文件失败: %v", err))
	}
	defer src.Close()

	// 在磁盘上创建目标文件（创建一个空文件等着写入）
	out, err := os.Create(dst)
	if err != nil {
		return "", common.InternalError(fmt.Sprintf("创建文件失败: %v", err))
	}
	defer out.Close()

	// 把内容从 src 复制到 out
	_, err = io.Copy(out, src)
	if err != nil {
		return "", common.InternalError(fmt.Sprintf("保存文件失败: %v", err))
	}

	if err := s.repo.UpdateAvatar(userID, dst, ctx); err != nil {
		return "", common.InternalError(fmt.Sprintf("更新头像失败: %v", err))
	}

	return dst, nil
}
