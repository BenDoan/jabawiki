package main

import (
	"compress/gzip"
	"crypto/md5"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/BurntSushi/toml"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/op/go-logging"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const DEFAULT_CONFIG_FILE_NAME = "config.toml"

var configFilePath = flag.String("config-file", DEFAULT_CONFIG_FILE_NAME, "A toml formatted config file")

type Config struct {
	Domain       string
	Port         int
	EnableSSL    bool
	LogLevel     string
	DataDir      string
	CookieSecret string
}

const (
	INTERNAL_SERVER_ERROR_MSG = "Internal server error"
)

var (
	listen = ":8080"

	baseTemplate = ""
	users        = map[string]User{}

	log       = logging.MustGetLogger("wiki")
	logFormat = logging.MustStringFormatter("%{color}%{shortfile} %{time:2006-01-02 15:04:05} %{level:.4s}%{color:reset} %{message}")

	store        *sessions.CookieStore
	articleStore ArticleStore
	conf         Config

	exePath, _ = filepath.Abs(filepath.Dir(os.Args[0]))
)

const (
	Admin = 1 << iota
	Verified
	Unverified
)

type User struct {
	Id, Email, Name string
	Role            int
	Password        []byte `json:"-"` // don't add password to json output
}

type IncomingArticle struct {
	Title, Body, Permission, Summary string
}

type OutgoingArticle struct {
	Title, Body, Permission string
}

type IncomingUser struct {
	Email, Name, Password string
}

func BaseHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, baseTemplate)
}

func HandleArticle(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	user, _ := getUserFromSession(r)

	vars := mux.Vars(r)
	title := vars["title"]
	article, err := articleStore.GetArticle(title)
	hasArticle := true

	if err != nil {
		log.Debug("Couldn't find article: %v", err)
		hasArticle = false
	}

	if !hasArticle || isUserAllowed(user, article.Metadata) {
		switch r.Method {
		case "GET":
			GetArticle(w, r, title)
			return
		case "PUT":
			UpdateArticle(w, r, title)
			return
		}
	}

	http.Error(w, "Not allowed", http.StatusUnauthorized)
	return
}

