package steplib

type Step struct {
	ID       string
	Versions []Version
	Latest   Version
}
