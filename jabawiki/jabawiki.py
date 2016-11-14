import argparse
import crypt
import datetime
import json
import sqlite3
import time
import os

from collections import namedtuple
from os import path

import lib.bottle as bottle

from lib.bottle import (
    abort,
    default_app,
    hook,
    request,
    response,
    route,
    run,
    static_file,
    template,
)

from util import namedtuple_factory

SCRIPT_PATH = os.path.dirname(os.path.realpath(__file__))
PORT = 8080
COOKIE_SECRET = "changeme1"

try:
    os.mkdir("/tmp/jabawiki")
except:
    pass

conn = sqlite3.connect('/tmp/jabawiki/jabawiki.db', detect_types=sqlite3.PARSE_DECLTYPES)

conn.row_factory = namedtuple_factory
c = conn.cursor()

# Routes
###################################
@route('/')
def index():
    with open(path.join(SCRIPT_PATH, "templates", "base.html")) as f:
        return "".join(f.readlines())

@route('/w/<rest:re:.+>')
def wiki(rest):
    with open(path.join(SCRIPT_PATH, "templates", "base.html")) as f:
        return "".join(f.readlines())

@route('/article/get/<name>')
def get_article(name):
    if not can_access(name):
        response.status = 401
        return "Not authorized"

    article = get_article_by_title(name)
    if article is None:
        response.status = 404
        return "Article missing"

    article_data = {
        "Title": article.title,
        "Body": article.text,
        "Permission": article.publicity
    }

    return json.dumps(article_data)

@route('/article/put/<name>', 'PUT')
def put_article(name):
    if not can_access(name):
        response.status = 401
        return "Not authorized"

    body = request.json
    article = get_article_by_title(name)

    user_id = get_cookie_user_id()
    if user_id is None:
        response.status = 401
        return "Not authorized"


    if article is None:
        create_article(body['title'], body['body'], body['permission'], body['summary'])
    else:
        update_article(body['title'], body['body'], body['permission'], body['summary'])

    update_history(body['title'], body['summary'])

    return "Good"

@route('/history/get/<name>', 'GET')
def get_history(name):
    if not can_access(name):
        response.status = 401
        return "Not authorized"

    history_items = get_history_by_article(name)

    ret_items = []
    for item in reversed(history_items):
        ret_items.append({
            "time": time.mktime(item.time.timetuple()),
            "user": get_user_by_id(item.user_id).name,
            "summary": item.summary
        })

    return json.dumps(ret_items)

# Auth
@route('/user/login', 'POST')
def user_login():
    body = request.json
    email = body["email"]
    password = body["password"]
    user = get_user_by_email(email)
    if user is not None:
        response.set_cookie("auth", user.id, secret=COOKIE_SECRET, path="/")
    else:
        response.status = 400
        return "Invalid login"

@route('/user/get', 'GET')
def user_get():
    user = get_user_by_id(get_cookie_user_id())
    if user is None:
        response.status = 401
        return "Not logged in"

    user_data = {
        "Name": user.name,
        "Email": user.email,
        "Role": user.role,
    }
    return json.dumps(user_data)

@route('/user/logout', 'POST')
def user_logout():
    response.delete_cookie("auth", path="/")
    return "Good"

@route('/user/register', 'POST')
def user_register():
    body = request.json
    email = body["email"]
    name = body["name"]
    password = body["password"]

    create_user(email, name, password)
    return "Good"

# Files
###################################
@route('/static/<pth:path>')
def static(pth):
    return static_file(pth, root=path.join(SCRIPT_PATH, "static"))

@route('/partials/<pth:path>')
def partials(pth):
    return static_file(pth, root=path.join(SCRIPT_PATH, "partials"))

# Auth
###################################
def create_hash_password(plain_pass):
    return crypt.crypt(plain_pass, crypt.mksalt())

def is_matching_pass(plain_pass, crypted_pass):
    return crypt.crypt(plain_pass, crypted_pass) == crypted_pass


