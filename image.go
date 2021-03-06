package main

import (
	"time"
)

type Image struct {
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
		(name, location, description, size, created_at)
		VALUES
		(?, ?, ?, ?, ?)
		`,
		image.Name,
		image.Location,
		image.Description,
		image.Size,
		image.CreatedAt,
	)
	return err
}

type Video struct {
	Name        string
	Location    string
	Size        int64
	CreatedAt   time.Time
	Description string
}

func SaveV(video *Video) error {
	_, err := db.Exec(
		`
		REPLACE INTO image_blog.videos
		(name, location, description, size, created_at)
		VALUES
		(?, ?, ?, ?, ?)
		`,
		video.Name,
		video.Location,
		video.Description,
		video.Size,
		video.CreatedAt,
	)
	return err
}

/*
func Find(disc string) ([]Image, error) {
	rows, err := db.Query(
		`
		SELECT name, location, description, size, created_at
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
*/
