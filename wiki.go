package main

import (
	"encoding/json"
	"fmt"
	"github.com/microcosm-cc/bluemonday"
	"github.com/op/go-logging"
	"github.com/russross/blackfriday"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
)

const (
	DATA_DIR = "data"
)

var (
	listen = ":8080"

	templates = template.Must(template.ParseFiles("templates/base.html"))
	articles  = map[string]bool{}

	log    = logging.MustGetLogger("wiki")
	format = logging.MustStringFormatter("%{color}%{shortfile} %{time:15:04:05} %{level:.4s}%{color:reset} %{message}")
)

func angularHandler(w http.ResponseWriter, r *http.Request) {
	err := templates.ExecuteTemplate(w, "base.html", nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func HandleArticle(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	switch r.Method {
	case "GET":
		GetArticle(w, r)
	case "PUT":
		CreateArticle(w, r)
	}
}

func GetArticle(w http.ResponseWriter, r *http.Request) {
	title := r.Form.Get("title")
	format := r.Form.Get("format")

	body, err := ioutil.ReadFile("data/" + title)
	if err != nil {
		log.Info("Could not find requested article: '%s'", title)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	switch format {
	case "markdown":
		fmt.Fprintf(w, string(body))
	case "html":
		processedBody := processMarkdown(body)
		safe := renderMarkdown(processedBody)
		fmt.Fprintf(w, string(safe))
	default:
		log.Info("Invalid format type requested: '%s'", format)
		http.Error(w, err.Error(), 400)
		return
	}
}

func renderMarkdown(body []byte) []byte {
	unsafe := blackfriday.MarkdownCommon(body)

	policy := bluemonday.UGCPolicy()
	policy.AllowAttrs("class").OnElements("a")

	safe := policy.SanitizeBytes(unsafe)

	return safe
}

func processMarkdown(text []byte) []byte {
	// create wiki links
	rp := regexp.MustCompile(`\[\[([a-zA-z0-9_]+)\]\]`)
	body_s := rp.ReplaceAllStringFunc(string(text), func(str string) (link string) {
		articleName := str[2 : len(str)-2]
		if articles[articleName] {
			link = fmt.Sprintf(`<a href="/%s">%s</a>`, articleName, articleName)
		} else {
			link = fmt.Sprintf(`<a class="wikilink-new" href="/%s">%s</a>`, articleName, articleName)
		}
		return link
	})

	return []byte(body_s)
}

type Article struct {
	Title string
	Body  string
}

func CreateArticle(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var article Article
	err := decoder.Decode(&article)

	if err != nil {
		log.Info("Couldn't parse article for saving")
		http.Error(w, err.Error(), 400)
		return
	}

	err = ioutil.WriteFile("data/"+article.Title, []byte(article.Body), 0644)

	if err != nil {
		log.Error("Error saving file: %s", err)
		http.Error(w, err.Error(), 500)
		return
	}

	articles[article.Title] = true
}

func init() {
	backend := logging.NewLogBackend(os.Stderr, "", 0)
	backendFormatter := logging.NewBackendFormatter(backend, format)
	logging.SetBackend(backendFormatter)

	article_files, err := ioutil.ReadDir(DATA_DIR)

	if err != nil {
		log.Error("Error reading articles: %v", err)
		return
	}

	for _, file := range article_files {
		if !file.IsDir() {
			articles[file.Name()] = true
		}
	}
}

func main() {
	http.HandleFunc("/", angularHandler)
	http.HandleFunc("/article", HandleArticle)

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static/"))))
	http.Handle("/partials/", http.StripPrefix("/partials/", http.FileServer(http.Dir("./partials/"))))

	log.Notice("Listening on %s", listen)
	http.ListenAndServe(listen, nil)
}
