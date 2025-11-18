package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/therealutkarshpriyadarshi/dag/internal/dag"
	"github.com/therealutkarshpriyadarshi/dag/pkg/api/dto"
	"github.com/therealutkarshpriyadarshi/dag/pkg/api/handlers"
	"github.com/therealutkarshpriyadarshi/dag/pkg/models"
)

// MockDAGRepository is a mock implementation of DAGRepository
type MockDAGRepository struct {
	mock.Mock
}

func (m *MockDAGRepository) Create(ctx interface{}, dag *models.DAG) error {
	args := m.Called(ctx, dag)
	return args.Error(0)
}

func (m *MockDAGRepository) Get(ctx interface{}, id string) (*models.DAG, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.DAG), args.Error(1)
}

func (m *MockDAGRepository) GetByID(ctx interface{}, id string) (*models.DAG, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.DAG), args.Error(1)
}

func (m *MockDAGRepository) GetByName(ctx interface{}, name string) (*models.DAG, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.DAG), args.Error(1)
}

func (m *MockDAGRepository) List(ctx interface{}, filters ...interface{}) ([]*models.DAG, error) {
	args := m.Called(ctx, filters)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.DAG), args.Error(1)
}

func (m *MockDAGRepository) Update(ctx interface{}, dag *models.DAG) error {
	args := m.Called(ctx, dag)
	return args.Error(0)
}

func (m *MockDAGRepository) Delete(ctx interface{}, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockDAGRepository) Pause(ctx interface{}, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockDAGRepository) Unpause(ctx interface{}, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func TestCreateDAG(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("successful creation", func(t *testing.T) {
		mockRepo := new(MockDAGRepository)
		engine := dag.NewEngine()
		handler := handlers.NewDAGHandler(mockRepo, engine)

		mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.DAG")).Return(nil)

		reqBody := dto.CreateDAGRequest{
			Name:        "test_dag",
			Description: "Test DAG",
			Schedule:    "0 0 * * *",
			StartDate:   time.Now(),
			Tasks: []dto.TaskDTO{
				{
					ID:      "task1",
					Name:    "Task 1",
					Type:    "bash",
					Command: "echo hello",
					Retries: 3,
				},
			},
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/dags", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router := gin.Default()
		router.POST("/api/v1/dags", handler.CreateDAG)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		mockRepo.AssertExpectations(t)
	})

	t.Run("invalid request body", func(t *testing.T) {
		mockRepo := new(MockDAGRepository)
		engine := dag.NewEngine()
		handler := handlers.NewDAGHandler(mockRepo, engine)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/dags", bytes.NewReader([]byte("invalid json")))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router := gin.Default()
		router.POST("/api/v1/dags", handler.CreateDAG)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestListDAGs(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("successful list", func(t *testing.T) {
		mockRepo := new(MockDAGRepository)
		engine := dag.NewEngine()
		handler := handlers.NewDAGHandler(mockRepo, engine)

		dags := []*models.DAG{
			{
				ID:          "dag1",
				Name:        "Test DAG 1",
				Description: "Description 1",
				Tasks:       []models.Task{},
			},
		}

		mockRepo.On("List", mock.Anything, mock.Anything).Return(dags, nil)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/dags", nil)
		w := httptest.NewRecorder()

		router := gin.Default()
		router.GET("/api/v1/dags", handler.ListDAGs)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response dto.DAGListResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(response.DAGs))
		mockRepo.AssertExpectations(t)
	})
}

func TestGetDAG(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("successful get", func(t *testing.T) {
		mockRepo := new(MockDAGRepository)
		engine := dag.NewEngine()
		handler := handlers.NewDAGHandler(mockRepo, engine)

		dag := &models.DAG{
			ID:          "dag1",
			Name:        "Test DAG",
			Description: "Description",
			Tasks:       []models.Task{},
		}

		mockRepo.On("Get", mock.Anything, "dag1").Return(dag, nil)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/dags/dag1", nil)
		w := httptest.NewRecorder()

		router := gin.Default()
		router.GET("/api/v1/dags/:id", handler.GetDAG)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response dto.DAGResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Test DAG", response.Name)
		mockRepo.AssertExpectations(t)
	})

	t.Run("DAG not found", func(t *testing.T) {
		mockRepo := new(MockDAGRepository)
		engine := dag.NewEngine()
		handler := handlers.NewDAGHandler(mockRepo, engine)

		mockRepo.On("Get", mock.Anything, "nonexistent").Return(nil, assert.AnError)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/dags/nonexistent", nil)
		w := httptest.NewRecorder()

		router := gin.Default()
		router.GET("/api/v1/dags/:id", handler.GetDAG)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		mockRepo.AssertExpectations(t)
	})
}

func TestDeleteDAG(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("successful delete", func(t *testing.T) {
		mockRepo := new(MockDAGRepository)
		engine := dag.NewEngine()
		handler := handlers.NewDAGHandler(mockRepo, engine)

		mockRepo.On("Delete", mock.Anything, "dag1").Return(nil)

		req := httptest.NewRequest(http.MethodDelete, "/api/v1/dags/dag1", nil)
		w := httptest.NewRecorder()

		router := gin.Default()
		router.DELETE("/api/v1/dags/:id", handler.DeleteDAG)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
		mockRepo.AssertExpectations(t)
	})
}

func TestPauseDAG(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("successful pause", func(t *testing.T) {
		mockRepo := new(MockDAGRepository)
		engine := dag.NewEngine()
		handler := handlers.NewDAGHandler(mockRepo, engine)

		mockRepo.On("Pause", mock.Anything, "dag1").Return(nil)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/dags/dag1/pause", nil)
		w := httptest.NewRecorder()

		router := gin.Default()
		router.POST("/api/v1/dags/:id/pause", handler.PauseDAG)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockRepo.AssertExpectations(t)
	})
}
