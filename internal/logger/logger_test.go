package logger

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInitLogger(t *testing.T) {

	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		level   string
		wantErr bool
	}{
		{
			level:   "Info",
			wantErr: false,
		},
		{
			level:   "Aaa",
			wantErr: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := InitLogger(test.level)
			if test.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_loggingResponseWriter_Write(t *testing.T) {

	data := []byte("qwerty")
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		b       []byte
		want    int
		wantErr bool
	}{
		{
			b:       data,
			want:    len(data),
			wantErr: false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			lw := loggingResponseWriter{
				ResponseWriter: w, // встраиваем оригинальный http.ResponseWriter
				responseData:   &responseData{},
			}
			len, err := lw.Write(test.b)
			if assert.NoError(t, err) {
				assert.Equal(t, test.want, lw.responseData.size)
				assert.Equal(t, test.want, len)
			}
		})
	}
}

func Test_loggingResponseWriter_WriteHeader(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		status int
		want   int
	}{
		{
			name:   "positive",
			status: http.StatusOK,
			want:   http.StatusOK,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			lw := loggingResponseWriter{
				ResponseWriter: w, // встраиваем оригинальный http.ResponseWriter
				responseData:   &responseData{},
			}
			lw.WriteHeader(test.status)
			assert.Equal(t, test.want, lw.responseData.status)

		})
	}
}
