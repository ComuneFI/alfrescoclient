package test

import (
	"alfrescoclient"
	"bytes"
	"fmt"
	"strings"
	"testing"
)

type Example struct {
	A string `json:"cm:description"`
	B string `json:"cm:title"`
}

var client alfrescoclient.AlfrescoClient = alfrescoclient.AlfrescoClient{}
var username string = "yyyy"
var password string = "xxxx"

func TestClient(t *testing.T) {
	client.Init("alfresco-test.comune.intranet", 8080, username, password)
	list1, err3 := client.ListNodes("-my-", 0, 10, nil, nil)
	if err3 != nil {

		panic(err3.Error())
	}
	for _, a := range list1.Entries {

		_ = client.DeleteNode(a.Entry.Id)
	}
	node, err := client.CreateNode("-my-", "prova", Example{A: "a", B: "b"})
	if err != nil {
		panic(err.Error())
	}

	id := node.Id
	err = client.SaveContent(id, "prova", strings.NewReader("abcde"))
	if err != nil {
		panic(err.Error())
	}
	fmt.Println("post ok " + id)

	a, err1 := client.GetNodeContent(id)
	if err1 != nil {
		panic(err1.Error())
	}
	defer a.Close()
	fmt.Println("download ok")
	buf := new(bytes.Buffer)
	buf.ReadFrom(a)
	newStr := buf.String()
	if newStr != "abcde" {
		panic("content not expected")
	}
	err = client.SaveContent(id, "prova1", strings.NewReader("abcde1"))
	if err != nil {
		panic(err.Error())
	}
	a, err1 = client.GetNodeContent(id)
	if err1 != nil {
		panic(err.Error())
	}
	defer a.Close()
	fmt.Println("download ok")
	buf = new(bytes.Buffer)
	buf.ReadFrom(a)
	newStr = buf.String()
	if newStr != "abcde1" {
		panic("content not expected")
	}

	_, err2 := client.ListNodes(node.ParentId, 0, 10, nil, &map[string]alfrescoclient.SortType{
		"name": alfrescoclient.SortType("DESC"),
	})
	if err2 != nil {
		panic(err.Error())
	}

	node, err = client.GetNodeMetadata(id)
	if err != nil {
		panic(err.Error())
	}

	err = client.DeleteNode(id)
	if err != nil {
		panic(err.Error())
	}

}
