package frontend

import (
	"fmt"
	"testing"
)

func TestIndex(t *testing.T) {
	b, err := Index(".")
	if err != nil {
		t.Fatalf("%v", err)
	}
	fmt.Printf("%s", b)
}
