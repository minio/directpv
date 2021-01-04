package utils

import (
	"encoding/json"
	
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/golang/glog"
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
