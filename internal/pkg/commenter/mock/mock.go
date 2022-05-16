package mock

type Mock struct{}

func NewMock() *Mock {
	return &Mock{}
}

func (c *Mock) WriteMultiLineComment(file, comment string, startLine, endLine int) error {
	return nil
}

func (c *Mock) WriteLineComment(file, comment string, line int) error {
	return nil
}
