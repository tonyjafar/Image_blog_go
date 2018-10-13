# Image_blog_go
## Project Structure
```.
├── assets
├── conf.json
├── data
│   └── thumb
├── funcs.go
├── Image_blog_go
├── image.go
├── main.go
├── README.md
├── routes.go
└── templates
    ├── footer.gohtml
    ├── header.gohtml
    ├── images.gohtml
    ├── index.gohtml
    ├── search.gohtml
    ├── signin.gohtml
    └── uplimage.gohtml
```
## User Table
id, username, password, sesion
, password created using golang.org/x/crypto/bcrypt
```
func GenerateFromPassword(password []byte, cost int) ([]byte, error) {
	p, err := newFromPassword(password, cost)
	if err != nil {
		return nil, err
	}
	return p.Hash(), nil
}
```
converting it to string

## Images and Videos Tables
name, location, description, size, created_at

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
