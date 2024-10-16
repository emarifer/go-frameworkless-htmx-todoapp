package services

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type Todo struct {
	ID          int       `json:"id"`
	CreatedBy   int       `json:"created_by"`
	Title       string    `json:"title"`
	Description string    `json:"description,omitempty"`
	Status      bool      `json:"status,omitempty"`
	CreatedAt   time.Time `json:"created_at,omitempty"`
}

type TodoService struct {
	Todo      Todo
	TodoStore *sql.DB
}

func NewTodoService(t Todo, tStore *sql.DB) *TodoService {

	return &TodoService{
		Todo:      t,
		TodoStore: tStore,
	}
}
func (ts *TodoService) CreateTodo(t Todo) (Todo, error) {

	query := `INSERT INTO todos (created_by, title, description)
		VALUES(?, ?, ?) RETURNING *`

	stmt, err := ts.TodoStore.Prepare(query)
	if err != nil {
		return Todo{}, err
	}

	defer stmt.Close()

	err = stmt.QueryRow(
		t.CreatedBy,
		t.Title,
		t.Description,
	).Scan(
		&ts.Todo.ID,
		&ts.Todo.CreatedBy,
		&ts.Todo.Title,
		&ts.Todo.Description,
		&ts.Todo.Status,
		&ts.Todo.CreatedAt,
	)
	if err != nil {
		return Todo{}, err
	}

	/* if i, err := result.RowsAffected(); err != nil || i != 1 {
		return errors.New("error: an affected row was expected")
	} */

	return ts.Todo, nil
}

func (ts *TodoService) GetAllTodos(createdBy int) ([]Todo, error) {
	query := fmt.Sprintf("SELECT id, title, status FROM todos WHERE created_by = %d ORDER BY created_at DESC", createdBy)

	rows, err := ts.TodoStore.Query(query)
	if err != nil {
		return []Todo{}, err
	}
	// We close the resource
	defer rows.Close()

	todos := []Todo{}
	for rows.Next() {
		err := rows.Scan(&ts.Todo.ID, &ts.Todo.Title, &ts.Todo.Status)
		if err != nil {
			continue
		}

		todos = append(todos, ts.Todo)
	}

	return todos, nil
}

func (ts *TodoService) GetTodoById(t Todo) (Todo, error) {

	query := `SELECT id, title, description, status, created_at FROM todos
		WHERE created_by = ? AND id=?`

	stmt, err := ts.TodoStore.Prepare(query)
	if err != nil {
		return Todo{}, err
	}

	defer stmt.Close()

	err = stmt.QueryRow(
		t.CreatedBy,
		t.ID,
	).Scan(
		&ts.Todo.ID,
		&ts.Todo.Title,
		&ts.Todo.Description,
		&ts.Todo.Status,
		&ts.Todo.CreatedAt,
	)
	if err != nil {
		return Todo{}, err
	}

	return ts.Todo, nil
}

func (ts *TodoService) UpdateTodo(t Todo) (Todo, error) {

	query := `UPDATE todos SET title = ?,  description = ?, status = ?
		WHERE created_by = ? AND id=? RETURNING id, title, description, status`

	stmt, err := ts.TodoStore.Prepare(query)
	if err != nil {
		return Todo{}, err
	}

	defer stmt.Close()

	err = stmt.QueryRow(
		t.Title,
		t.Description,
		t.Status,
		t.CreatedBy,
		t.ID,
	).Scan(
		&ts.Todo.ID,
		&ts.Todo.Title,
		&ts.Todo.Description,
		&ts.Todo.Status,
	)
	if err != nil {
		return Todo{}, err
	}

	return ts.Todo, nil
}

func (ts *TodoService) DeleteTodo(t Todo) error {

	query := `DELETE FROM todos
		WHERE created_by = ? AND id=?`

	stmt, err := ts.TodoStore.Prepare(query)
	if err != nil {
		return err
	}

	defer stmt.Close()

	result, err := stmt.Exec(t.CreatedBy, t.ID)
	if err != nil {
		return err
	}

	if i, err := result.RowsAffected(); err != nil || i != 1 {
		return errors.New("an affected row was expected")
	}

	return nil
}

func ConvertDateTime(tz string, dt time.Time) string {
	loc, _ := time.LoadLocation(tz)

	return dt.In(loc).Format(time.RFC822Z)
}
