package backend

type validatable interface {
	Validate() error
}

// Validate reports value's problems as an invalid file at path.
func Validate(value validatable, path string) error {
	if err := value.Validate(); err != nil {
		return InvalidFile(path, err)
	}
	return nil
}
