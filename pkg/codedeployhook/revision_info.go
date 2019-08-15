package codedeployhook

import (
	"fmt"
)

type RevisionInfo struct {
	Version   string                `json:"version"`
	Resources []map[string]Resource `json:"Resources"`
}

type Resource struct {
	Type       string     `json:"Type"`
	Properties Properties `json:"Properties"`
}

type Properties struct {
	Name           string `json:"Name"`
	Alias          string `json:"Alias"`
	CurrentVersion string `json:"CurrentVersion"`
	TargetVersion  string `json:"TargetVersion"`
}

var AwsAccountId = "unknown"
var AwsRegion = "unknown"

func (r RevisionInfo) functionArn() string {
	res := Resource{}
	for _, r := range r.Resources[0] {
		res = r
	}

	name := res.Properties.Name
	version := res.Properties.TargetVersion

	return fmt.Sprintf("arn:aws:lambda:%s:%s:function:%s:%s", AwsRegion, AwsAccountId, name, version)
}
