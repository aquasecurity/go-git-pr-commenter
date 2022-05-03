package mock

type Mock struct{}

func NewMock() *Mock {
	return &Mock{}
}

func (c *Mock) WriteMultiLineComment(file, comment, startLine, endLine string) error {
	return nil
}

func (c *Mock) WriteLineComment(file, comment, line string) error {
	return nil
}
