#!/bin/sh

ip=$1

cd /srv/my_repos/Image_blog_go/
ssh ubuntu@$ip sudo systemctl stop blog
/srv/my_repos/Image_blog_go/convert_vids.py

mysql -u root -pXXX\! image_blog -e "UPDATE videos set name = REPLACE(name, '.MOV', '.mp4');"

mysql -u root -pXXX\! image_blog -e "UPDATE videos set name = REPLACE(name, '.mov', '.mp4');"

cd && mysqldump -u root -pXXX\! --databases image_blog > blog.sql

rsync -e "ssh -i /srv/Keys/XXXX.pem"  -r /srv/my_repos/Image_blog_go/  ubuntu@$ip:/home/ubuntu/my_blog/ -v

scp -i /srv/Keys/XXX.pem blog.sql /srv/my_repos/Image_blog_go/image_blog ubuntu@$ip:/home/ubuntu/my_blog/

ssh ubuntu@$ip mysql -u root -pXXX\! -e 'drop database image_blog;'

ssh ubuntu@$ip 'mysql -u root -pXXX\! < /home/ubuntu/my_blog/blog.sql'

ssh ubuntu@$ip sudo systemctl start blog
