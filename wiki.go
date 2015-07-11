package main

import (
	"bytes"
	"compress/gzip"
	"crypto/rand"
	"encoding/csv"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/BurntSushi/toml"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/microcosm-cc/bluemonday"
	"github.com/op/go-logging"
	"github.com/russross/blackfriday"
	"golang.org/x/crypto/bcrypt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

var configFile = flag.String("config-file", "config.toml", "A toml formatted config file")

type Config struct {
	Domain    string
	Port      int
	EnableSSL bool
	LogLevel  string
}

const (
	DATA_DIR                  = "data"
	INTERNAL_SERVER_ERROR_MSG = "Internal server error"
)

const (
	Admin = 1 << iota
	Verified
	Unverified
)

var (
	listen = ":8080"

	baseTemplate = ""
	articles     = map[string]bool{}
	users        = map[string]User{}

	log       = logging.MustGetLogger("wiki")
	logFormat = logging.MustStringFormatter("%{color}%{shortfile} %{time:2006-01-02 15:04:05} %{level:.4s}%{color:reset} %{message}")

	store = sessions.NewCookieStore([]byte("xxxxsecret"))

	conf Config

	errUserNotFound = errors.New("User not found in session")
)

type User struct {
	Id, Email, Name string
	Role            int
	Password        []byte `json:"-"`
}

type Article struct {
	Title, Body string
}

type WikiData struct {
	User    User
	Article Article
}

type IncomingArticle struct {
	Title   string
	Body    string
	Summary string
}

type IncomingUser struct {
	Email, Name, Password string
}

func BaseHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, baseTemplate)
}

func HandleArticle(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	session, err := store.Get(r, "user")

	if err != nil {
		log.Error("Session had error: %s", err)
		http.Error(w, INTERNAL_SERVER_ERROR_MSG, http.StatusInternalServerError)
		return
	}

	if data, ok := session.Values["id"]; ok {
		if userId, ok := data.(string); ok {
			if user, ok := users[userId]; ok {
				if isUserAllowed(user) {
					vars := mux.Vars(r)
					title := vars["title"]

					switch r.Method {
					case "GET":
						GetArticle(w, r, title, user)
						return
					case "PUT":
						UpdateArticle(w, r, title)
						return
					}
				}
			}
		}
	}

	log.Debug("Access not authorized")
	http.Error(w, "Not allowed", http.StatusUnauthorized)
	return
}

func isUserAllowed(user User) bool {
	return user.Role != Unverified
}

