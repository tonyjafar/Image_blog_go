package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
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
	req, writer, err := newfileUploadRequest(
		"/add_image", extraParams, "nf", "./test_image/test-product-test.png")
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
	var myImage string
	dbError := db.QueryRow("select name from image_blog.images where description=?", extraParams["description"]).Scan(&myImage)
	if myImage == "" || dbError != nil {
		if dbError != nil {
			fmt.Println(dbError.Error())
		}
		t.Error("Image not added to DB")
	}
	if status := res.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}
	req2, writer2, err2 := newfileUploadRequest("/add_image", extraParams, "nf", "./test_image/test.txt")
	if err2 != nil {
		t.Fatal(err2.Error())
	}
	req2.AddCookie(c)
	req2.Header.Set("Content-Type", writer2.FormDataContentType())
	handler.ServeHTTP(res, req2)
	if !Data.ErrorFile.IsError {
		t.Error("Should return true")
	}
}

func TestInfoQuery(t *testing.T) {
	cookieToSet := "XXX,admin"
	var myImage string
	db.QueryRow("select name from image_blog.images where description=\"test upload\"").Scan(&myImage)
	if myImage == "" {
		t.Error("Image not Found - Could not Continue")
	}
	req, err := http.NewRequest("GET", "/info?query={image(name:\""+myImage+"\"){name,description,created_at}}", nil)
	if err != nil {
		t.Fatal(err.Error())
	}
	res := httptest.NewRecorder()

	handler := http.HandlerFunc(getInfo)
	c := &http.Cookie{
		Name:   "session",
		Value:  cookieToSet,
		MaxAge: cAge,
	}
	req.AddCookie(c)
	handler.ServeHTTP(res, req)
	var responsJson struct {
		Data struct {
			Image struct {
				CreatedAt   string `json:"created_at"`
				Description string `json:"description"`
				Name        string `json:"name"`
			} `json:"image"`
		} `json:"data"`
	}
	fb, err := ioutil.ReadAll(res.Body)
	j := json.Unmarshal(fb, &responsJson)
	if j != nil {
		t.Error(j.Error())
	}
	if responsJson.Data.Image.Name != myImage {
		t.Errorf("Expect to get %s but get %s", myImage, responsJson.Data.Image.Name)
	}

}

func TestSearch(t *testing.T) {
	cookieToSet := "XXX,admin"
	form := url.Values{}
	form.Add("search_desc", "test upload")
	req, err := http.NewRequest("POST", "/search", strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatal(err.Error())
	}
	res := httptest.NewRecorder()

	handler := http.HandlerFunc(search)
	c := &http.Cookie{
		Name:   "session",
		Value:  cookieToSet,
		MaxAge: cAge,
	}
	req.AddCookie(c)
	req.Form = form
	handler.ServeHTTP(res, req)
	if status := res.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}
	if len(Data.List) != 1 {
		t.Errorf("List length should be 1 but get %v", len(Data.List))
	}
	var myImage string
	db.QueryRow("select name from image_blog.images where description=\"test upload\"").Scan(&myImage)
	if Data.List[0] != myImage {
		t.Errorf("getting %s but expected %s", Data.List[0], myImage)
	}
	if Data.MyVar.PageNumber != 1 {
		t.Errorf("got %v expected 1", Data.MyVar.PageNumber)
	}
	if Data.MyVar.Next || Data.MyVar.Prev {
		t.Errorf("Should both value return false, get %v %v", Data.MyVar.Next, Data.MyVar.Prev)
	}

}

func TestInfoDelete(t *testing.T) {
	cookieToSet := "XXX,admin"
	var myImage string
	db.QueryRow("select name from image_blog.images where description=\"test upload\"").Scan(&myImage)
	if myImage == "" {
		t.Error("Image not Found - Could not Continue")
	}
	req, err := http.NewRequest("GET", "/info?query=mutation+_{delete(name:\""+myImage+"\"){name,description,created_at}}", nil)
	if err != nil {
		t.Fatal(err.Error())
	}
	res := httptest.NewRecorder()

	handler := http.HandlerFunc(getInfo)
	c := &http.Cookie{
		Name:   "session",
		Value:  cookieToSet,
		MaxAge: cAge,
	}
	req.AddCookie(c)
	handler.ServeHTTP(res, req)
	var checkImage string
	db.QueryRow("select name from image_blog.images where description=\"test upload\"").Scan(&checkImage)
	if checkImage != "" {
		t.Error("Image not Deleted")
	}
	var responsJson struct {
		Data struct {
			Delete struct {
				CreatedAt   string `json:"created_at"`
				Description string `json:"description"`
				Name        string `json:"name"`
			} `json:"delete"`
		} `json:"data"`
	}
	fb, err := ioutil.ReadAll(res.Body)
	j := json.Unmarshal(fb, &responsJson)
	if j != nil {
		t.Error(j.Error())
	}
	if responsJson.Data.Delete.Name != myImage {
		t.Errorf("Expect to delete %s but delete %s", myImage, responsJson.Data.Delete.Name)
	}
}

