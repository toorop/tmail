package rest

import (
	"github.com/stretchr/testify/assert"
	"net/http/httptest"
	"testing"
)

func Test_httpWriteJson(t *testing.T) {
	w := httptest.NewRecorder()
	body := `{"foo":"bar"}`
	httpWriteJson(w, []byte(body))
	assert.Equal(t, w.Code, 200)
	assert.Equal(t, w.Header().Get("content-type"), "application/json; charset=UTF-8")
	assert.Equal(t, w.Body.String(), body)
}

func Test_httpWriteErrorJson(t *testing.T) {
	errCode := []int{404, 500}
	msg := "message"
	raw := "raw"
	for _, code := range errCode {
		w := httptest.NewRecorder()
		httpWriteErrorJson(w, code, msg, raw)
		assert.Equal(t, code, w.Code)
		assert.Equal(t, w.Body.String(), `{"msg":"message","raw":"raw"}`)
	}
}