func GetArticle(w http.ResponseWriter, r *http.Request, title string, user User) {
	format := r.Form.Get("format")

	fileName := fmt.Sprintf("%s/articles/%s.txt", DATA_DIR, title)
	body, err := ioutil.ReadFile(fileName)
	if err != nil {
		log.Debug("Couldn't find article: %v", err)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	article := Article{Title: title}
	switch format {
	case "markdown":
		article.Body = string(body)
	case "html":
		processedMarkdown := processMarkdown(body)
		safeHtml := renderMarkdown(processedMarkdown)

		article.Body = string(safeHtml)
	default:
		msg := "Invalid article format"
		log.Debug("%s: %v", msg, format)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	json_resp, err := json.Marshal(article)
	if err != nil {
		log.Debug("Couldn't marshal json response: %v", err)
		http.Error(w, INTERNAL_SERVER_ERROR_MSG, http.StatusInternalServerError)
		return
	}

	fmt.Fprint(w, string(json_resp))
}

func processMarkdown(text []byte) []byte {
	// create wiki links
	//TODO: think about normalizing the input here
	pattern := regexp.MustCompile(`\[\[[a-zA-Z0-9_]+\]\]`)
	newBody := pattern.ReplaceAllStringFunc(string(text), func(str string) string {
		articleName := str[2 : len(str)-2] //remove brackets
		spacedArticleName := strings.Replace(articleName, "_", " ", -1)
		if articles[articleName] {
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
	fileName := fmt.Sprintf("%s/articles/%s.txt", DATA_DIR, article.Title)
	err = ioutil.WriteFile(fileName, []byte(article.Body), 0644)

	if err != nil {
		log.Error("Error saving article: %s", err)
		http.Error(w, INTERNAL_SERVER_ERROR_MSG, http.StatusInternalServerError)
		return
	}

	articles[article.Title] = true

	writeMetadata(w, r, article)

	archiveArticle(w, article)
}

func archiveArticle(w http.ResponseWriter, article IncomingArticle) {
	var b bytes.Buffer
	gzipWriter := gzip.NewWriter(&b)
	gzipWriter.Write([]byte(article.Body))
	gzipWriter.Close()

	fileName := fmt.Sprintf("%s/archive/%s.%d.txt.gz", DATA_DIR, article.Title, time.Now().Unix())
	err := ioutil.WriteFile(fileName, b.Bytes(), 0644)

	if err != nil {
		log.Error("Error saving archive: %s", err)
		http.Error(w, INTERNAL_SERVER_ERROR_MSG, http.StatusInternalServerError)
		return
	}
}

func writeMetadata(w http.ResponseWriter, r *http.Request, article IncomingArticle) {
	fileName := fmt.Sprintf("%s/metadata/%s.meta", DATA_DIR, article.Title)
	metadataFile, err := os.OpenFile(fileName, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0660)

	if err != nil {
		log.Error("Error saving metadata: %s", err)
		http.Error(w, INTERNAL_SERVER_ERROR_MSG, http.StatusInternalServerError)
		return
	}

	metadata := fmt.Sprintf("%d | %s | %s\n", time.Now().Unix(), r.RemoteAddr, article.Summary)
	fmt.Fprint(metadataFile, metadata)
}

func genUUID() string {
	uuid := make([]byte, 16)
	n, err := io.ReadFull(rand.Reader, uuid)

	if n != len(uuid) || err != nil {
		panic(fmt.Sprintf("Couldn't generate uuid %v", err))
	}

	uuid[8] = uuid[8]&^0xc0 | 0x80
	uuid[6] = uuid[6]&^0xf0 | 0x40
	return fmt.Sprintf("%x-%x-%x-%x-%x", uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:])
}

func HandleRegister(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var incomingUser IncomingUser
	err := decoder.Decode(&incomingUser)

	if err != nil {
		msg := "Couldn't decode register request data"
		log.Debug("%s: %v", msg, err)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	if _, ok := users[incomingUser.Email]; ok {
		msg := "Couldn't create account, user already exists"
		log.Debug("%s: %s", msg, incomingUser.Email)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(incomingUser.Password), 10)

	if err != nil {
		log.Error("Couldn't generate password with bcrypt: %v", err)
		http.Error(w, INTERNAL_SERVER_ERROR_MSG, http.StatusInternalServerError)
	}

	usersFile, err := os.OpenFile(DATA_DIR+"/users.txt", os.O_RDWR|os.O_APPEND|os.O_CREATE, 0660)

	if err != nil {
		log.Error("Couldn't open users file: ", err)
		http.Error(w, INTERNAL_SERVER_ERROR_MSG, http.StatusInternalServerError)
		return
	}

	user := User{genUUID(), incomingUser.Email, incomingUser.Name, Unverified, hashedPassword}
	_, err = fmt.Fprintf(usersFile, fmt.Sprintf("%s,%s,%s,%d,%s\n", user.Id, user.Email, user.Name, user.Role, user.Password))
	if err != nil {
		log.Error("Couldn't write to users file: %v", err)
		http.Error(w, INTERNAL_SERVER_ERROR_MSG, http.StatusInternalServerError)
		return
	}

	// allow user to be looked up by id or email
	users[user.Id] = user
	users[user.Email] = user
	fmt.Fprint(w, "Good")
}

func HandleLogin(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)

	var incomingUser IncomingUser
	err := decoder.Decode(&incomingUser)

	if err != nil {
		msg := "Couldn't decode login request data"
		log.Debug("%s: %v", msg, err)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	if storedUser, ok := users[incomingUser.Email]; ok {
		if bcrypt.CompareHashAndPassword(storedUser.Password, []byte(incomingUser.Password)) == nil {
			// login user
			session, _ := store.Get(r, "user")
			session.Values["id"] = storedUser.Id
			session.Save(r, w)
			fmt.Fprint(w, "Good")
		} else {
			log.Debug("Invalid password during login")
			http.Error(w, "Invalid email or password", http.StatusBadRequest)
			return
		}
	} else {
		log.Debug("Invalid email during login")
		http.Error(w, "Invalid email or password", http.StatusBadRequest)
		return
	}
}

func HandleLogout(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "user")
	session.Values["id"] = -1
	session.Save(r, w)

	fmt.Fprint(w, "Good")
}

func getUserFromSession(r *http.Request) (User, error) {
	session, err := store.Get(r, "user")
	if err != nil {
		log.Debug("Couldn't find user: %v", err)
		return User{}, err
	}

	if data, ok := session.Values["id"]; ok {
		if userId, ok := data.(string); ok {
			if user, ok := users[userId]; ok {
				return user, nil
			}
		}
	}

	return User{}, errUserNotFound
}

func HandleUserGet(w http.ResponseWriter, r *http.Request) {
	user, err := getUserFromSession(r)
	if err != nil {
		msg := "Couldn't find user in session"
		log.Debug("%s: %v", msg, err)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	userJson, err := json.Marshal(user)
	if err != nil {
		log.Error("Couldn't marshal user json: %v", err)
		http.Error(w, INTERNAL_SERVER_ERROR_MSG, http.StatusInternalServerError)
		return
	}

	fmt.Fprint(w, string(userJson))
}

func HandleGetAllArticles(w http.ResponseWriter, r *http.Request) {
	files, err := ioutil.ReadDir(DATA_DIR + "/articles")

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

func init() {
	flag.Parse()

	// read config file
	configData, err := ioutil.ReadFile(*configFile)
	if err != nil {
		panic(fmt.Sprintf("Error reading config file: %v", err))
	}

	if _, err := toml.Decode(string(configData), &conf); err != nil {
		panic(fmt.Sprintf("Error parsing config file: %v", err))
	}

	// setup logging
	log_level, err := logging.LogLevel(conf.LogLevel)
	if err != nil {
		panic(err.Error())
	}

	backend := logging.NewLogBackend(os.Stderr, "", 0)
	backendFormatter := logging.NewBackendFormatter(backend, logFormat)

	backendLeveled := logging.AddModuleLevel(backend)
	backendLeveled.SetLevel(log_level, "")

	logging.SetBackend(backendLeveled, backendFormatter)

	// load base template
	baseTemplateBytes, err := ioutil.ReadFile("templates/base.html")
	if err != nil {
		log.Fatal("Error reading base template: %v", err)
		panic(err)
	}
	baseTemplate = string(baseTemplateBytes)

	// populate articles cache
	articleDir, err := ioutil.ReadDir(DATA_DIR + "/articles")

	if err != nil {
		log.Fatal("Error reading articles: %v", err)
		panic(err)
	}

	for _, file := range articleDir {
		if !file.IsDir() {
			articleName := strings.Split(file.Name(), ".")[0]
			articles[articleName] = true
		}
	}

	// populate users cache
	usersFileName := DATA_DIR + "/users.txt"
	csvfile, err := os.Open(usersFileName)

	if err != nil {
		if _, err := os.Stat(usersFileName); err != nil {
			csvfile, _ = os.Create(usersFileName)
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
}

func redirToHttps(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "https://"+conf.Domain, 301)
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	r := mux.NewRouter()
	r.HandleFunc("/", BaseHandler)
	r.HandleFunc("/article/{title}", HandleArticle)

	r.HandleFunc("/articles/all", HandleGetAllArticles)
	r.HandleFunc("/articles/preview", HandleGetPreview)

	r.HandleFunc("/user/register", HandleRegister)
	r.HandleFunc("/user/login", HandleLogin)
	r.HandleFunc("/user/logout", HandleLogout)
	r.HandleFunc("/user/get", HandleUserGet)

	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	r.PathPrefix("/partials/").Handler(http.StripPrefix("/partials/", http.FileServer(http.Dir("partials"))))

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
		err := http.ListenAndServe(fmt.Sprintf("%s:%d", conf.Domain, conf.Port), r)
		if err != nil {
			panic(fmt.Sprintf("Failed to start server: %v", err))
		}
	}
}
