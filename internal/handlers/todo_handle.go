package handlers

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/emarifer/go-frameworkless-htmx/internal/services"
	"github.com/emarifer/go-frameworkless-htmx/internal/utils/upper"
)

type TaskService interface {
	CreateTodo(t services.Todo) (services.Todo, error)
	GetAllTodos(createdBy int) ([]services.Todo, error)
	GetTodoById(t services.Todo) (services.Todo, error)
	UpdateTodo(t services.Todo) (services.Todo, error)
	DeleteTodo(t services.Todo) error
}

func NewTodoHandle(l *slog.Logger, ts TaskService) *TodoHandle {

	return &TodoHandle{
		logger:      l,
		todoService: ts,
	}
}

type TodoHandle struct {
	logger      *slog.Logger
	todoService TaskService
}

func (th *TodoHandle) todoListHandle(
	w http.ResponseWriter, r *http.Request,
) error {
	errMsg, succMsg := GetMessages(w, r)

	// fmt.Println("User data:", requestUserData(r.Context()))

	todos, err := th.todoService.GetAllTodos(requestUserData(r.Context()).ID)
	if err != nil {
		if strings.Contains(err.Error(), "no such table") ||
			strings.Contains(err.Error(), "database is locked") {
			// "no such table" is the error that SQLite3 produces
			// when some table does not exist, and we have only
			// used it as an example of the errors that can be caught.
			// Here you can add the errors that you are interested
			// in throwing as `500` codes.
			return apiError{
				message: "error 500: database temporarily out of service",
				status:  http.StatusInternalServerError,
				handler: asCaller(),
				logger:  th.logger,
			}
		}
	}

	title := fmt.Sprintf(
		"| %s's Task List",
		upper.Cap(requestUserData(r.Context()).Username),
	)

	data := map[string]any{
		"title":         title,
		"fromProtected": true,
		"username":      upper.Cap(requestUserData(r.Context()).Username),
		"todos":         todos,
		"errMsg":        errMsg,
		"succMsg":       succMsg,
	}

	return renderView(
		w, r, asCaller(), th.logger, "todo_list.tmpl", data,
	)
}

func (th *TodoHandle) createTodoHandle(
	w http.ResponseWriter, r *http.Request,
) error {

	data := map[string]any{
		"title":         "| Create Todo",
		"fromProtected": true,
		"username":      upper.Cap(requestUserData(r.Context()).Username),
	}

	return renderView(
		w, r, asCaller(), th.logger, "todo_create.tmpl", data,
	)
}

func (th *TodoHandle) createTodoPostHandle(
	w http.ResponseWriter, r *http.Request,
) error {
	newTodo := services.Todo{
		CreatedBy:   requestUserData(r.Context()).ID,
		Title:       strings.Trim(r.FormValue("title"), " "),
		Description: strings.Trim(r.FormValue("description"), " "),
	}

	if newTodo.Title == "" {
		fm := []byte("Task title empty!!")
		SetFlash(w, "error", fm)

		http.Redirect(w, r, "/todo", http.StatusSeeOther)

		return nil
	}

	_, err := th.todoService.CreateTodo(newTodo)
	if err != nil {
		if strings.Contains(err.Error(), "no such table") ||
			strings.Contains(err.Error(), "database is locked") {
			// "no such table" is the error that SQLite3 produces
			// when some table does not exist, and we have only
			// used it as an example of the errors that can be caught.
			// Here you can add the errors that you are interested
			// in throwing as `500` codes.
			return apiError{
				message: "error 500: database temporarily out of service",
				status:  http.StatusInternalServerError,
				handler: asCaller(),
				logger:  th.logger,
			}
		}
	}

	fm := []byte("Task successfully created!!")
	SetFlash(w, "success", fm)

	http.Redirect(w, r, "/todo", http.StatusSeeOther)

	return nil
}

