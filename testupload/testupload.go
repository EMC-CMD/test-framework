package main

import (
	"log"
	"io/ioutil"
"net/http"
	"bytes"
	"fmt"
	"encoding/json"
)

type Docker struct {
	Name string `json:"Name"`
	Image string `json:"Image"`
	Command string `json:"Command"`
}

type Tarball struct {
	Data []byte `json:"Data"`
	Container Docker `json:"Container"`
}

func main(){
	d := Docker{
		Name: "foo",
		Image: "busybox:latest",
		Command: `/bin/sh -c 'i=0; while true; do echo "%s: $i"; i=$(expr $i + 1); sleep 1; done'`,
	}
	url := "http://192.168.0.15:3000"

	tarPath := "/tmp/foo.tar.gz"
	data, err := ioutil.ReadFile(tarPath)
	if err != nil {
		log.Fatalf("Error reading tarball during export: %s", err.Error())
	}
	tarball := Tarball{
		Container: d,
		Data: data,
	}
	body, err := json.Marshal(tarball)
	if err != nil {
		log.Fatalf("Error marshalling tarball to json: %s", err.Error())
	}
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/upload_container", url), bytes.NewReader(body))
	if err != nil {
		log.Fatalf("Error generating request: %s", err.Error())
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatalf("Error sending request: %s", err.Error())
	}
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Error reading response from upload: %s", err.Error())
	}
	if resp.StatusCode != 200 {
		msg, _ := ioutil.ReadAll(resp.Body)
		log.Fatalf("Upload not accepted: HTTP %v: %s", resp.StatusCode, msg)
	}
}