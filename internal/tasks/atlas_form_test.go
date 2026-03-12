package tasks

import (
	"log"
	"testing"
)

func TestSubmitAtlasForm(t *testing.T) {

	result, err := SubmitAtlasForm(1)

	if err != nil {
		t.Fatal(err)
	}

	log.Println("result:", result)
}
func TestCleanMail(t *testing.T) {

	CleanMail(1)
}
