package docker
import (
	"log"
	"os/exec"
	"fmt"
	"net/http"
	"io/ioutil"
	"encoding/json"
	"bytes"
	"os"
)

//todo: add support for volumes, ports/config. all settings must be the same to migrate
type Docker struct {
	Name string `json:"Name"`
	Image string `json:"Image"`
	Command string `json:"Command"`
}

type Tarball struct {
	Data []byte `json:"Data"`
	Container Docker `json:"Container"`
}

func (d *Docker) Create() string {
	if d.Name == "" || d.Name == nil {
		log.Fatalf("Container needs to be named")
	}
	if d.Image == "" || d.Image == nil {
		log.Fatalf("Image needs to be specified")
	}
	cmd := fmt.Sprintf(`create %s --name %s`, d.Image, d.Name)
	return dockerCommand(cmd)
}

func (d *Docker) RM() string {
	if d.Name == "" || d.Name == nil {
		log.Fatalf("Container needs to be named")
	}
	if d.Image == "" || d.Image == nil {
		log.Fatalf("Image needs to be specified")
	}
	cmd := fmt.Sprintf(`rm %s`, d.Name)
	return dockerCommand(cmd)
}

func (d *Docker) Start() string {
	if d.Name == "" || d.Name == nil {
		log.Fatalf("Container needs to be named")
	}
	if d.Image == "" || d.Image == nil {
		log.Fatalf("Image needs to be specified")
	}
	cmd := fmt.Sprintf(`start %s`, d.Name)
	return dockerCommand(cmd)
}

func (d *Docker) Stop() string {
	if d.Name == "" || d.Name == nil {
		log.Fatalf("Container needs to be named")
	}
	if d.Image == "" || d.Image == nil {
		log.Fatalf("Image needs to be specified")
	}
	cmd := fmt.Sprintf(`stop %s`, d.Name)
	return dockerCommand(cmd)
}

func (d *Docker) Run() string {
	if d.Name == "" || d.Name == nil {
		log.Fatalf("Container needs to be named")
	}
	if d.Image == "" || d.Image == nil {
		log.Fatalf("Image needs to be specified")
	}
	cmd := fmt.Sprintf(`run %s %s --name %s`, d.Image, d.Command, d.Name)
	return dockerCommand(cmd)
}

func (d *Docker) Logs() string {
	if d.Name == "" || d.Name == nil {
		log.Fatalf("Container needs to be named")
	}
	return dockerCommand("logs " + d.Name)
}

func (d *Docker) Checkpoint(imageDir string) string {
	if d.Name == "" || d.Name == nil {
		log.Fatalf("Container needs to be named")
	}
	cmd := fmt.Sprintf(`checkpoint --image-dir=%s %s`, imageDir, d.Name)
	return dockerCommand(cmd)
}

func (d *Docker) Restore(imageDir string) string {
	if d.Name == "" || d.Name == nil {
		log.Fatalf("Container needs to be named")
	}
	cmd := fmt.Sprintf(`restore --force=true --image-dir=%s %s`, imageDir, d.Name)
	out := dockerCommand(cmd)
	os.RemoveAll(imageDir)
	return dockerCommand(out)
}

func (d *Docker) Export(url string) string {
	imageDir := fmt.Sprintf("/tmp/checkpoint_%s", d.Name)
	d.Checkpoint(imageDir)
	tarPath := fmt.Sprintf("/tmp/checkpoint_%s.tar.gz", d.Name)
	cmdStr := fmt.Sprintf("tar czf %s %s", tarPath, imageDir)
	out, err := exec.Command("/bin/sh", "-c", cmdStr).Output()
	if err != nil {
		log.Fatalf("Error running tar command: %s, %s, %s", cmdStr, err.Error(), out)
	}
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
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/upload_container", url), body)
	if err != nil {
		log.Fatalf("Error generating request: %s", err.Error())
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatalf("Error sending request: %s", err.Error())
	}
	body, err = ioutil.ReadAll(bytes.NewReader(resp.Body))
	if err != nil {
		log.Fatalf("Error reading response from upload: %s", err.Error())
	}
	if resp.StatusCode != 200 {
		log.Fatalf("Upload not accepted: %s", resp.Body)
	}
	os.Remove(tarPath)
	os.RemoveAll(imageDir)
	return d.Name
}

func Import(url string, containerName string) *Docker {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/upload_container/%s", url, containerName), nil)
	if err != nil {
		log.Fatalf("Error generating request: %s", err.Error())
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatalf("Error sending request: %s", err.Error())
	}
	body, err := ioutil.ReadAll(bytes.NewReader(resp.Body))
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
	cmdStr := fmt.Sprintf("tar -xf %s -C %s", tarPath, imageDir)
	out, err := exec.Command("/bin/sh", "-c", cmdStr).Output()
	if err != nil {
		log.Fatalf("Error running untar command: %s, %s, %s", cmdStr, err.Error(), out)
	}
	os.Remove(tarPath)
	container := tarball.Container
	container.Restore(imageDir)
	return container
}


func dockerCommand(command string) string {
	cmdStr := fmt.Sprintf(`docker %s`, command)
	out, err := exec.Command("/bin/sh", "-c", cmdStr).Output()
	if err != nil {
		log.Fatalf("Got error running command: ", cmdStr)
	}
	return string(out)
}