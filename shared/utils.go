package shared
import (
	mesos "github.com/mesos/mesos-go/mesosproto"
	"errors"
	"fmt"
)


func GetValueFromLabels(labels *mesos.Labels, key string) (string, error) {
	for _, label := range labels.Labels {
		if *label.Key == key {
			return *label.Value, nil
		}
	}
	return nil, errors.New(fmt.Sprintf("KEY %s NOT FOUND IN TASK INFO! Here were the labels, if you want to see: %v", key, labels))
}

func CreateLabel(key string, value string) *mesos.Label{
	return &mesos.Label{
		Key: &key,
		Value: &value,
	}
}