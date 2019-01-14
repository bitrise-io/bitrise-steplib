package steplib

import (
	stepmanModels "github.com/bitrise-io/stepman/models"
)

type Version struct {
	ID          string
	StepID      string
	StepYMLPath string
	Raw         []byte
	StepModel   stepmanModels.StepModel
}

func (s Step) FindVersionByID(id string) Version {
	for _, version := range s.Versions {
		if version.ID == id {
			return version
		}
	}
	return Version{}
}
