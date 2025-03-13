package image

import (
	"bufio"
	"math/rand"
	"os"
	"time"

	"github.com/fadedpez/tucoramirez/pkg/entities"
)

// Service provides operations for game images
type Service struct {
	images []*entities.Image
	rng    *rand.Rand
}

// NewService creates a new image service
func NewService(imagePath string) (*Service, error) {
	// Read image URLs from file
	file, err := os.Open(imagePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var images []*entities.Image
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		url := scanner.Text()
		if url != "" {
			images = append(images, &entities.Image{URL: url})
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Create random number generator with time-based seed
	source := rand.NewSource(time.Now().UnixNano())
	rng := rand.New(source)
	
	return &Service{
		images: images,
		rng:    rng,
	}, nil
}

// GetRandomImage returns a random image from the collection
func (s *Service) GetRandomImage() *entities.Image {
	if len(s.images) == 0 {
		return &entities.Image{URL: ""} // Return empty image if none available
	}
	
	randomIndex := s.rng.Intn(len(s.images))
	return s.images[randomIndex]
}
