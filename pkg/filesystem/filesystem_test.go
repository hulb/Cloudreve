package filesystem

import (
	"github.com/DATA-DOG/go-sqlmock"
	model "github.com/HFO4/cloudreve/models"
	"github.com/HFO4/cloudreve/pkg/filesystem/local"
	"github.com/HFO4/cloudreve/pkg/filesystem/remote"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"net/http/httptest"

	"testing"
)

func TestNewFileSystem(t *testing.T) {
	asserts := assert.New(t)
	user := model.User{
		Policy: model.Policy{
			Type: "local",
		},
	}

	// 本地 成功
	fs, err := NewFileSystem(&user)
	asserts.NoError(err)
	asserts.NotNil(fs.Handler)
	asserts.IsType(local.Handler{}, fs.Handler)
	// 远程
	user.Policy.Type = "remote"
	fs, err = NewFileSystem(&user)
	asserts.NoError(err)
	asserts.NotNil(fs.Handler)
	asserts.IsType(remote.Handler{}, fs.Handler)

	user.Policy.Type = "unknown"
	fs, err = NewFileSystem(&user)
	asserts.Error(err)
}

func TestNewFileSystemFromContext(t *testing.T) {
	asserts := assert.New(t)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("user", &model.User{
		Policy: model.Policy{
			Type: "local",
		},
	})
	fs, err := NewFileSystemFromContext(c)
	asserts.NotNil(fs)
	asserts.NoError(err)

	c, _ = gin.CreateTestContext(httptest.NewRecorder())
	fs, err = NewFileSystemFromContext(c)
	asserts.Nil(fs)
	asserts.Error(err)
}

func TestDispatchHandler(t *testing.T) {
	asserts := assert.New(t)
	fs := &FileSystem{
		User: &model.User{Policy: model.Policy{
			Type: "local",
		}},
	}

	// 未指定，使用用户默认
	err := fs.dispatchHandler()
	asserts.NoError(err)
	asserts.IsType(local.Handler{}, fs.Handler)

	// 已指定，发生错误
	fs.Policy = &model.Policy{Type: "unknown"}
	err = fs.dispatchHandler()
	asserts.Error(err)
}

func TestFileSystem_SetTargetFileByIDs(t *testing.T) {
	asserts := assert.New(t)

	// 成功
	{
		fs := &FileSystem{}
		mock.ExpectQuery("SELECT(.+)").
			WithArgs(1, 2).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(1, "1.txt"))
		err := fs.SetTargetFileByIDs([]uint{1, 2})
		asserts.NoError(mock.ExpectationsWereMet())
		asserts.Len(fs.FileTarget, 1)
		asserts.NoError(err)
	}

	// 未找到
	{
		fs := &FileSystem{}
		mock.ExpectQuery("SELECT(.+)").WithArgs(1, 2).WillReturnRows(sqlmock.NewRows([]string{"id", "name"}))
		err := fs.SetTargetFileByIDs([]uint{1, 2})
		asserts.NoError(mock.ExpectationsWereMet())
		asserts.Len(fs.FileTarget, 0)
		asserts.Error(err)
	}
}

func TestFileSystem_CleanTargets(t *testing.T) {
	asserts := assert.New(t)
	fs := &FileSystem{
		FileTarget: []model.File{{}, {}},
		DirTarget:  []model.Folder{{}, {}},
	}

	fs.CleanTargets()
	asserts.Len(fs.FileTarget, 0)
	asserts.Len(fs.DirTarget, 0)
}

func TestNewAnonymousFileSystem(t *testing.T) {
	asserts := assert.New(t)

	// 正常
	{
		mock.ExpectQuery("SELECT(.+)").WillReturnRows(sqlmock.NewRows([]string{"id", "name", "policies"}).AddRow(3, "游客", "[]"))
		fs, err := NewAnonymousFileSystem()
		asserts.NoError(mock.ExpectationsWereMet())
		asserts.NoError(err)
		asserts.Equal("游客", fs.User.Group.Name)
	}

	// 游客用户组不存在
	{
		mock.ExpectQuery("SELECT(.+)").WillReturnRows(sqlmock.NewRows([]string{"id", "name", "policies"}))
		fs, err := NewAnonymousFileSystem()
		asserts.NoError(mock.ExpectationsWereMet())
		asserts.Error(err)
		asserts.Nil(fs)
	}
}

func TestFileSystem_Recycle(t *testing.T) {
	fs := &FileSystem{
		User:       &model.User{},
		Policy:     &model.Policy{},
		FileTarget: []model.File{model.File{}},
		DirTarget:  []model.Folder{model.Folder{}},
		Hooks:      map[string][]Hook{"AfterUpload": []Hook{GenericAfterUpdate}},
	}
	fs.Recycle()
	newFS := getEmptyFS()
	if fs != newFS {
		t.Error("指针不一致")
	}
}
