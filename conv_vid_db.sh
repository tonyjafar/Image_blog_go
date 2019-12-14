#!/bin/bash

./convert_vids.py

mysql -u root -p image_blog -e "UPDATE videos set name = REPLACE(name, '.MOV', '.mp4');"

mysql -u root -p image_blog -e "UPDATE videos set name = REPLACE(name, '.mov', '.mp4');"



