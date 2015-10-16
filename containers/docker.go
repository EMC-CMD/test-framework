package docker
import (
	"log"
	"os/exec"
	"fmt"
)

//todo: add support for volumes, ports/config. all settings must be the same to migrate
type Docker struct {
	Name string `json:"Name"`
	Image string `json:"Image"`
	Command string `json:"Command"`
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
	return dockerCommand(cmd)
}


func dockerCommand(command string) string {
	cmdStr := fmt.Sprintf(`docker %s`, command)
	out, err := exec.Command("/bin/sh", "-c", cmdStr).Output()
	if err != nil {
		log.Fatalf("Got error running command: ", cmdStr)
	}
	return string(out)
}