func TestUsersAdminAddLoginChange(t *testing.T) {
	cookieToSet := "XXX,admin"
	c := &http.Cookie{
		Name:   "session",
		Value:  cookieToSet,
		MaxAge: cAge,
	}

	formAdd := url.Values{}
	formAdd.Add("username", "test")
	formAdd.Add("password", "Test1!222")
	formAdd.Add("admin", "no")
	reqAdd, errAdd := http.NewRequest("POST", "/add-user", strings.NewReader(formAdd.Encode()))
	if errAdd != nil {
		t.Fatal(errAdd.Error())
	}
	resAdd := httptest.NewRecorder()

	handlerAdd := http.HandlerFunc(addUserAdmin)

	reqAdd.AddCookie(c)
	reqAdd.Form = formAdd
	handlerAdd.ServeHTTP(resAdd, reqAdd)

	if !Data.PassError.IsSucc {
		t.Error("User not added!")
	}
	var userAdd string
	dbErrAdd := db.QueryRow("select username from image_blog.Users where username = \"test\"").Scan(&userAdd)
	if dbErrAdd != nil {
		t.Errorf("Failed to get User from DB: %s", dbErrAdd.Error())
	}
	if userAdd != "test" {
		t.Errorf("Expect user to be test - but got: %s", userAdd)
	}
	formLog := url.Values{}
	formLog.Add("username", "test")
	formLog.Add("password", "Test1!222")
	reqLog, errLog := http.NewRequest("POST", "/signin", strings.NewReader(formLog.Encode()))
	if errLog != nil {
		t.Fatal(errLog.Error())
	}
	resLog := httptest.NewRecorder()
	handleLog := http.HandlerFunc(login)
	reqLog.Form = formLog
	handleLog.ServeHTTP(resLog, reqLog)
	if Data.Username != "test" {
		t.Errorf("Expected user test got %s", Data.Username)
	}
	var session string
	dbErrLog := db.QueryRow("select session from image_blog.Users where username = \"test\"").Scan(&session)
	if dbErrLog != nil {
		t.Errorf("Failed to get session value: %s", dbErrLog.Error())
	}
	if session == "" {
		t.Error("expexted to get value but got nil")
	}
	reqDel, errDel := http.NewRequest("GET", "/edit-user?delete=test", nil)
	if errDel != nil {
		t.Fatal(errDel.Error())
	}
	resDel := httptest.NewRecorder()
	handlerDel := http.HandlerFunc(usersAdminChange)

	reqDel.AddCookie(c)
	handlerDel.ServeHTTP(resDel, reqDel)
	var userDel string
	dbErrDel := db.QueryRow("select username from image_blog.Users where username = \"test\"").Scan(&userDel)
	if dbErrDel == nil {
		t.Errorf("Failed to delete User from DB: %s", dbErrDel.Error())
	}
	if userDel != "" {
		t.Errorf("Expect user to be empty - but got: %s", userDel)
	}

}

func TestSignout(t *testing.T) {
	cookieToSet := "XXX,admin"
	req, err := http.NewRequest("GET", "/signout", nil)
	if err != nil {
		t.Fatal(err.Error())
	}
	res := httptest.NewRecorder()

	handler := http.HandlerFunc(signout)
	c := &http.Cookie{
		Name:   "session",
		Value:  cookieToSet,
		MaxAge: cAge,
	}
	req.AddCookie(c)
	handler.ServeHTTP(res, req)
	var session string
	db.QueryRow("select session from image_blog.Users where username=\"admin\"").Scan(&session)
	if session != "" {
		t.Error("Session not Deleted")
	}
	newCookie, cookErr := req.Cookie("session")
	if cookErr != nil {
		t.Error("could not get cookie from request")
	}
	if newCookie.MaxAge > 1 {
		t.Error("session not expired")
	}
}

func TestLoginFailed(t *testing.T) {
	formLog := url.Values{}
	formLog.Add("username", "test")
	formLog.Add("password", "Test1!222rrrr")
	reqLog, errLog := http.NewRequest("POST", "/signin", strings.NewReader(formLog.Encode()))
	if errLog != nil {
		t.Fatal(errLog.Error())
	}
	resLog := httptest.NewRecorder()
	handleLog := http.HandlerFunc(login)
	reqLog.Form = formLog
	handleLog.ServeHTTP(resLog, reqLog)
	if Data.Username == "test" {
		t.Errorf("got %s", Data.Username)
	}
	var session string
	db.QueryRow("select session from image_blog.Users where username = \"test\"").Scan(&session)
	if session != "" {
		t.Error("expexted to get nil but got value")
	}
}
