package generator

/**
 * Manipulate maps each character to similar-looking characters.
 * This means passphrase should be as powerful as possible while
 * still being easy to read and remember.
 */
var ManipulateMap = map[string][]string{
	"q": {"Q", "q"},
	"w": {"W", "m", "M", "w"},
	"e": {"E", "e"},
	"r": {"R", "r"},
	"t": {"T", "7", "t"},
	"y": {"Y", "h", "y"},
	"u": {"U", "u", "n"},
	"i": {"I", "1", "i"},
	"o": {"O", "0", "o"},
	"p": {"P", "p"},
	"a": {"A", "4", "@", "a"},
	"s": {"S", "$", "5", "s"},
	"d": {"D", "d"},
	"f": {"F", "f"},
	"g": {"G", "6", "9", "g"},
	"h": {"H", "y", "h"},
	"j": {"J", "j"},
	"k": {"K", "k"},
	"l": {"L", "l"},
	"z": {"Z", "2", "z"},
	"x": {"X", "x"},
	"c": {"C", "c"},
	"v": {"V", "v"},
	"b": {"B", "3", "8", "b"},
	"n": {"N", "n", "u"},
	"m": {"M", "W", "w", "m"},
	"$": {"S", "s", "5"},
	"@": {"A", "a"},
	"?": {"7"},
	"0": {"O", "o", "0"},
	"1": {"i", "1"},
	"2": {"Z", "z", "2"},
	"3": {"B", "3"},
	"4": {"A", "4"},
	"5": {"S", "s", "$"},
	"6": {"G", "6"},
	"7": {"7", "?", "T"},
	"8": {"B", "8"},
	"9": {"g", "9"},
}

const (
	Specials = "!@#$%^&*()_+-=[]{}|;:,.<>?/"
	Lowers   = "abcdefghijklmnopqrstuvyz"
	Uppers   = "ABCDEFGHIJKLMNOPQRSTUVYZ"
	Numbers  = "0123456789"
	Chars    = Lowers + Uppers + Numbers + Specials
)
