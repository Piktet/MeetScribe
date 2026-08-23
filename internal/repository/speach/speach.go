// Package speach — взаимодействие с SaluteSpeech API (загрузка файлов, создание задач, проверка статуса, скачивание результатов).
package speach

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"

	"github.com/Piktet/tg_bot/internal/logger"
	"github.com/Piktet/tg_bot/internal/model"

	"go.uber.org/zap"
)

// Upload загружает аудиофайл в SaluteSpeech API.
// host — хост API, token — токен авторизации, data — поток аудио данных.
// Возвращает ID загруженного файла.
func Upload(ctx context.Context, host, token string, data io.Reader) (string, error) {
	u, err := url.JoinPath(host, "/rest/v1/data:upload")
	if err != nil {
		logger.Log().Error("UploadVoice - create path error", zap.Error(err))
		return "", err
	}
	r, err := http.NewRequestWithContext(ctx, "POST", u, data)
	if err != nil {
		logger.Log().Error("UploadVoice - create request", zap.Error(err))
		return "", err
	}

	r.Header.Add("Authorization", "Bearer "+token)
	r.Header.Add("Content-Type", "audio/mpeg")
	r.Header.Set("Accept", "application/json")

	client := &http.Client{}
	resp, err := client.Do(r)
	if err != nil {
		logger.Log().Error("UploadVoice - send request", zap.Error(err))
		return "", err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err := errors.New(http.StatusText(resp.StatusCode))
		logger.Log().Error("UploadVoice - get response", zap.Error(err))
		return "", err
	}

	var x model.SpeachUploadResponse
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&x); err != nil {
		logger.Log().Error("UploadVoice - decode result", zap.Error(err))
		return "", err
	}

	if x.Status != http.StatusOK {
		return "", errors.New(http.StatusText(x.Status))
	}

	return x.Result.FileID, nil

}

// CreateTask создает задачу на распознавание речи в SaluteSpeech API.
// host — хост API, token — токен авторизации, fileID — ID загруженного аудиофайла.
// Возвращает ID задачи, ID выходного файла, статус и ошибку.
func CreateTask(ctx context.Context, host, token, fileID string) (string, string, model.ResultStatusType, error) {
	u, err := url.JoinPath(host, "rest/v1/speech:async_recognize")
	if err != nil {
		logger.Log().Error("CreateTask - create path error", zap.Error(err))
		return "", "", model.SpeachResultStatusEmpty, err
	}
	data, err := json.Marshal(model.SpeachCreateTaskRequest{
		FileID: fileID,
		Options: model.SpeachCreateTaskOptionRequest{
			AudioEncoding: "PCM_S16LE",
		},
	})
	r, err := http.NewRequestWithContext(ctx, "POST", u, bytes.NewReader(data))
	if err != nil {
		logger.Log().Error("connection to speach - create request", zap.Error(err))
		return "", "", model.SpeachResultStatusEmpty, err
	}
	r.Header.Set("Authorization", "Bearer "+token)
	r.Header.Set("Accept", "application/json")
	r.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(r)
	if err != nil {
		logger.Log().Error("CreateTask - send request", zap.Error(err))
		return "", "", model.SpeachResultStatusEmpty, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err := errors.New(http.StatusText(resp.StatusCode))
		logger.Log().Error("CreateTask - get response", zap.Error(err))
		return "", "", model.SpeachResultStatusEmpty, err
	}

	return parseStatus(resp.Body)

}

// GetStatus запрашивает статус задачи распознавания речи.
// host — хост API, token — токен авторизации, taskID — ID задачи.
// Возвращает ID выходного файла, статус задачи, флаг необходимости повтора и ошибку.
// isRetry = true означает, что запрос стоит повторить (например, при 500 ошибке).
func GetStatus(ctx context.Context, host, token, taskID string) (string, model.ResultStatusType, bool, error) {

	u, err := url.JoinPath(host, "/rest/v1/task:get", taskID)
	if err != nil {
		logger.Log().Error("UploadVoice - create path error", zap.Error(err))
		return "", model.SpeachResultStatusEmpty, false, err
	}
	r, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		logger.Log().Error("UploadVoice - create request", zap.Error(err))
		return "", model.SpeachResultStatusEmpty, false, err
	}

	r.Header.Set("Authorization", "Bearer "+token)
	r.Header.Set("Accept", "application/octet-stream")

	client := &http.Client{}
	resp, err := client.Do(r)
	if err != nil {
		logger.Log().Error("UploadVoice - send request", zap.Error(err))
		return "", model.SpeachResultStatusEmpty, false, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err := errors.New(http.StatusText(resp.StatusCode))
		logger.Log().Error("UploadVoice - get response", zap.Error(err))
		switch resp.StatusCode {
		case http.StatusInternalServerError:
			return "", model.SpeachResultStatusEmpty, true, err
		default:
			return "", model.SpeachResultStatusEmpty, false, err

		}
	}

	_, fileID, status, err := parseStatus(resp.Body)
	if err != nil {
		return "", model.SpeachResultStatusEmpty, false, err
	}
	return fileID, status, false, nil

}

// Download скачивает результат распознавания по ID файла.
// host — хост API, token — токен авторизации, fileID — ID файла с результатом.
func Download(ctx context.Context, host, token, fileID string) ([]byte, error) {

	u, err := url.JoinPath(host, "rest/v1/data:download")
	if err != nil {
		logger.Log().Error("Download - create path error", zap.Error(err))
		return nil, err
	}

	v := url.Values{}
	v.Set("response_file_id", fileID)

	r, err := http.NewRequestWithContext(ctx, "GET", u+"?"+v.Encode(), nil)
	if err != nil {
		logger.Log().Error("Download - create request", zap.Error(err))
		return nil, err
	}

	r.Header.Set("Authorization", "Bearer "+token)
	r.Header.Set("Accept", "application/octet-stream")

	client := &http.Client{}
	resp, err := client.Do(r)
	if err != nil {
		logger.Log().Error("Download - send request", zap.Error(err))
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err := errors.New(http.StatusText(resp.StatusCode))
		logger.Log().Error("Download - get response", zap.Error(err))
		return nil, err
	}

	x, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Log().Error("Download - read body", zap.Error(err))
		return nil, err

	}
	return x, nil
}

// parseStatus парсит ответ API задачи распознавания.
func parseStatus(body io.Reader) (string, string, model.ResultStatusType, error) {

	var x model.SpeachCreateTaskResponse
	dec := json.NewDecoder(body)
	if err := dec.Decode(&x); err != nil {
		logger.Log().Error("CreateTask - decode result", zap.Error(err))
		return "", "", model.SpeachResultStatusEmpty, err
	}

	if x.Status != http.StatusOK {
		return "", "", model.SpeachResultStatusEmpty, errors.New(http.StatusText(x.Status))
	}

	return x.Result.ID, x.Result.FileID, model.ResultStatusType(x.Result.Status), nil

}
