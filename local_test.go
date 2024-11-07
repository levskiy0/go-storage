package filesystem

import (
	"testing"
	"time"
)

func TestLocal(t *testing.T) {
	FS().
		AddDriver("local", NewLocal(LocalConfig{
			Root: "./storage",
			Url:  "https://test.io",
		})).
		SetDefaultDisk("local")

	if FS().Put("test.txt", "test data") != nil {
		t.Fail()
	}

	time.Sleep(time.Second * 5)
}
