package main

import (
	"log"
	"io/ioutil"
	"net/http"
	"bytes"
	"fmt"
	"encoding/json"
	"os/exec"
	"os"
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

//uncomment to test download
func testUpload(){
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

func testDownload(){
	containerName := "foo"
	url := "http://192.168.0.15:3000"
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/download_container/%s", url, containerName), nil)
	if err != nil {
		log.Fatalf("Error generating request: %s", err.Error())
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatalf("Error sending request: %s", err.Error())
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Error reading response from upload: %s", err.Error())
	}
	var tarball Tarball
	err = json.Unmarshal(body, &tarball)
	if err != nil {
		log.Fatalf("Could not read json into tarball struct")
	}

	tarPath := fmt.Sprintf("/tmp/checkpoint_%s.tar.gz", containerName)
	err = ioutil.WriteFile(tarPath, tarball.Data, 0666)
	if err != nil {
		log.Fatalf("Could not write downloaded tarball to disk")
	}
	imageDir := fmt.Sprintf("/tmp/checkpoint_%s", containerName)
	cmdStr := fmt.Sprintf("tar -xf %s -C %s  --absolute-names", tarPath, imageDir)
	out, err := exec.Command("/bin/sh", "-c", cmdStr).Output()
	if err != nil {
		log.Fatalf("Error running untar command: %s, %s, %s", cmdStr, err.Error(), out)
	}
	os.Remove(tarPath)
}

//uncomment to test upload
func main(){
//	testUpload()
//	testDownload()
}