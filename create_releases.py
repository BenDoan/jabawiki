#!/usr/bin/env python
"""
Builds jabawiki for all major platforms.

Usage: python create_release.py (version) [op_sys arch]
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

if len(sys.argv) <= 1:
    print("Incorrect usage")
    print(__doc__)
    sys.exit(1)

if len(sys.argv) > 1:
    version = sys.argv[1]

if len(sys.argv) == 4:
    targets = ((sys.argv[2], sys.argv[3]), )

print("Building base")

if os.path.isdir("releases"):
    shutil.rmtree("releases")

# Make releases dir
os.mkdir("releases")
os.chdir("releases")

# Clone project
perform.git("clone", "..", "basewiki")
os.chdir("basewiki")
shutil.rmtree(".git")

# Install js/css depdencies with bower
perform.bower("install")

os.chdir("..")

# Build release for each target
for op_sys, arch in targets:
    name = "{}_{}".format(op_sys, arch)
    print("Building", name)

    shutil.copytree("basewiki", "jabawiki")
    os.chdir("jabawiki")
    os.system("env GOOS={} GOARCH={} go build".format(op_sys, arch))

    os.chdir("..")

    # Special case: archive windows stuff in zips, otherwise use tar.gz
    if op_sys != "windows":
        archive_name = "jabawiki-{}-{}_{}.tar.gz".format(sys.argv[1], op_sys, arch)
        perform.tar("-czf", archive_name, "jabawiki")
    else:
        archive_name = "jabawiki-{}-{}_{}.zip".format(sys.argv[1], op_sys, arch)
        perform.zip("-r", archive_name, "jabawiki")

    shutil.rmtree("jabawiki")

# Clean up
shutil.rmtree("basewiki")
