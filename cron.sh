#!/usr/bin/env bash

set -xe

cd /home/ekzyis/hnbot
sqlite3 hnbot.sqlite3 < hacker_news.csv.sql
venv/bin/python plot.py
mv hn_*.png plots/
rsync plots/* vps:/var/www/files/public/hn/

