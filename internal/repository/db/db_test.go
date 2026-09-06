package db_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Piktet/MeetScribe/internal/mock"
	"github.com/Piktet/MeetScribe/internal/model"
	"github.com/Piktet/MeetScribe/internal/repository/db"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeRows — заглушка model.Rows с настраиваемыми данными и ошибками.
type fakeRows struct {
	data    [][]any
	i       int
	err     error // ошибка, возвращаемая Err()
	scanErr error // ошибка, возвращаемая Scan()
	closed  bool
}

var _ model.Rows = (*fakeRows)(nil)

func (r *fakeRows) Close() error               { r.closed = true; return nil }
func (r *fakeRows) Columns() ([]string, error) { return []string{"id"}, nil }
func (r *fakeRows) Next() bool {
	if r.i >= len(r.data) {
		return false
	}
	r.i++
	return true
}
func (r *fakeRows) Scan(dest ...any) error {
	if r.scanErr != nil {
		return r.scanErr
	}
	row := r.data[r.i-1]
	for i := range dest {
		if i >= len(row) {
			continue
		}
		switch d := dest[i].(type) {
		case *string:
			*d = row[i].(string)
		case *int64:
			*d = row[i].(int64)
		case *time.Time:
			*d = row[i].(time.Time)
		case *model.TranscriptionStatus:
			*d = row[i].(model.TranscriptionStatus)
		}
	}
	return nil
}
func (r *fakeRows) Err() error { return r.err }

func TestCreate(t *testing.T) {
	ctx := context.Background()
	m := &mock.MockDB{}

	err := db.Create(ctx, m)
	require.NoError(t, err)
	assert.Equal(t, 1, m.ExecuteCalls)
	assert.Contains(t, m.ExecuteSQL, "create table if not exists users")
	assert.Contains(t, m.ExecuteSQL, "create table if not exists transcriptions")
}

func TestAddUser(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		m := &mock.MockDB{}
		err := db.AddUser(ctx, m, 1, 2, "alice")
		require.NoError(t, err)
		assert.Equal(t, 1, m.ExecuteCalls)
		assert.Contains(t, m.ExecuteSQL, "insert into users")
		assert.Equal(t, []any{int64(1), int64(2), "alice"}, m.ExecuteArgs)
	})

	t.Run("error", func(t *testing.T) {
		m := &mock.MockDB{ExecuteError: errors.New("db error")}
		err := db.AddUser(ctx, m, 1, 2, "alice")
		assert.Error(t, err)
	})
}

func TestSaveTranscription(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		m := &mock.MockDB{}
		tr := &model.Transcription{
			UserID:        1,
			ChatID:        2,
			Name:          "meeting",
			FilePath:      "in-file",
			OutputFileID:  "out-file",
			TaskID:        "task-1",
			Transcription: "text",
			Summary:       "short",
			Status:        model.StatusDone,
		}
		err := db.SaveTranscription(ctx, m, tr)
		require.NoError(t, err)
		assert.Equal(t, 1, m.ExecuteCalls)
		assert.Contains(t, m.ExecuteSQL, "insert into transcriptions")
		assert.Equal(t, []any{
			int64(1), int64(2), "meeting", "in-file", "out-file",
			"task-1", "text", "short", model.StatusDone,
		}, m.ExecuteArgs)
	})

	t.Run("error", func(t *testing.T) {
		m := &mock.MockDB{ExecuteError: errors.New("db error")}
		err := db.SaveTranscription(ctx, m, &model.Transcription{})
		assert.Error(t, err)
	})
}

func TestUpdateTranscriptionStatus(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		m := &mock.MockDB{}
		err := db.UpdateTranscriptionStatus(ctx, m, 7, "new text", "new summary", "DONE")
		require.NoError(t, err)
		assert.Equal(t, 1, m.ExecuteCalls)
		assert.Contains(t, m.ExecuteSQL, "update transcriptions")
		assert.Equal(t, []any{int64(7), "new text", "new summary", "DONE"}, m.ExecuteArgs)
	})

	t.Run("error", func(t *testing.T) {
		m := &mock.MockDB{ExecuteError: errors.New("db error")}
		err := db.UpdateTranscriptionStatus(ctx, m, 7, "", "", "")
		assert.Error(t, err)
	})
}

