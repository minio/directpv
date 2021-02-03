package utils

import (
	"encoding/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/yaml"

	"github.com/golang/glog"

	"fmt"
)

func JSONifyAndLog(val interface{}) {
	jsonBytes, err := json.MarshalIndent(val, "", " ")
	if err != nil {
		return
	}
	glog.V(3).Infof(string(jsonBytes))
}

func BoolToCondition(val bool) metav1.ConditionStatus {
	if val {
		return metav1.ConditionTrue
	}
	return metav1.ConditionFalse
}

func LogYAML(obj interface{}) error {
	y, err := yaml.Marshal(obj)
	if err != nil {
		return err
	}
	PrintYaml(y)
	return nil
}

func PrintYaml(data []byte) {
	fmt.Print(string(data))
	fmt.Println()
	fmt.Println("---")
	fmt.Println()
}
