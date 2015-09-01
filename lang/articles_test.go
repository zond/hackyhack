package lang

import "testing"

func validate(t *testing.T, word, article string) {
	if Art(word).String() != article {
		t.Errorf("%v %v", article, word)
	}
}

func TestArticles(t *testing.T) {
	validate(t, "apple", "an")
	validate(t, "banana", "a")
	validate(t, "aardvark", "an")
	validate(t, "astronaut", "an")
	validate(t, "anonymous", "an")
}
