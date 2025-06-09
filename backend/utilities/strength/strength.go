package strength

import "strings"

func CalculateStrength(passphrase string) (int, error) {
	lengthScore := getLengthScore(passphrase)
	characterSetScore := getCharacterSetScore(passphrase)
	return lengthScore + characterSetScore, nil
}

type lengthRange struct {
	min int
	max int
}

var lengthScores = map[lengthRange]int{
	{0, 5}:   -2,
	{6, 7}:   0,
	{8, 12}:  1,
	{13, 16}: 2,
	{17, 20}: 3,
}

func getLengthScore(passphrase string) int {
	length := len(passphrase)

	// Find the range that the length falls into
	for rangeIn, score := range lengthScores {
		if length >= rangeIn.min && length <= rangeIn.max {
			return score
		}
	}
	return 4
}

var characterSetScores = map[string]int{
	"abcdefghijklmnopqrstuvwxyz":         1,
	"ABCDEFGHIJKLMNOPQRSTUVWXYZ":         1,
	"0123456789":                         1,
	"!\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~": 1,
}

func getCharacterSetScore(passphrase string) int {
	score := 0
	for characterSet, characterSetScore := range characterSetScores {
		if strings.Contains(passphrase, characterSet) {
			score += characterSetScore
			continue
		}
	}
	return score
}