func (th *TodoHandle) editTodoHandle(
	w http.ResponseWriter, r *http.Request,
) error {
	idStr := r.URL.Query().Get("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		msg := fmt.Sprintf("Go could not convert to integer: %s", err)
		return apiError{
			message: msg,
			status:  http.StatusInternalServerError,
			handler: asCaller(),
			logger:  th.logger,
		}
	}

	// fmt.Println("ID:", id)
	userID := requestUserData(r.Context()).ID
	username := requestUserData(r.Context()).Username
	tzone := requestUserData(r.Context()).Tzone

	t := services.Todo{
		ID:        id,
		CreatedBy: userID,
	}

	todo, err := th.todoService.GetTodoById(t)
	if err != nil {
		if strings.Contains(err.Error(), "no such table") ||
			strings.Contains(err.Error(), "database is locked") {
			// "no such table" is the error that SQLite3 produces
			// when some table does not exist, and we have only
			// used it as an example of the errors that can be caught.
			// Here you can add the errors that you are interested
			// in throwing as `500` codes.
			return apiError{
				message: "error 500: database temporarily out of service",
				status:  http.StatusInternalServerError,
				handler: asCaller(),
				logger:  th.logger,
			}
		}
		msg := fmt.Sprintf("something went wrong:%s", err)
		fm := []byte(msg)
		SetFlash(w, "error", fm)

		http.Redirect(w, r, "/todo", http.StatusSeeOther)

		return nil
	}

	data := map[string]any{
		"title":         fmt.Sprintf("| Edit Todo #%s", idStr),
		"fromProtected": true,
		"username":      upper.Cap(username),
		"taskID":        todo.ID,
		"taskTitle":     todo.Title,
		"taskDesc":      todo.Description,
		"taskStatus":    todo.Status,
		"createdAt":     services.ConvertDateTime(tzone, todo.CreatedAt),
	}

	return renderView(
		w, r, asCaller(), th.logger, "todo_update.tmpl", data,
	)
}

func (th *TodoHandle) editTodoPostHandle(
	w http.ResponseWriter, r *http.Request,
) error {
	idStr := r.URL.Query().Get("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		msg := fmt.Sprintf("Go could not convert to integer: %s", err)
		return apiError{
			message: msg,
			status:  http.StatusInternalServerError,
			handler: asCaller(),
			logger:  th.logger,
		}
	}

	var status bool
	if r.FormValue("status") == "on" {
		status = true
	} else {
		status = false
	}

	fmt.Println("Status:", status)

	t := services.Todo{
		ID:          id,
		Title:       strings.Trim(r.FormValue("title"), " "),
		Description: strings.Trim(r.FormValue("description"), " \n"),
		Status:      status,
		CreatedBy:   requestUserData(r.Context()).ID,
	}

	_, err = th.todoService.UpdateTodo(t)
	if err != nil {
		if strings.Contains(err.Error(), "no such table") ||
			strings.Contains(err.Error(), "database is locked") {
			// "no such table" is the error that SQLite3 produces
			// when some table does not exist, and we have only
			// used it as an example of the errors that can be caught.
			// Here you can add the errors that you are interested
			// in throwing as `500` codes.
			return apiError{
				message: "error 500: database temporarily out of service",
				status:  http.StatusInternalServerError,
				handler: asCaller(),
				logger:  th.logger,
			}
		}
		return apiError{
			message: fmt.Sprintf(
				"error 500: task could not be updated: %s", err,
			),
			status:  http.StatusInternalServerError,
			handler: asCaller(),
			logger:  th.logger,
		}
	}

	fm := []byte("Task successfully updated!!")
	SetFlash(w, "success", fm)

	http.Redirect(w, r, "/todo", http.StatusSeeOther)

	return nil
}

func (th *TodoHandle) deleteTodoHandle(
	w http.ResponseWriter, r *http.Request,
) error {
	idStr := r.URL.Query().Get("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		msg := fmt.Sprintf("Go could not convert to integer: %s", err)
		return apiError{
			message: msg,
			status:  http.StatusInternalServerError,
			handler: asCaller(),
			logger:  th.logger,
		}
	}

	t := services.Todo{
		ID:        id,
		CreatedBy: requestUserData(r.Context()).ID,
	}

	err = th.todoService.DeleteTodo(t)
	if err != nil {
		if strings.Contains(err.Error(), "no such table") ||
			strings.Contains(err.Error(), "database is locked") {
			// "no such table" is the error that SQLite3 produces
			// when some table does not exist, and we have only
			// used it as an example of the errors that can be caught.
			// Here you can add the errors that you are interested
			// in throwing as `500` codes.
			return apiError{
				message: "error 500: database temporarily out of service",
				status:  http.StatusInternalServerError,
				handler: asCaller(),
				logger:  th.logger,
			}
		}

		msg := fmt.Sprintf("something went wrong:%s", err)
		fm := []byte(msg)
		SetFlash(w, "error", fm)

		http.Redirect(w, r, "/todo", http.StatusSeeOther)

		return nil
	}

	fm := []byte("Task successfully deleted!!")
	SetFlash(w, "success", fm)

	http.Redirect(w, r, "/todo", http.StatusSeeOther)

	return nil
}
