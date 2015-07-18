#!/usr/bin/env python
"""
Builds jabawiki for all major platforms.

Usage: python create_release.py (version)
"""

import os
import shutil
import sys

import perform

targets = (
            ("linux", "386"),
            ("linux", "amd64"),
            ("linux", "arm"),
            ("freebsd", "386"),
            ("freebsd", "amd64"),
            ("darwin", "386"),
            ("darwin", "amd64"),
            ("windows", "386"),
            ("windows", "amd64")
        )

version = sys.argv[1]

print("Building base")

if os.path.isdir("releases"):
    shutil.rmtree("releases")

os.mkdir("releases")
os.chdir("releases")

perform.git("clone", "..", "basewiki")

os.chdir("basewiki")

shutil.rmtree(".git")
perform.bower("install")

os.chdir("..")

for op_sys, arch in targets:
    name = "{}_{}".format(op_sys, arch)
    print("Building", name)

    shutil.copytree("basewiki", "jabawiki")
    os.chdir("jabawiki")
    os.system("env GOOS={} GOARCH={} go build".format(op_sys, arch))

    os.chdir("..")

    if op_sys != "windows":
        archive_name = "jabawiki-{}-{}_{}.tar.gz".format(sys.argv[1], op_sys, arch)
        perform.tar("-czf", archive_name, "jabawiki")
    else:
        archive_name = "jabawiki-{}-{}_{}.zip".format(sys.argv[1], op_sys, arch)
        perform.zip("-r", archive_name, "jabawiki")

    shutil.rmtree("jabawiki")

shutil.rmtree("basewiki")
