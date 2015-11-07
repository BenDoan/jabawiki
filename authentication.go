package main

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
)

var (
	errUserNotFound = errors.New("User not found in session")
)

func isUserAllowed(user User, articleMetadata ArticleMetadata) bool {
	hasUser := !reflect.DeepEqual(user, User{})
	hasArticle := !reflect.DeepEqual(articleMetadata, ArticleMetadata{})

	return hasUser && user.Role == Admin ||
		hasArticle && articleMetadata.Permission == "public" ||
		hasUser && hasArticle && articleMetadata.Permission == "registered" && user.Role == Verified
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

	usersFile, err := os.OpenFile(filepath.Join(getDataDirPath(), "users.txt"), os.O_RDWR|os.O_APPEND|os.O_CREATE, 0660)

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

	log.Debug("Registered user: %s", user.Email)
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
