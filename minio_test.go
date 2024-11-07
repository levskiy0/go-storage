package go_storage

import (
	"context"
	"testing"
)

func TestMinio(t *testing.T) {
	// play.min.io credentials

	minio, err := NewMinio(context.Background(), MinioConfig{
		Key:      "BDmRm19w1noJiPkP5BOt",
		Secret:   "9h8kCd151keCqVpuxSme71Ol95PpybwFxupjmySG",
		Bucket:   "example222",
		Endpoint: "https://play.min.io:9000/",
		Url:      "https://test.io",
		Ssl:      true,
	})

	if err != nil {
		t.Fail()
		return
	}

	FS().
		AddDriver("minio", minio).
		SetDefaultDisk("minio")

	err = FS().Put("test.txt", "test data")
	if err != nil {
		t.Fail()
		return
	}

	f, err := NewFile("../baboon.png")
	if err != nil {
		t.Fail()
		return
	}

	fn, err := FS().PutFile("baboon", f)
	if err != nil {
		t.Fail()
		return
	}

	_, err = FS().Get(fn)
	if err != nil {
		t.Fail()
		return
	}
}