func TestGetUserTranscriptions(t *testing.T) {
	ctx := context.Background()
	created := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	t.Run("success", func(t *testing.T) {
		m := &mock.MockDB{
			QueryRows: &fakeRows{data: [][]any{
				{"1", "first", model.StatusDone, created},
				{"2", "second", model.StatusPending, created},
			}},
		}
		list, err := db.GetUserTranscriptions(ctx, m, 10)
		require.NoError(t, err)
		require.Len(t, list, 2)
		assert.Equal(t, "1", list[0].ID)
		assert.Equal(t, "first", list[0].Name)
		assert.Equal(t, model.StatusDone, list[0].Status)
		assert.Equal(t, created, list[0].CreatedAt)
		assert.Equal(t, int64(10), list[0].UserID)
	})

	t.Run("empty", func(t *testing.T) {
		m := &mock.MockDB{QueryRows: &fakeRows{}}
		list, err := db.GetUserTranscriptions(ctx, m, 10)
		require.NoError(t, err)
		assert.Empty(t, list)
	})

	t.Run("query error", func(t *testing.T) {
		m := &mock.MockDB{QueryError: errors.New("db error")}
		list, err := db.GetUserTranscriptions(ctx, m, 10)
		assert.Error(t, err)
		assert.Nil(t, list)
	})

	t.Run("scan error", func(t *testing.T) {
		m := &mock.MockDB{
			QueryRows: &fakeRows{
				data:    [][]any{{"1", "first", model.StatusDone, created}},
				scanErr: errors.New("scan error"),
			},
		}
		list, err := db.GetUserTranscriptions(ctx, m, 10)
		assert.Error(t, err)
		assert.Nil(t, list)
	})

	t.Run("rows error", func(t *testing.T) {
		m := &mock.MockDB{
			QueryRows: &fakeRows{data: [][]any{
				{"1", "first", model.StatusDone, created},
			}, err: errors.New("rows error")},
		}
		list, err := db.GetUserTranscriptions(ctx, m, 10)
		assert.Error(t, err)
		assert.Nil(t, list)
	})
}

func TestGetTranscriptionByID(t *testing.T) {
	ctx := context.Background()
	created := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	updated := time.Date(2026, 1, 2, 4, 4, 5, 0, time.UTC)

	fullRow := []any{
		"5", int64(10), int64(20), "meeting", "in-file", "out-file",
		"task-1", "transcript text", "summary text", model.StatusDone, created, updated,
	}

	t.Run("success", func(t *testing.T) {
		m := &mock.MockDB{QueryRows: &fakeRows{data: [][]any{fullRow}}}
		tr, err := db.GetTranscriptionByID(ctx, m, 10, 5)
		require.NoError(t, err)
		assert.Equal(t, "5", tr.ID)
		assert.Equal(t, int64(10), tr.UserID)
		assert.Equal(t, int64(20), tr.ChatID)
		assert.Equal(t, "meeting", tr.Name)
		assert.Equal(t, "in-file", tr.FilePath)
		assert.Equal(t, "out-file", tr.OutputFileID)
		assert.Equal(t, "task-1", tr.TaskID)
		assert.Equal(t, "transcript text", tr.Transcription)
		assert.Equal(t, "summary text", tr.Summary)
		assert.Equal(t, model.StatusDone, tr.Status)
		assert.Equal(t, created, tr.CreatedAt)
		assert.Equal(t, updated, tr.UpdatedAt)
	})

	t.Run("not found", func(t *testing.T) {
		m := &mock.MockDB{QueryRows: &fakeRows{}}
		_, err := db.GetTranscriptionByID(ctx, m, 10, 5)
		assert.ErrorContains(t, err, "not found")
	})

	t.Run("access denied", func(t *testing.T) {
		m := &mock.MockDB{QueryRows: &fakeRows{data: [][]any{fullRow}}}
		_, err := db.GetTranscriptionByID(ctx, m, 999, 5)
		assert.ErrorContains(t, err, "access denied")
	})

	t.Run("query error", func(t *testing.T) {
		m := &mock.MockDB{QueryError: errors.New("db error")}
		_, err := db.GetTranscriptionByID(ctx, m, 10, 5)
		assert.Error(t, err)
	})

	t.Run("scan error", func(t *testing.T) {
		m := &mock.MockDB{
			QueryRows: &fakeRows{
				data:    [][]any{fullRow},
				scanErr: errors.New("scan error"),
			},
		}
		_, err := db.GetTranscriptionByID(ctx, m, 10, 5)
		assert.Error(t, err)
	})
}

func TestSearchTranscriptions(t *testing.T) {
	ctx := context.Background()
	created := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	t.Run("success", func(t *testing.T) {
		m := &mock.MockDB{
			QueryRows: &fakeRows{data: [][]any{
				{"1", "meeting about go", model.StatusDone, created},
			}},
		}
		list, err := db.SearchTranscriptions(ctx, m, 10, "go")
		require.NoError(t, err)
		require.Len(t, list, 1)
		assert.Equal(t, "1", list[0].ID)
		assert.Equal(t, int64(10), list[0].UserID)
	})

	t.Run("empty", func(t *testing.T) {
		m := &mock.MockDB{QueryRows: &fakeRows{}}
		list, err := db.SearchTranscriptions(ctx, m, 10, "go")
		require.NoError(t, err)
		assert.Empty(t, list)
	})

	t.Run("query error", func(t *testing.T) {
		m := &mock.MockDB{QueryError: errors.New("db error")}
		list, err := db.SearchTranscriptions(ctx, m, 10, "go")
		assert.Error(t, err)
		assert.Nil(t, list)
	})

	t.Run("rows error", func(t *testing.T) {
		m := &mock.MockDB{
			QueryRows: &fakeRows{
				data: [][]any{{"1", "meeting", model.StatusDone, created}},
				err:  errors.New("rows error"),
			},
		}
		list, err := db.SearchTranscriptions(ctx, m, 10, "go")
		assert.Error(t, err)
		assert.Nil(t, list)
	})
}

