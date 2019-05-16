# Image_blog_go
## Project Structure
```.
├── conf.json
├── data
│   ├── thumb
│   └── videos
├── static
│   ├── css
│   │   └── plugins
│   ├── font-awesome
│   │   ├── css
│   │   ├── fonts
│   │   ├── less
│   │   └── scss
│   ├── fonts
│   └── js
│       └── plugins
│           ├── flot
│           └── morris
└── templates
```
## Users Table
| Column Name   | Data Type    | PK | NN | UQ | AI | Default |
| ------------- | ------------ | -- | -- | -- | -- | ------- |
| id            | INT(11)      | Y  | Y  | N  | Y  | N       |
| username      | VARCHAR(255) | N  | Y  | Y  | N  | N       | 
| password      | VARCHAR(255) | N  | Y  | N  | N  | N       |
| session       | VARCHAR(255) | N  | N  | N  | N  | NULL    |
| retry         | INT(11)      | N  | N  | N  | N  | 0       |
| last_activity | DATETIME     | N  | N  | N  | N  | NULL    |
| admin         | VARCHAR(45)  | N  | N  | N  | N  | no      |

### Create Initial Admin User
create enrcypted password using the [script](https://github.com/tonyjafar/go_examples/blob/master/crypt_check_pass.go)
then insert a new user in the DB :
```
INSERT INTO IMAGE_BLOG.USERS (username, password, admin) VALUES (<your_user>, <YOUR_ENCRYPTED_PASS>, "yes");
```
then you can add other users using the admin Page.

Or you can use the template.sql to import Sample DB with default admin user with password 'Admin1!'

## Images and Videos Tables
| Column Name   | Data Type    | PK | NN | UQ | AI |
| ------------- | ------------ | -- | -- | -- | -- |
| name          | VARCHAR(255) | Y  | Y  | N  | N  |
| location      | VARCHAR(255) | N  | Y  | N  | N  | 
| description   | TEXT         | N  | Y  | N  | N  |
| size          | INT(11)      | N  | Y  | N  | N  |
| created_at    | DATETIME     | N  | Y  | N  | N  |

## conf.json
```
{
    "username": "*******",
    "password": "*********",
    "ipaddress": "127.0.0.1",
    "port": "3306",
    "database": "image_blog"
    
}
```

## Steps to start the Server:

```
# Prerequist "mysql DB" and "GOlang"
$ go get github.com/graphql-go/graphql
$ go get github.com/go-sql-driver/mysql
$ go get github.com/op/go-logging
$ go get github.com/disintegration/imaging
$ go get github.com/satori/go.uuid
$ go get golang.org/x/crypto/bcrypt
$ git clone https://github.com/tonyjafar/Image_blog_go.git
$ cd Image_blog_go
$ mkdir -p data/thumb && mkdir data/videos && mkdir logs
$ mysql -u $USER -p < template.sql
$ touch conf.json && vim conf.json
$ go build
#Run Tests
$ go test -v
#Run Server
$ ./Image_blog_go
```
then you can access the server in browser http://localhost:8000

Alternatively you can run both mysql and server using [docker](https://github.com/tonyjafar/docker-compose-examples/tree/master/image_blog_docker)
