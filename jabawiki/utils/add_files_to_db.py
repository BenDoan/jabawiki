#!/usr/bin/env python

import datetime
import os
import sqlite3
import sys

from os import path

def _main():
    _, db_file, article_dir = sys.argv

    conn = sqlite3.connect(db_file, detect_types=sqlite3.PARSE_DECLTYPES)

    c = conn.cursor()

    for fname in os.listdir(article_dir):
        with open(path.join(article_dir, fname)) as f:
            txt = "".join(f.readlines())
            article_data = (fname.split(".txt")[0], txt, "private", datetime.datetime.utcnow())
            c.execute("""INSERT INTO article
                         (title, text, publicity, creation_date)
                         VALUES (?, ?, ?, ?)""", article_data)
    conn.commit()

if __name__ == '__main__':
    _main()
