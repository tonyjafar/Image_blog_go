package main

import (
	"crypto/sha1"
	"fmt"
	"time"
)

type Image struct {
	ID          string
	Name        string
	Location    string
	Size        int64
	CreatedAt   time.Time
	Description string
}

func Save(image *Image) error {
	_, err := db.Exec(
		`
		REPLACE INTO image_blog.images
		(id, name, location, description, size, created_at)
		VALUES
		(?, ?, ?, ?, ?, ?)
		`,
		image.ID,
		image.Name,
		image.Location,
		image.Description,
		image.Size,
		image.CreatedAt,
	)
	return err
}

func Find(disc string) ([]Image, error) {
	rows, err := db.Query(
		`
		SELECT id, name, location, description, size, created_at
		from image_blog.images
		where description = ?
		`, disc,
	)
	if err != nil {
		return nil, err
	}
	allImages := []Image{}
	for rows.Next() {
		image := Image{}
		err := rows.Scan(
			&image.ID,
			&image.Name,
			&image.Location,
			&image.Description,
			&image.Size,
			&image.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		allImages = append(allImages, image)

	}
	return allImages, nil
}

func NewImage() *Image {
	h := sha1.New()
	return &Image{
		ID:        "img" + fmt.Sprintf("%x", h.Sum(nil)),
		CreatedAt: time.Now(),
	}
}
