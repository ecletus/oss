package ftp_test

import (
	"github.com/goftp/server"
	"testing"
	"github.com/aghape/oss/ftp"
	"bytes"
	"io/ioutil"
)

var client *ftp.Client

var port = 7778

func init() {
	var err error

	client, err = ftp.New(ftp.Config{
		Hosts:    []string{"localhost:21"},
		User:     "test_user",
		Password: "test",
		Endpoint: "http://localhost/u/test_user/root/dir",
		RootDir:  "root/dir",
	})

	if err != nil {
		panic(err)
	}

}

type Auth struct {
	server.Auth
}

func (Auth) CheckPasswd(username string, password string) (bool, error) {
	return username == "user" && password == "pwd", nil
}

func TestAll(t *testing.T) {
	TestPath(t)
}

func TestPath(t *testing.T) {
	if p := client.Path("a/b"); p != "root/dir/a/b" {
		t.Log("t1")
		t.Fail()
	}
	if p := client.Path("/a/b/"); p != "root/dir/a/b" {
		t.Log("t2")
		t.Fail()
	}
}

func TestPut(t *testing.T) {
	buf := bytes.NewBufferString("d1")
	o, err := client.Put("b/a", buf)
	if err != nil {
		t.Error("#1: ", err)
		t.Fail()
	}
	if o.Path != "b/a" {
		t.Error("#2: path failed")
	}
}

func TestList(t *testing.T) {
	items, err := client.List("b")
	if err != nil {
		t.Error("#1: ", err)
		t.Fail()
	} else {
		if len(items) != 1 {
			t.Error("#2: have many items")
			t.Fail()
		}
	}
}

func TestGet(t *testing.T) {
	file, err := client.Get("b/a")
	if err != nil {
		t.Error("#1: ", err)
		t.Fail()
	}

	data, err := ioutil.ReadAll(file)
	if err != nil {
		t.Error("#2: ", err)
		t.Fail()
	}

	if string(data) != "d1" {
		t.Error("#3: invalid data")
		t.Fail()
	}
}
/*
func TestDelete(t *testing.T) {
	err := client.Delete("b/a")
	if err != nil {
		t.Error("#1: ", err)
		t.Fail()
	}
}*/


func TestEndpoint(t *testing.T) {
	ep := client.GetEndpoint()
	println( ep + "/b/a")
}