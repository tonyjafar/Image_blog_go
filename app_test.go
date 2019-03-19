package main

import (
	"bytes"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestCheckFileName(t *testing.T) {
	if checkFileName("test") {
		t.Error("File name is incorrect, checkFileName should be false")
	}
	if !checkFileName("test.p") {
		t.Error("Should be true")
	}
}

func TestPassPolicy(t *testing.T) {
	myBoolUser, _ := passPolicy("admin", "test1!T")
	myBoolPass, _ := passPolicy("admin22", "test")
	if myBoolUser {
		t.Error("Should be False - User already There")
	}
	if myBoolPass {
		t.Error("Schould be False - Password is weak")
	}

}

func TestIsAdmin(t *testing.T) {
	if !isAdmin("admin") {
		t.Error("Should be true since admin user is admin")
	}
	if isAdmin("test") {
		t.Error("should be false")
	}
}

func TestGetAndUpdateRetry(t *testing.T) {
	myBoolFalse, _ := getAndUpdateRetry("admin")
	myBoolTrue, _ := getAndUpdateRetry("tester")
	if myBoolFalse {
		t.Error("user not blocked but get true")
	}
	if !myBoolTrue {
		t.Error("User is blocked but get false")
	}
	// TODO : Reset retries for DB Test users
}

func TestUpdateUserSession(t *testing.T) {
	err := updateUserSession("XXX", "admin")

	if err != nil {
		t.Error("Failed to update sesion")
	}
}

func TestVideosAdminEdit(t *testing.T) {
	cookieToSet := "XXX,admin"
	req, err := http.NewRequest("GET", "/edit-video", nil)
	if err != nil {
		t.Fatal(err.Error())
	}
	res := httptest.NewRecorder()

	handler := http.HandlerFunc(videosAdminEdit)
	c := &http.Cookie{
		Name:   "session",
		Value:  cookieToSet,
		MaxAge: cAge,
	}
	req.AddCookie(c)
	handler.ServeHTTP(res, req)
	if status := res.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}
	req2, err := http.NewRequest("GET", "/edit-video", nil)
	if err != nil {
		t.Fatal(err.Error())
	}
	handler.ServeHTTP(res, req2)
	if status := res.Code; status != http.StatusSeeOther {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusSeeOther)
	}
}

func newfileUploadRequest(uri string, params map[string]string, paramName, path string) (*http.Request, *multipart.Writer, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	fileContents, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, nil, err
	}
	fi, err := file.Stat()
	if err != nil {
		return nil, nil, err
	}
	file.Close()

	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile(paramName, fi.Name())
	if err != nil {
		return nil, nil, err
	}
	part.Write(fileContents)

	for key, val := range params {
		_ = writer.WriteField(key, val)
	}
	err = writer.Close()
	if err != nil {
		return nil, nil, err
	}
	req, err := http.NewRequest("POST", uri, body)
	return req, writer, err
}
func TestAddImage(t *testing.T) {
	cookieToSet := "XXX,admin"
	extraParams := map[string]string{
		"location":    "test",
		"description": "test upload",
	}
	req, writer, err := newfileUploadRequest("/add_image", extraParams, "nf", "/Users/tony/Desktop/test-product-test.png")
	if err != nil {
		t.Fatal(err.Error())
	}
	res := httptest.NewRecorder()

	handler := http.HandlerFunc(addImage)
	c := &http.Cookie{
		Name:   "session",
		Value:  cookieToSet,
		MaxAge: cAge,
	}
	req.AddCookie(c)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	handler.ServeHTTP(res, req)
	if status := res.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}
}
