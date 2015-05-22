package main

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

var (
	testArticle = &Article{"testarticle", "This is the body"}
)

func TestAddArticle(t *testing.T) {
	articleJson, _ := json.Marshal(testArticle)
	reader := bytes.NewReader(articleJson)

	req, _ := http.NewRequest("PUT", "/article", reader)
	resp := httptest.NewRecorder()

	HandleArticle(resp, req)

	if resp.Code != http.StatusOK {
		t.Errorf("Article not added")
	}
}

func TestGetArticleMarkdown(t *testing.T) {
	TestAddArticle(t)

	url := fmt.Sprintf("/article?title=%s&format=markdown", testArticle.Title)
	req, _ := http.NewRequest("GET", url, nil)
	resp := httptest.NewRecorder()

	HandleArticle(resp, req)

	if resp.Code != http.StatusOK {
		t.Errorf("Article not found")
	}

	b := fmt.Sprintf("%s", resp.Body)

	if b != testArticle.Body {
		t.Errorf("Wrong output")
	}
}

func TestGetArticleHtml(t *testing.T) {
	TestAddArticle(t)

	url := fmt.Sprintf("/article?title=%s&format=html", testArticle.Title)
	req, _ := http.NewRequest("GET", url, nil)
	resp := httptest.NewRecorder()

	HandleArticle(resp, req)

	if resp.Code != http.StatusOK {
		t.Errorf("Article not found")
	}

	b := fmt.Sprintf("%s", resp.Body)
	expectedOutput := fmt.Sprintf("<p>%s</p>\n", testArticle.Body)
	if b != expectedOutput {
		t.Errorf("Wrong output")
	}
}

func TestGetArticleMissing(t *testing.T) {
	TestAddArticle(t)

	req, _ := http.NewRequest("GET", "/article?title=ello&format=markdown", nil)
	resp := httptest.NewRecorder()

	HandleArticle(resp, req)

	if resp.Code != http.StatusNotFound {
		t.Errorf("Failed to return error code")
	}
}

func GetRandomString() string {
	size := 25
	dict := "abcdefghijklmnopqrstuvwxyz"

	bytes := make([]byte, size)
	rand.Read(bytes)

	for k, v := range bytes {
		bytes[k] = dict[v%byte(len(dict))]
	}

	return string(bytes)
}

func BenchmarkCreation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		json := fmt.Sprintf(`{"title": "test%s", "body": "this is body"}`, GetRandomString())
		reader := strings.NewReader(json)

		req, _ := http.NewRequest("PUT", "/article", reader)
		resp := httptest.NewRecorder()

		HandleArticle(resp, req)
	}
}