# Model
###################################
def create_tables():
    c.execute("""CREATE TABLE IF NOT EXISTS user(
                 id INTEGER PRIMARY KEY,
                 name TEXT NOT NULL,
                 email TEXT NOT NULL,
                 role TEXT NOT NULL,
                 password TEXT NOT NULL,
                 creation_date TIMESTAMP NOT NULL,
                 UNIQUE(email))""")

    c.execute("""CREATE TABLE IF NOT EXISTS article(
                 id INTEGER PRIMARY KEY,
                 title TEXT NOT NULL,
                 text TEXT NOT NULL,
                 publicity TEXT NOT NULL,
                 creation_date TIMESTAMP NOT NULL,
                 UNIQUE (title))""")

    c.execute("""CREATE TABLE IF NOT EXISTS article_history(
                 id INTEGER PRIMARY KEY,
                 article_id INTEGER NOT NULL,
                 user_id INTEGER NOT NULL,
                 summary TEXT NOT NULL,
                 time TIMESTAMP NOT NULL,
                 FOREIGN KEY(article_id) REFERENCES article(id),
                 FOREIGN KEY(user_id) REFERENCES user(id))""")
    conn.commit()

# user
def get_user_by_email(email):
    c.execute("""SELECT * from user
                 WHERE user.email=?""", (email,))
    return c.fetchone()

def get_user_by_id(uid):
    c.execute("""SELECT * from user
                 WHERE user.id=?""", (uid,))
    return c.fetchone()

def create_user(email, name, password):
    user_data = (name, email, "user", create_hash_password(password), datetime.datetime.utcnow())
    c.execute("""INSERT INTO user
                 (name, email, role, password, creation_date)
                 VALUES (?, ?, ?, ?, ?)""", user_data)
    conn.commit()

# article
def get_article_by_title(title):
    c.execute("""SELECT * from article
                 WHERE article.title=?""", (title,))
    return c.fetchone()
def create_article(title, text, publicity, summary):
    c.execute("""INSERT INTO article
                 (title, text, publicity, creation_date)
                 VALUES (?, ?, ?, ?)""", (title, text, publicity, datetime.datetime.utcnow()))
    conn.commit()

def update_article(title, text, publicity, summary):
    c.execute("""UPDATE article
                 SET text=?, publicity=?
                 WHERE article.title=?""", (text, publicity, title))
    conn.commit()

def get_history_by_article(article_title):
    article_id = get_article_by_title(article_title).id
    c.execute("""SELECT * from article_history
                 WHERE article_history.article_id=?""", (article_id,))
    return c.fetchall()

def update_history(article_title, summary):
    user_id = get_cookie_user_id()
    article_id = get_article_by_title(article_title).id

    c.execute("""INSERT INTO article_history
                 (article_id, user_id, summary, time)
                 VALUES (?, ?, ?, ?)""", (article_id, user_id, summary, datetime.datetime.utcnow()))
    conn.commit()
# Util
###################################
# remove ending slash from requests
@hook('before_request')
def strip_path():
    request.environ['PATH_INFO'] = request.environ['PATH_INFO'].rstrip('/')

def setup():
    create_tables()
    # dev_setup(conn, c)

def get_cookie_user_id():
    user_id = request.get_cookie("auth", secret=COOKIE_SECRET)
    return user_id

# TODO: the permission system is messy and
# needs to be revamped
def can_access(article_title):
    article = get_article_by_title(article_title)
    user_id = get_cookie_user_id()
    user = get_user_by_id(user_id)

    if (article is not None and
        article.publicity == "private" and
        ((user is not None and user.id != 1) or (user is None))):
        return False

    return True

def dev_setup(conn, cursor):
    user_data = ("ben",
                 "ben@bendoan.me",
                 "admin",
                 create_hash_password("pass"),
                 datetime.datetime.utcnow())
    c.execute("""INSERT INTO user
                 (name, email, role, password, creation_date)
                 VALUES (?, ?, ?, ?, ?)""", user_data)

    article_data = ("Home",
                    "this is the home page",
                    "public",
                    datetime.datetime.utcnow())

    c.execute("""INSERT INTO article
                 (title, text, publicity, creation_date)
                 VALUES (?, ?, ?, ?)""", article_data)
    conn.commit()

# Run
###################################
setup()
if __name__ == '__main__':
    parser = argparse.ArgumentParser(description='starts the jabawiki server')
    parser.add_argument('--config', help='specifies the config file location (default: ./config.json)',
                            default="./config.json")
    args = parser.parse_args()

    run(host='0.0.0.0', port=PORT)

app = default_app()
