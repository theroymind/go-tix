package store

type validatable interface {
	Validate() error
}

func Validate(value validatable, path string) error {
	if err := value.Validate(); err != nil {
		return InvalidFile(path, err)
	}
	return nil
}
