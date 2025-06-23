package services

import (
	"math/rand"
	"passenger-go/backend/pipes"
	"passenger-go/backend/schemas"
	"passenger-go/backend/utilities/generator"
	"strings"

	"github.com/go-playground/validator/v10"
)

type GenerateService struct {
	validator *validator.Validate
}

func NewGenerateService() *GenerateService {
	return &GenerateService{
		validator: pipes.GetValidator(),
	}
}

func (service *GenerateService) Generate(
	length int,
) *schemas.ResponseGenerate {
	actualLength := min(max(length, 8), 4096)

	passphrase := make([]byte, actualLength)
	for index := range actualLength {
		passphrase[index] = generator.Chars[rand.Intn(len(generator.Chars))]
	}

	/**
	 * We do not use a while loop to ensure that
	 * the passphrase contains at least two characters
	 * from each set. Using a while loop, can result
	 * in an infinite loop if the random number
	 * generator keeps generating the same index.
	 */

	/* Generate 8 different indices */
	positions := make([]int, actualLength)
	for index := range actualLength {
		positions[index] = index
	}

	// Shuffle the positions array
	for index := len(positions) - 1; index > 0; index-- {
		j := rand.Intn(index + 1)
		positions[index], positions[j] = positions[j], positions[index]
	}

	// Take the first 8 positions
	selectedPositions := positions[:8]

	// Ensure that the passphrase contains at least two characters from each set
	characterSets := []string{
		generator.Lowers,
		generator.Uppers,
		generator.Numbers,
		generator.Specials,
	}

	for position := range 8 {
		setIndex := position % 4
		set := characterSets[setIndex]
		passphrase[selectedPositions[position]] = set[rand.Intn(len(set))]
	}

	return &schemas.ResponseGenerate{
		Generated: string(passphrase),
	}
}

func (service *GenerateService) Alternate(
	passphrase string,
) *schemas.ResponseAlternate {
	var output strings.Builder

	for _, character := range passphrase {
		lowerChar := strings.ToLower(string(character))

		if alternatives, exists := generator.ManipulateMap[lowerChar]; exists {
			randomIndex := rand.Intn(len(alternatives))
			output.WriteString(alternatives[randomIndex])
		} else {
			output.WriteString(lowerChar)
		}
	}

	return &schemas.ResponseAlternate{
		Alternative: output.String(),
	}
}
