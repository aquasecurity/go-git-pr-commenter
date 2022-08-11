package mock

type Mock struct{}

func NewMock() *Mock {
	return &Mock{}
}

func (c *Mock) WriteMultiLineComment(_, _ string, _, _ int) error {
	return nil
}

func (c *Mock) WriteLineComment(_, _ string, _ int) error {
	return nil
}

func (c *Mock) RemovePreviousAquaComments(_ string) error {
	return nil
}