func TestAddTask(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		m := &mock.MockDB{}
		task := &model.SpeachTaskResponse{
			SpeachTaskData: &model.SpeachTaskData{
				User:   1,
				ChatID: 2,
				Name:   "meeting",
			},
			InFileID:  "in-file",
			OutFileID: "out-file",
			TaskID:    "task-1",
			Output:    []byte("transcript"),
		}
		err := db.AddTask(ctx, m, task, "short summary")
		require.NoError(t, err)
		assert.Equal(t, 1, m.ExecuteCalls)
		assert.Contains(t, m.ExecuteSQL, "insert into transcriptions")
		assert.Equal(t, []any{
			int64(1), int64(2), "meeting", "in-file", "out-file",
			"task-1", "transcript", "short summary", model.StatusDone,
		}, m.ExecuteArgs)
	})

	t.Run("error", func(t *testing.T) {
		m := &mock.MockDB{ExecuteError: errors.New("db error")}
		err := db.AddTask(ctx, m, &model.SpeachTaskResponse{SpeachTaskData: &model.SpeachTaskData{}}, "")
		assert.Error(t, err)
	})
}

func TestGetUserFile(t *testing.T) {
	ctx := context.Background()
	created := time.Date(2026, 1, 2, 15, 4, 0, 0, time.UTC)

	t.Run("empty list", func(t *testing.T) {
		m := &mock.MockDB{QueryRows: &fakeRows{}}
		result, err := db.GetUserFile(ctx, m, 10)
		require.NoError(t, err)
		assert.Equal(t, "список пуст", result)
	})

	t.Run("with data", func(t *testing.T) {
		m := &mock.MockDB{
			QueryRows: &fakeRows{data: [][]any{
				{"1", "meeting", model.StatusDone, created},
			}},
		}
		result, err := db.GetUserFile(ctx, m, 10)
		require.NoError(t, err)
		assert.Contains(t, result, "ID: 1")
		assert.Contains(t, result, "meeting")
		assert.Contains(t, result, "2026-01-02 15:04")
	})

	t.Run("error", func(t *testing.T) {
		m := &mock.MockDB{QueryError: errors.New("db error")}
		_, err := db.GetUserFile(ctx, m, 10)
		assert.Error(t, err)
	})
}

func TestGetUserFileItem(t *testing.T) {
	ctx := context.Background()
	created := time.Date(2026, 1, 2, 15, 4, 0, 0, time.UTC)
	fullRow := []any{
		"5", int64(10), int64(20), "meeting", "in-file", "out-file",
		"task-1", "transcript text", "summary text", model.StatusDone, created, created,
	}

	t.Run("invalid id", func(t *testing.T) {
		m := &mock.MockDB{}
		_, err := db.GetUserFileItem(ctx, m, 10, "not-a-number")
		assert.ErrorContains(t, err, "invalid id")
	})

	t.Run("success", func(t *testing.T) {
		m := &mock.MockDB{QueryRows: &fakeRows{data: [][]any{fullRow}}}
		result, err := db.GetUserFileItem(ctx, m, 10, "5")
		require.NoError(t, err)
		assert.True(t, strings.Contains(result, "meeting"))
		assert.True(t, strings.Contains(result, "transcript text"))
		assert.True(t, strings.Contains(result, "summary text"))
	})

	t.Run("not found", func(t *testing.T) {
		m := &mock.MockDB{QueryRows: &fakeRows{}}
		_, err := db.GetUserFileItem(ctx, m, 10, "5")
		assert.Error(t, err)
	})
}

func TestGetFileByWord(t *testing.T) {
	ctx := context.Background()
	created := time.Date(2026, 1, 2, 15, 4, 0, 0, time.UTC)

	t.Run("nothing found", func(t *testing.T) {
		m := &mock.MockDB{QueryRows: &fakeRows{}}
		result, err := db.GetFileByWord(ctx, m, 10, "go")
		require.NoError(t, err)
		assert.Equal(t, "ничего не найдено", result)
	})

	t.Run("found", func(t *testing.T) {
		m := &mock.MockDB{
			QueryRows: &fakeRows{data: [][]any{
				{"1", "meeting about go", model.StatusDone, created},
			}},
		}
		result, err := db.GetFileByWord(ctx, m, 10, "go")
		require.NoError(t, err)
		assert.Contains(t, result, "ID: 1")
		assert.Contains(t, result, "meeting about go")
	})

	t.Run("error", func(t *testing.T) {
		m := &mock.MockDB{QueryError: errors.New("db error")}
		_, err := db.GetFileByWord(ctx, m, 10, "go")
		assert.Error(t, err)
	})
}
