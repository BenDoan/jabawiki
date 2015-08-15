package main

import (
	"fmt"
	"github.com/microcosm-cc/bluemonday"
	"github.com/russross/blackfriday"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"strings"
)

type ArticleMetadata struct {
	Permission string
}

type Article struct {
	Title, Body string
	Metadata    ArticleMetadata
}

type ArticleStore struct {
	availableArticles map[string]bool
	articles          map[string]Article
}

func NewArticleStore() ArticleStore {
	return ArticleStore{make(map[string]bool), make(map[string]Article)}
}

func (a ArticleStore) AddAvailableArticle(key string) {
	a.availableArticles[key] = true
}

func (a ArticleStore) IsArticleAvailable(key string) bool {
	_, ok := a.availableArticles[key]
	return ok
}

func (a ArticleStore) AddArticle(key string, article Article) {
	a.articles[key] = article
}

func (a ArticleStore) AddArticleFromIncoming(key string, incomingArticle IncomingArticle) {
	article := Article{Title: incomingArticle.Title, Body: incomingArticle.Body}

	a.articles[key] = article
}

func (a ArticleStore) GetArticle(title string) (Article, error) {
	if cachedArticle, ok := a.articles[title]; ok {
		return cachedArticle, nil
	}

	articlePath := filepath.Join(getDataDirPath(), "articles", title+".txt")
	body, err := ioutil.ReadFile(articlePath)

	article := Article{Title: title, Body: string(body)}
	a.articles[title] = article
	return article, err
}

func (a ArticleStore) HasArticle(key string) bool {
	_, ok := a.articles[key]
	return ok
}

func (a Article) GetMarkdownBody() string {
	processedMarkdown := processMarkdown([]byte(a.Body))
	safeHtml := renderMarkdown(processedMarkdown)
	return string(safeHtml)
}

func processMarkdown(text []byte) []byte {
	// create wiki links
	//TODO: think about normalizing the input here
	pattern := regexp.MustCompile(`\[\[[a-zA-Z0-9_]+\]\]`)
	newBody := pattern.ReplaceAllStringFunc(string(text), func(str string) string {
		articleName := str[2 : len(str)-2] //remove brackets
		spacedArticleName := strings.Replace(articleName, "_", " ", -1)
		if articleStore.IsArticleAvailable(articleName) {
			return fmt.Sprintf(`<a href="/w/%s">%s</a>`, articleName, spacedArticleName)
		} else {
			return fmt.Sprintf(`<a class="wikilink-new" href="/w/%s">%s</a>`, articleName, spacedArticleName)
		}
	})

	return []byte(newBody)
}

func renderMarkdown(body []byte) []byte {
	htmlFlags := 0 |
		blackfriday.HTML_USE_SMARTYPANTS |
		//blackfriday.HTML_SMARTYPANTS_FRACTIONS |
		//TODO: need to add class to generated html
		//blackfriday.HTML_TOC |
		blackfriday.HTML_SMARTYPANTS_LATEX_DASHES

	extensions := 0 |
		blackfriday.EXTENSION_NO_INTRA_EMPHASIS |
		blackfriday.EXTENSION_TABLES |
		blackfriday.EXTENSION_FENCED_CODE |
		blackfriday.EXTENSION_AUTOLINK |
		blackfriday.EXTENSION_STRIKETHROUGH |
		blackfriday.EXTENSION_HEADER_IDS |
		blackfriday.EXTENSION_AUTO_HEADER_IDS |
		blackfriday.EXTENSION_TITLEBLOCK |
		//blackfriday.EXTENSION_SPACE_HEADERS |
		blackfriday.EXTENSION_BACKSLASH_LINE_BREAK

	renderer := blackfriday.HtmlRenderer(htmlFlags, "", "")

	unsafe := blackfriday.MarkdownOptions(body, renderer, blackfriday.Options{
		Extensions: extensions})

	policy := bluemonday.UGCPolicy()
	policy.AllowAttrs("class").OnElements("a")

	safe := policy.SanitizeBytes(unsafe)

	return safe
}