func GetArticle(w http.ResponseWriter, r *http.Request, title string) {
	format := r.Form.Get("format")

	article, err := articleStore.GetArticle(title)
	if err != nil {
		log.Debug("Couldn't find article: %v", err)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	outgoingArticle := OutgoingArticle{Title: title, Permission: article.Metadata.Permission}

	switch format {
	case "markdown":
		outgoingArticle.Body = article.Body
	case "html":
		outgoingArticle.Body = article.GetMarkdownBody()
	default:
		msg := "Invalid article format"
		log.Debug("%s: %v", msg, format)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	json_resp, err := json.Marshal(outgoingArticle)
	if err != nil {
		log.Debug("Couldn't marshal json response: %v", err)
		http.Error(w, INTERNAL_SERVER_ERROR_MSG, http.StatusInternalServerError)
		return
	}

	fmt.Fprint(w, string(json_resp))
}

func UpdateArticle(w http.ResponseWriter, r *http.Request, title string) {
	decoder := json.NewDecoder(r.Body)
	var article IncomingArticle
	err := decoder.Decode(&article)

	if err != nil {
		msg := "Couldn't decode incoming article"
		log.Debug("%s: %v", msg, err)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	// write article
	articlePath := AbsPathFromExe(getDataDirPath(), "articles", article.Title+".txt")
	err = ioutil.WriteFile(articlePath, []byte(article.Body), 0644)

	if err != nil {
		log.Error("Error saving article: %s", err)
		http.Error(w, INTERNAL_SERVER_ERROR_MSG, http.StatusInternalServerError)
		return
	}

	err = writeMetadata(article)
	if err != nil {
		log.Error("Error writing metadata: %s", err)
		http.Error(w, INTERNAL_SERVER_ERROR_MSG, http.StatusInternalServerError)
		return
	}

	articleStore.AddAvailableArticle(article.Title)
	articleStore.AddArticleFromIncoming(article.Title, article)

	creator := ""
	user, err := getUserFromSession(r)
	if err == nil {
		creator = user.Name
	} else {
		creator = r.RemoteAddr
	}

	err = writeHistory(article, creator)
	if err != nil {
		log.Error("Error writing history: %s", err)
		http.Error(w, INTERNAL_SERVER_ERROR_MSG, http.StatusInternalServerError)
	}

	err = archiveArticle(article)
	if err != nil {
		log.Error("Error archiving article: %s", err)
		http.Error(w, INTERNAL_SERVER_ERROR_MSG, http.StatusInternalServerError)
	}
}

func writeMetadata(article IncomingArticle) error {
	metadataString := fmt.Sprintf("%s\n%s\n%s", article.Permission, "", "")
	metadataFilePath := AbsPathFromExe(getDataDirPath(), "metadata", fmt.Sprintf("%s.meta", article.Title))

	err := ioutil.WriteFile(metadataFilePath, []byte(metadataString), 0644)

	if err != nil {
		return err
	}
	return nil
}

func archiveArticle(article IncomingArticle) error {
	archiveFilePath := AbsPathFromExe(getDataDirPath(), "archive", fmt.Sprintf("%s.%d.txt.gz", article.Title, time.Now().Unix()))
	archiveFile, err := os.OpenFile(archiveFilePath, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0660)

	gzipWriter, err := gzip.NewWriterLevel(archiveFile, gzip.DefaultCompression)
	defer gzipWriter.Close()

	if err != nil {
		return err
	}

	gzipWriter.Write([]byte(article.Body))
	gzipWriter.Flush()

	return nil
}

func writeHistory(article IncomingArticle, creator string) error {
	historyFilePath := AbsPathFromExe(getDataDirPath(), "history", article.Title+".hist")
	historyFile, err := os.OpenFile(historyFilePath, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0660)

	if err != nil {
		return err
	}

	history := fmt.Sprintf("%d | %s | %s\n", time.Now().Unix(), creator, article.Summary)
	fmt.Fprint(historyFile, history)

	return nil
}

func HandleGetAllArticleNames(w http.ResponseWriter, r *http.Request) {
	files, err := ioutil.ReadDir(AbsPathFromExe(getDataDirPath(), "articles"))

	if err != nil {
		log.Error("Couldn't get articles", err)
		http.Error(w, INTERNAL_SERVER_ERROR_MSG, http.StatusInternalServerError)
	}

	var articleNames = []string{}
	for _, file := range files {
		if !strings.HasPrefix(file.Name(), ".") {
			articleNames = append(articleNames, file.Name()[0:len(file.Name())-4])
		}
	}

	articlesJson, err := json.Marshal(articleNames)
	if err != nil {
		log.Error("Couldn't marshal article list to json: %v", err)
		http.Error(w, INTERNAL_SERVER_ERROR_MSG, http.StatusInternalServerError)
	}

	fmt.Fprint(w, string(articlesJson))
}

func HandleGetPreview(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var article IncomingArticle
	err := decoder.Decode(&article)

	if err != nil {
		log.Debug("Couldn't decode incoming article: %v", err)
		http.Error(w, "Couldn't decode incoming article", http.StatusBadRequest)
		return
	}

	processedMarkdown := processMarkdown([]byte(article.Body))
	safeHtml := renderMarkdown(processedMarkdown)

	outArticle := Article{Title: article.Title, Body: string(safeHtml)}

	articlesJson, err := json.Marshal(outArticle)
	if err != nil {
		log.Error("Couldn't marshal article list to json: %v", err)
		http.Error(w, INTERNAL_SERVER_ERROR_MSG, http.StatusInternalServerError)
	}

	fmt.Fprint(w, string(articlesJson))
}

func CanAccessTitle(title string, r *http.Request) (bool, error) {
	user, err := getUserFromSession(r)

	if err != nil {
		return false, err
	}

	metadata, err := GetMetadata(title)
	if err != nil {
		return false, err
	}

	return isUserAllowed(user, metadata), nil
}

// HandleHistoryGet returns the full history of a specific page
func HandleHistoryGet(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	title := vars["title"]

	canAccess, err := CanAccessTitle(title, r)

	if err != nil {
		msg := "User not authorized"
		log.Debug("%s: %v", msg, err)
		http.Error(w, msg, http.StatusUnauthorized)
	}

	if !canAccess {
		log.Debug("User is not allowed to access page: %v", err)
		http.Error(w, "Not allowed", http.StatusUnauthorized)
		return
	}

	histfileName := fmt.Sprintf("%s.hist", title)
	hist, err := ioutil.ReadFile(AbsPathFromExe(getDataDirPath(), "history", histfileName))
	if err != nil {
		msg := "Couldn't find article history"
		log.Debug("%s: %v", msg, err)
		http.Error(w, msg, http.StatusNotFound)
		return
	}

	var histItems = []map[string]interface{}{}

	historyByLine := strings.Split(string(hist), "\n")
	for i := len(historyByLine) - 1; i >= 0; i-- {
		splitLine := strings.Split(historyByLine[i], " | ")

		if len(splitLine) != 3 {
			continue
		}

		timeCol := splitLine[0]
		ipCol := splitLine[1]
		summaryCol := splitLine[2]

		timeInt, err := strconv.ParseInt(string(timeCol), 10, 64)
		if err != nil {
			msg := "Couldn't parse time from history entry"
			log.Error("%s: %v", msg, err)
		}

		histItems = append(histItems, map[string]interface{}{
			"time":    timeInt,
			"ip":      ipCol,
			"summary": summaryCol,
		})
	}

	json_resp, err := json.Marshal(histItems)
	if err != nil {
		log.Error("Unable to marshal history to json: %v", err)
		http.Error(w, INTERNAL_SERVER_ERROR_MSG, http.StatusInternalServerError)
		return
	}

	fmt.Fprint(w, string(json_resp))
}

func HandleArchiveGet(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	title := vars["title"]
	time := vars["archiveTime"]
	format := vars["format"]

	canAccess, err := CanAccessTitle(title, r)

	if err != nil {
		msg := "User not authorized"
		log.Debug("%s: %v", msg, err)
		http.Error(w, msg, http.StatusUnauthorized)
	}

	if !canAccess {
		log.Debug("User is not allowed to access page: %v", err)
		http.Error(w, "Not allowed", http.StatusUnauthorized)
		return
	}

	archiveFilename := fmt.Sprintf("%s.%s.txt.gz", title, time)
	f, err := os.Open(AbsPathFromExe(getDataDirPath(), "archive", archiveFilename))

	if err != nil {
		msg := "Couldn't find article archive"
		log.Debug("%s: %v", msg, err)
		http.Error(w, msg, http.StatusNotFound)
		return
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		log.Error("Couldn't create gzip reader from file, file may be corrupt: %v", err)
		http.Error(w, INTERNAL_SERVER_ERROR_MSG, http.StatusInternalServerError)
		return
	}
	defer gr.Close()

	b, err := ioutil.ReadAll(gr)
	if err != nil {
		log.Error("Couldn't read gzipped archive file, file may be corrupt: %v", err)
		http.Error(w, INTERNAL_SERVER_ERROR_MSG, http.StatusInternalServerError)
		return
	}

	switch format {
	case "markdown":
		fmt.Fprint(w, string(b))
	case "html":
		fmt.Fprint(w, Markdownify(string(b)))
	default:
		msg := "Invalid article format"
		log.Debug("%s: %v", msg, format)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}
}

func ComputeMd5(filePath string) ([]byte, error) {
	var result []byte
	file, err := os.Open(filePath)
	if err != nil {
		return result, err
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return result, err
	}

	return hash.Sum(result), nil
}

func HandleUploadImage(w http.ResponseWriter, r *http.Request) {
	file, header, err := r.FormFile("file")
	if err != nil {
		msg := "Didn't receive file"
		log.Info("%s: %v", msg, err)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}
	defer file.Close()

	filename := fmt.Sprintf("%d-%s", time.Now().Unix(), header.Filename)
	out, err := os.Create(AbsPathFromExe(getDataDirPath(), "images", filename))
	if err != nil {
		log.Error("Couldn't create file: %v", err)
		http.Error(w, INTERNAL_SERVER_ERROR_MSG, http.StatusInternalServerError)
		return
	}
	defer out.Close()

	_, err = io.Copy(out, file)
	if err != nil {
		log.Error("Couldn't copy file to filesystem: %v", err)
		http.Error(w, INTERNAL_SERVER_ERROR_MSG, http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, header.Filename)
}

func loadConfigFile() string {
	finalPath := *configFilePath
	if *configFilePath == DEFAULT_CONFIG_FILE_NAME {
		if _, err := os.Stat(*configFilePath); err == nil {
			finalPath = *configFilePath
		} else if _, err := os.Stat(AbsPathFromExe(*configFilePath)); err == nil {
			finalPath = AbsPathFromExe(*configFilePath)
		}
	}

	configData, err := ioutil.ReadFile(filepath.FromSlash(finalPath))
	if err != nil {
		panic(fmt.Sprintf("Error reading config file: %v", err))
	}

	if _, err := toml.Decode(string(configData), &conf); err != nil {
		panic(fmt.Sprintf("Error parsing config file: %v", err))
	}

	return finalPath
}

func setupLogging() {
	log_level, err := logging.LogLevel(conf.LogLevel)
	if err != nil {
		panic(err.Error())
	}

	logging.SetFormatter(logFormat)

	log_backend := logging.NewLogBackend(os.Stdout, "", 0)
	log_backend.Color = true

	log_backend_level := logging.AddModuleLevel(log_backend)
	log_backend_level.SetLevel(log_level, "")

	log.SetBackend(log_backend_level)
}

func init() {
	flag.Parse()

	confFilePath := loadConfigFile()

	if len(conf.CookieSecret) == 0 {
		panic("CookieSecret not set in config")
	}
	store = sessions.NewCookieStore([]byte(conf.CookieSecret))

	setupLogging()

	log.Notice("Using config file: %s", confFilePath)

	// load base template
	baseTemplateBytes, err := ioutil.ReadFile(AbsPathFromExe("templates", "base.html"))
	if err != nil {
		log.Fatal("Error reading base template: %v", err)
		panic(err)
	}
	baseTemplate = string(baseTemplateBytes)

	// populate articles cache
	articleStore = NewArticleStore()
	articleDir, err := ioutil.ReadDir(AbsPathFromExe(getDataDirPath(), "articles"))

	if err != nil {
		log.Fatal("Error reading articles: %v", err)
		panic(err)
	}

	numArticles := 0
	for _, file := range articleDir {
		if !file.IsDir() {
			articleName := strings.Split(file.Name(), ".")[0]
			articleStore.AddAvailableArticle(articleName)

			numArticles++
		}
	}
	log.Debug("Found %d available articles", numArticles)

	// populate users cache
	usersFilePath := AbsPathFromExe(getDataDirPath(), "users.txt")
	csvfile, err := os.Open(usersFilePath)

	if err != nil {
		if _, err := os.Stat(usersFilePath); err != nil {
			csvfile, _ = os.Create(usersFilePath)
		} else {
			log.Fatal("Error opening users file: %v", err)
			panic(err)
		}
	}
	defer csvfile.Close()

	reader := csv.NewReader(csvfile)
	reader.FieldsPerRecord = -1

	for {
		user, err := reader.Read()

		if err == io.EOF {
			break
		}

		if err != nil {
			log.Fatal("Error reading users file: %v", err)
			panic(err)
		}

		if len(user) == 5 {
			role, err := strconv.Atoi(user[3])
			if err != nil {
				panic(err)
			}

			u := User{user[0],
				user[1],
				user[2],
				role,
				[]byte(user[4])}

			users[user[0]] = u
			users[user[1]] = u
		} else {
			log.Error("Invalid row in csv file: %v", user)
		}
	}
	log.Debug("Loaded %d users", len(users)/2)
}

func redirToHttps(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "https://"+conf.Domain, 301)
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	r := mux.NewRouter()
	r.HandleFunc("/", BaseHandler)
	r.HandleFunc("/article/{title}", HandleArticle)

	r.HandleFunc("/articles/all", HandleGetAllArticleNames)
	r.HandleFunc("/articles/preview", HandleGetPreview)

	r.HandleFunc("/user/register", HandleRegister)
	r.HandleFunc("/user/login", HandleLogin)
	r.HandleFunc("/user/logout", HandleLogout)
	r.HandleFunc("/user/get", HandleUserGet)

	r.HandleFunc("/image/upload", HandleUploadImage)

	r.HandleFunc("/history/get/{title}", HandleHistoryGet)

	r.HandleFunc("/archives/get/{title}/{archiveTime}/{format}", HandleArchiveGet)

	r.PathPrefix("/images/").Handler(http.StripPrefix("/images/", http.FileServer(http.Dir(AbsPathFromExe(getDataDirPath(), "images")))))

	r.PathPrefix("/static/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, AbsPathFromExe(r.URL.Path[1:]))
	})

	r.PathPrefix("/partials/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, AbsPathFromExe(r.URL.Path[1:]))
	})

	r.PathPrefix("/").HandlerFunc(BaseHandler)

	http.Handle("/", r)

	if conf.EnableSSL {
		go func() {
			log.Notice("Listening on :443")
			httpsAddress := fmt.Sprintf("%s:%d", conf.Domain, 443)
			err := http.ListenAndServeTLS(httpsAddress, "cert.pem", "key.pem", nil)
			if err != nil {
				panic(fmt.Sprintf("Failed to start server: %v", err))
			}
		}()

		log.Notice("Listening on :%d", conf.Port)
		httpAddress := fmt.Sprintf("%s:%d", conf.Domain, conf.Port)
		err := http.ListenAndServe(httpAddress, http.HandlerFunc(redirToHttps))
		if err != nil {
			panic(fmt.Sprintf("Failed to start server: %v", err))
		}
	} else {
		log.Notice("Listening on :%d", conf.Port)
		err := http.ListenAndServe(fmt.Sprintf(":%d", conf.Port), r)
		if err != nil {
			panic(fmt.Sprintf("Failed to start server: %v", err))
		}
	}
}
