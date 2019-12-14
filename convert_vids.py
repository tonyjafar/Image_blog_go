#!/usr/bin/python3

# use to convert videos to mp4 so firefox and chrom can plays the videos

import os
import subprocess

folder = 'data/videos/'

os.chdir(folder)

files = os.listdir(".")
for file in files:
    name, file_ext = os.path.splitext(file)
    if file_ext.lower() == '.mov':
        try:
            subprocess.call("ffmpeg -y -i %s -vcodec h264 -acodec aac -strict -2 %s.mp4 -async 1 -vsync 1" % (file, name), shell=True)
            print("%s is converted" % file)
            os.remove(file)
        except Exception as e:
            print(str(e))
            pass

