package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/kerucko/diploma/gateway/internal/models"
	"github.com/kerucko/diploma/gateway/internal/utils"
)

type userService interface {
	Register(ctx context.Context, input models.RegisterRequest) (*models.User, error)
	Login(ctx context.Context, input models.LoginRequest) (string, error)
	GetAll(ctx context.Context) ([]models.User, error)
	Create(ctx context.Context, input *models.User) (*models.User, error)
	GetByName(ctx context.Context, name string) (*models.User, error)
	GetByID(ctx context.Context, id int64) (*models.User, error)
}

type taskService interface {
	Create(ctx context.Context, input *models.Task) (*models.Task, error)
	GetByID(ctx context.Context, id int64) (*models.Task, error)
	GetByUserID(ctx context.Context, userID int64) ([]models.Task, error)
	GetByUser(ctx context.Context, userID int64) ([]models.Task, error)
	GetAll(ctx context.Context) ([]models.Task, error)
}

type Handler struct {
	UserService userService
	TaskService taskService
	Auth        *utils.AuthManager
}

func NewHandler(us userService, ts taskService, auth *utils.AuthManager) *Handler {
	return &Handler{
		UserService: us,
		TaskService: ts,
		Auth:        auth,
	}
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var input models.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}
	user, err := h.UserService.Register(r.Context(), input)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(user)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var input models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}
	token, err := h.UserService.Login(r.Context(), input)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"token": token})
}

func (h *Handler) CreateTask(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(utils.ContextUserID).(int64)
	var task *models.Task
	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}
	task.UserID = userID
	created, err := h.TaskService.Create(r.Context(), task)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(created)
}

func (h *Handler) GetTask(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid task ID", http.StatusBadRequest)
		return
	}
	task, err := h.TaskService.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(task)
}

func (h *Handler) GetMyTasks(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(utils.ContextUserID).(int64)
	tasks, err := h.TaskService.GetByUser(r.Context(), userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(tasks)
}

func (h *Handler) AdminGetAllTasks(w http.ResponseWriter, r *http.Request) {
	tasks, err := h.TaskService.GetAll(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(tasks)
}

func (h *Handler) AdminGetUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.UserService.GetAll(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(users)
}

func (h *Handler) AdminGetUserProfile(w http.ResponseWriter, r *http.Request) {
	userID, _ := strconv.ParseInt(chi.URLParam(r, "userID"), 10, 64)
	user, err := h.UserService.GetByID(r.Context(), userID)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(user)
}

func (h *Handler) AdminGetUserTasks(w http.ResponseWriter, r *http.Request) {
	userID, _ := strconv.ParseInt(chi.URLParam(r, "userID"), 10, 64)
	tasks, err := h.TaskService.GetByUser(r.Context(), userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(tasks)
}
